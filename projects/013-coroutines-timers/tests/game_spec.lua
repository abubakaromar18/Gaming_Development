local Game = require("core.game")
local World = require("core.world")
local Config = require("core.config")
local Rng = require("core.rng")
local Waves = require("core.waves")

local IDLE = { move_x = 0, move_y = 0, fire = false }

local function advance(game, seconds, dt, controls)
  dt = dt or 1 / 60
  for _ = 1, math.floor(seconds / dt + 0.5) do
    game:update(dt, controls or IDLE)
  end
end

-- A wave list that spawns nothing, so tests can place enemies by hand.
local function empty_waves()
  return {
    {
      name = "TEST",
      run = function(game)
        game.sched:wait_until(function()
          return game.finish_test_wave
        end)
      end,
    },
  }
end

-- Steers under the lowest enemy and holds fire; stays invulnerable.
local function autopilot(game)
  local target
  for _, e in ipairs(game.world.enemies) do
    if not target or e.y > target.y then
      target = e
    end
  end
  local px, mx = game.world.player.x, 0
  if target and target.x > px + 4 then
    mx = 1
  elseif target and target.x < px - 4 then
    mx = -1
  end
  return { move_x = mx, move_y = 0, fire = true }
end

describe("World", function()
  local function world()
    local config = Config.build()
    return World.new(config, Rng.new(1)), config
  end

  it("fires 1 + floor(t / interval) shots while fire is held, at any frame rate", function()
    for _, dt in ipairs({ 1 / 30, 1 / 60, 1 / 144 }) do
      local w, config = world()
      local shots = 0
      for _ = 1, math.floor(1 / dt + 0.5) do
        local before = #w.bullets
        w:update(dt, { fire = true })
        shots = shots + (#w.bullets - before)
        w.bullets = {} -- count only, ignore culling
      end
      assert.equal(1 + math.floor(1 / config.fire_interval), shots, "dt = " .. dt)
    end
  end)

  it("keeps the player inside its movement zone and normalizes diagonals", function()
    local w, config = world()
    for _ = 1, 600 do
      w:update(1 / 60, { move_x = -1, move_y = -1 })
    end
    assert.equal(w.player.r, w.player.x)
    assert.equal(config.height * 0.45, w.player.y)
    local p = w.player
    p.x, p.y = 200, 500
    w:update(0.1, { move_x = 1, move_y = 1 })
    local moved = math.sqrt((p.x - 200) ^ 2 + (p.y - 500) ^ 2)
    assert.is_near(config.player_speed * 0.1, moved, 1e-9)
  end)

  it("damages enemies, ignores shielded ones, and reports kills", function()
    local w = world()
    local gunner = w:spawn_enemy("gunner", 100, 100)
    gunner.shielded = true
    w.bullets = { { x = 100, y = 100, vx = 0, vy = 0, r = 3 } }
    assert.same({}, w:update(0, IDLE))
    assert.equal(4, gunner.hp)
    gunner.shielded = false
    local events
    for _ = 1, 4 do
      w.bullets = { { x = 100, y = 100, vx = 0, vy = 0, r = 3 } }
      events = w:update(0, IDLE)
    end
    assert.equal(1, #events)
    assert.equal("enemy_killed", events[1].kind)
    assert.is_false(gunner.alive)
    assert.equal(0, w:enemy_count())
  end)

  it("reports a player hit from bullets unless invulnerable", function()
    local w = world()
    local p = w.player
    w:fire(p.x, p.y, 0, 0)
    p.invulnerable = true
    assert.same({}, w:update(0, IDLE))
    p.invulnerable = false
    local events = w:update(0, IDLE)
    assert.equal("player_hit", events[1].kind)
    assert.equal(0, #w.enemy_bullets)
  end)

  it("aims at the player and culls bullets that leave the arena", function()
    local w = world()
    local p = w.player
    local angle = w:aim(p.x, p.y - 100)
    assert.is_near(math.pi / 2, angle, 1e-12) -- straight down
    w:fire(10, 10, -math.pi / 2, 500)
    w:update(1, IDLE)
    assert.equal(0, #w.enemy_bullets)
  end)

  it("expires particles after their lifetime", function()
    local w = world()
    w:burst(50, 50, 5, "spark")
    w:update(World.PARTICLE_LIFE / 2, IDLE)
    assert.equal(5, #w.particles)
    w:update(World.PARTICLE_LIFE, IDLE)
    assert.equal(0, #w.particles)
  end)
end)

describe("Behaviors", function()
  it("a drone flies its path to the slot, sways, dives and is removed", function()
    local game = Game.new({ seed = 3, waves = empty_waves() })
    local drone = game:spawn_enemy("drone", 0, 0, "drone", {
      path = { { 100, 100 }, { 200, 120 } },
      dive_after = 1,
    })
    advance(game, 0.85) -- 0.35 + 0.5 s of entry tweens
    assert.is_near(200, drone.x, 1e-6)
    assert.is_near(120, drone.y, 1e-6)
    game.world.player.invulnerable = true
    advance(game, 3.5)
    assert.is_false(drone.alive)
    assert.equal(0, game.world:enemy_count())
    assert.equal("dead", drone.task.status)
    assert.equal(0, game.score) -- escaping isn't a kill
  end)

  it("a gunner is shielded on entry, then fires aimed spreads", function()
    local game = Game.new({ seed = 4, waves = empty_waves() })
    game.world.player.invulnerable = true
    local gunner = game:spawn_enemy("gunner", -30, 200, "gunner", { slot = { 150, 200 } })
    assert.is_true(gunner.shielded)
    advance(game, 0.8)
    assert.is_false(gunner.shielded)
    local fired = false
    for _ = 1, 180 do
      game:update(1 / 60, IDLE)
      fired = fired or #game.world.enemy_bullets >= 3
    end
    assert.is_true(fired)
  end)

  it("the boss switches phase when its health drops below half", function()
    local game = Game.new({ seed = 5, waves = empty_waves() })
    game.world.player.invulnerable = true
    local boss = game:spawn_enemy("boss", 240, -60, "boss")
    advance(game, 2.0)
    assert.is_near(130, boss.y, 1e-6)
    assert.is_false(boss.shielded)
    boss.hp = boss.max_hp / 2
    advance(game, 0.1)
    assert.is_true(boss.shielded)
    assert.equal("ENRAGED", game.banner.text)
    advance(game, 1.5)
    assert.is_false(boss.shielded)
    -- Phase 2's spiral timer is a child of the boss task.
    local names = table.concat(game.sched:names(), ",")
    assert.matches("boss spiral", names)
    boss.hp = 1
    game.world.bullets = { { x = boss.x, y = boss.y, vx = 0, vy = 0, r = 3 } }
    game:update(1 / 60, IDLE)
    assert.is_false(boss.alive)
    advance(game, 1)
    assert.is_nil(table.concat(game.sched:names(), ","):find("boss"))
  end)
end)

describe("Game", function()
  it("a kill scores, cancels the enemy's script and floats a popup", function()
    local game = Game.new({ seed = 6, waves = empty_waves() })
    local drone = game:spawn_enemy("drone", 100, 100, "drone", { path = { { 100, 100 } }, dive_after = 9 })
    game.world.bullets = { { x = 100, y = 100, vx = 0, vy = 0, r = 3 } }
    game:update(1 / 60, IDLE)
    assert.equal(100, game.score)
    assert.equal("cancelled", drone.task.status)
    assert.equal(1, #game.popups)
    assert.equal("+100", game.popups[1].text)
    advance(game, 1)
    assert.equal(0, #game.popups)
  end)

  it("chains quick kills into a score multiplier that resets after a pause", function()
    local game = Game.new({ seed = 7, waves = empty_waves() })
    local function kill_one()
      local d = game:spawn_enemy("drone", 50, 50, "drone", { path = { { 50, 50 } }, dive_after = 9 })
      game.world.bullets = { { x = d.x, y = d.y, vx = 0, vy = 0, r = 3 } }
      game:update(1 / 60, IDLE)
      advance(game, 0.2)
    end
    for _ = 1, 4 do
      kill_one()
    end
    assert.equal(4, game.combo)
    assert.equal(2, game:multiplier())
    kill_one()
    assert.equal(4 * 100 + 200, game.score)
    advance(game, Game.COMBO_WINDOW + 0.1)
    assert.equal(0, game.combo)
    assert.equal(1, game:multiplier())
  end)

  it("respawns with a locked fly-in and blinking invulnerability", function()
    local game = Game.new({ seed = 8, waves = empty_waves() })
    local p = game.world.player
    game.world:fire(p.x, p.y, 0, 0)
    game:update(1 / 60, IDLE)
    assert.equal(game.config.lives - 1, game.lives)
    assert.is_true(p.invulnerable)
    assert.is_true(p.locked)
    advance(game, 0.6)
    assert.is_false(p.locked)
    assert.is_true(p.invulnerable)
    advance(game, game.config.invulnerable_time + 0.1)
    assert.is_false(p.invulnerable)
    assert.is_true(p.visible)
  end)

  it("game over cancels the director, which clears every enemy", function()
    local game = Game.new({ seed = 9, config = { lives = 1 } })
    advance(game, 3) -- wave 1 drones are streaming in
    assert.is_true(game.world:enemy_count() > 0)
    local p = game.world.player
    game.world:fire(p.x, p.y, 0, 0)
    game:update(1 / 60, IDLE)
    assert.equal("game_over", game.state)
    assert.equal("cancelled", game.director.status)
    assert.equal(0, game.world:enemy_count())
    advance(game, 2)
    assert.equal("GAME OVER", game.banner.text)
    assert.is_near(1, game.banner.alpha, 1e-9)
  end)

  it("shows each wave's banner before spawning it", function()
    local game = Game.new({ seed = 10 })
    advance(game, 0.5)
    assert.equal("WAVE 1", game.banner.text)
    assert.equal(0, game.world:enemy_count())
    advance(game, 1.5)
    assert.is_true(game.world:enemy_count() > 0)
  end)

  it("an invulnerable autopilot clears every wave and wins", function()
    local game = Game.new({ seed = 11 })
    local seconds = 0
    while game.state == "playing" and seconds < 180 do
      game.world.player.invulnerable = true
      game:update(1 / 60, autopilot(game))
      seconds = seconds + 1 / 60
    end
    assert.equal("victory", game.state)
    assert.equal(#Waves.list, game.wave)
    assert.is_true(game.score > 0)
  end)

  it("replays identically from the same seed and inputs", function()
    local function play(seed)
      local game = Game.new({ seed = seed })
      for _ = 1, 60 * 40 do
        game:update(1 / 60, autopilot(game))
      end
      return game.score, game.lives, game.wave, #game.world.enemy_bullets
    end
    assert.same({ play(12) }, { play(12) })
  end)

  it("scripted motion is identical at 30 and 120 FPS", function()
    local function snapshot(dt)
      local game = Game.new({ seed = 13 })
      game.world.player.invulnerable = true
      advance(game, 4, dt)
      local out = {}
      for i, e in ipairs(game.world.enemies) do
        out[i] = { e.x, e.y }
      end
      return out
    end
    local slow, fast = snapshot(1 / 30), snapshot(1 / 120)
    assert.equal(#slow, #fast)
    assert.is_true(#slow > 0)
    for i = 1, #slow do
      assert.is_near(slow[i][1], fast[i][1], 1e-6)
      assert.is_near(slow[i][2], fast[i][2], 1e-6)
    end
  end)

  it("validates construction and dt", function()
    assert.has_error(function()
      Game.new({})
    end)
    local game = Game.new({ seed = 1 })
    assert.has_error(function()
      game:update(-0.1, IDLE)
    end)
  end)
end)
