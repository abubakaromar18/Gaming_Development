--- Enemy behaviors, each written as one straight-line coroutine script.
--
-- Every behavior runs as the enemy's own task (see Game:spawn_enemy). When
-- the enemy dies, the game cancels that task and every child it spawned, so a
-- script never has to check "am I still alive?" between steps. The same code
-- as a state machine would need an explicit state enum, a timer per state, and
-- transition bookkeeping for each of the steps below.

local Behaviors = {}

--- Drone: swoop in along a path to a formation slot, sway in formation for a
-- while, then dive at the player and leave the screen.
-- opts: path = list of {x, y} waypoints ending at the slot; dive_after = s.
function Behaviors.drone(game, enemy, opts)
  local sched, world = game.sched, game.world
  for i, point in ipairs(opts.path) do
    local last = i == #opts.path
    sched:tween(enemy, last and 0.5 or 0.35, { x = point[1], y = point[2] }, last and "back_out" or "linear")
  end

  -- race(): sway until the dive timer wins, then the sway task is cancelled.
  local slot_x = enemy.x
  sched:race({
    function()
      while true do
        sched:tween(enemy, 0.9, { x = slot_x + 14 }, "sine_in_out")
        sched:tween(enemy, 0.9, { x = slot_x - 14 }, "sine_in_out")
      end
    end,
    function()
      sched:wait(opts.dive_after)
    end,
  })

  -- Telegraph, then dive: x and y tween in parallel with different curves, so
  -- the drone accelerates downward while drifting toward where you were.
  sched:tween(enemy, 0.15, { flash = 1 })
  sched:tween(enemy, 0.15, { flash = 0 })
  local target_x = world.player.x
  sched:all({
    function()
      sched:tween(enemy, 1.4, { x = target_x }, "sine_in_out")
    end,
    function()
      sched:tween(enemy, 1.4, { y = game.config.height + 40 }, "quad_in")
    end,
  })
end

--- Gunner: slide in from the side, then forever: wait a random beat, glow to
-- telegraph, fire an aimed three-bullet spread, and strafe somewhere new.
-- opts: slot = {x, y}.
function Behaviors.gunner(game, enemy, opts)
  local sched, world, rng = game.sched, game.world, game.rng
  enemy.shielded = true
  sched:tween(enemy, 0.8, { x = opts.slot[1], y = opts.slot[2] }, "back_out")
  enemy.shielded = false
  local margin = 40
  while true do
    sched:wait(rng:range(0.7, 1.4))
    sched:tween(enemy, 0.25, { flash = 1 }, "quad_in")
    local aim = world:aim(enemy.x, enemy.y)
    for k = -1, 1 do
      world:fire(enemy.x, enemy.y, aim + k * 0.22)
    end
    sched:tween(enemy, 0.2, { flash = 0 })
    local x = rng:range(margin, game.config.width - margin)
    sched:tween(enemy, 0.9, { x = x }, "quad_in_out")
  end
end

local function ring(world, enemy, count, offset, speed)
  for k = 0, count - 1 do
    world:fire(enemy.x, enemy.y, offset + k * (2 * math.pi / count), speed)
  end
end

--- Boss: a multi-phase fight. Each phase is a race between an attack loop and
-- a health check; when health drops past the threshold, the attack loop and
-- all of its timers are cancelled together and the next phase starts.
function Behaviors.boss(game, enemy)
  local sched, world, config = game.sched, game.world, game.config
  local cx = config.width / 2
  enemy.shielded = true
  sched:tween(enemy, 2.0, { y = 130 }, "cubic_out")
  enemy.shielded = false

  local function health_below(fraction)
    return function()
      sched:wait_until(function()
        return enemy.hp <= enemy.max_hp * fraction
      end)
    end
  end

  -- Phase 1: slow sway, rotating rings of bullets on a fixed timer.
  sched:race({
    function()
      sched:all({
        function()
          while true do
            sched:tween(enemy, 1.6, { x = cx + 110 }, "sine_in_out")
            sched:tween(enemy, 1.6, { x = cx - 110 }, "sine_in_out")
          end
        end,
        function()
          local offset = 0
          while true do
            sched:wait(1.1)
            ring(world, enemy, 12, offset, 150)
            offset = offset + 0.13
          end
        end,
      })
    end,
    health_below(0.5),
  })

  -- Transition: shield up, return to center, flash three times.
  enemy.shielded = true
  game:announce("ENRAGED", 0.9)
  sched:tween(enemy, 0.6, { x = cx }, "quad_out")
  for _ = 1, 3 do
    sched:tween(enemy, 0.12, { flash = 1 })
    sched:tween(enemy, 0.12, { flash = 0 })
  end
  enemy.shielded = false

  -- Phase 2: a spiral from a fast repeating timer plus aimed bursts. No exit
  -- condition is needed: when the boss dies, cancellation ends this script.
  local spin = 0
  game.sched:every(0.09, function()
    spin = spin + 0.37
    world:fire(enemy.x, enemy.y, spin, 170)
    world:fire(enemy.x, enemy.y, spin + math.pi, 170)
  end, nil, "boss spiral")
  while true do
    sched:wait(1.6)
    sched:tween(enemy, 0.3, { flash = 1 }, "quad_in")
    local aim = world:aim(enemy.x, enemy.y)
    for k = -2, 2 do
      world:fire(enemy.x, enemy.y, aim + k * 0.12, 260)
    end
    sched:tween(enemy, 0.2, { flash = 0 })
  end
end

return Behaviors
