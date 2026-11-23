--- Composition root: wires config, RNG, scheduler, world and the wave
-- director together, and turns world events into scripted reactions (score
-- popups, respawn sequences, game over).
--
-- Update order each frame: scripts first (they place enemies and fire), then
-- physics and collisions, then reactions to this frame's events.

local Config = require("core.config")
local Rng = require("core.rng")
local Scheduler = require("core.scheduler")
local World = require("core.world")
local Behaviors = require("core.behaviors")
local Waves = require("core.waves")

local Game = {}
Game.__index = Game

local NO_INPUT = { move_x = 0, move_y = 0, fire = false }
local COMBO_WINDOW = 1.0 -- seconds allowed between kills to keep a chain
local COMBO_STEP = 4 -- kills per extra multiplier
local COMBO_MAX_MULTIPLIER = 4

-- Kill chains as a script: wait for a kill, then keep racing "another kill"
-- against "the window runs out". Signals keep the world unaware of combos.
local function combo_tracker(self)
  local sched = self.sched
  return function()
    while true do
      sched:wait_signal("enemy_killed")
      self.combo = 1
      while sched:race({
        function()
          sched:wait_signal("enemy_killed")
        end,
        function()
          sched:wait(COMBO_WINDOW)
        end,
      }) == 1 do
        self.combo = self.combo + 1
      end
      self.combo = 0
    end
  end
end

--- opts: seed (number, required), config (overrides), waves (list, defaults
-- to Waves.list; tests pass shorter ones).
function Game.new(opts)
  opts = opts or {}
  if type(opts.seed) ~= "number" then
    error("Game.new needs a numeric seed", 2)
  end
  local config = Config.build(opts.config)
  local rng = Rng.new(opts.seed)
  local self = setmetatable({
    config = config,
    rng = rng,
    sched = Scheduler.new(),
    world = World.new(config, rng),
    score = 0,
    lives = config.lives,
    state = "playing", -- "playing" | "game_over" | "victory"
    wave = 0,
    wave_count = #(opts.waves or Waves.list),
    banner = { text = "", alpha = 0, y = 0 },
    popups = {},
    combo = 0,
  }, Game)
  self.sched:spawn(combo_tracker(self), "combo")
  self.director = self.sched:spawn(Waves.director(self, opts.waves or Waves.list), "director")
  return self
end

--- Creates an enemy and runs `behavior` as its task. The task removes the
-- enemy when it ends for any reason (finished its path, or cancelled because
-- it died), via defer, so no exit path can leak an enemy.
function Game:spawn_enemy(kind, x, y, behavior, behavior_opts)
  local world, sched = self.world, self.sched
  local enemy = world:spawn_enemy(kind, x, y)
  enemy.task = sched:spawn(function()
    sched:defer(function()
      world:remove_enemy(enemy)
    end)
    Behaviors[behavior or kind](self, enemy, behavior_opts or {})
  end, kind)
  return enemy
end

--- Shows a banner: drop in with an overshoot, hold, fade out. Returns the
-- task so callers can join it ("show the banner, then start the wave").
function Game:announce(text, hold)
  local sched, banner = self.sched, self.banner
  if self.banner_task then
    self.banner_task:cancel()
  end
  local rest_y = self.config.height * 0.38
  self.banner_task = sched:spawn(function()
    sched:defer(function()
      banner.alpha = 0
    end)
    banner.text, banner.alpha, banner.y = text, 0, rest_y - 50
    sched:tween(banner, 0.45, { alpha = 1, y = rest_y }, "back_out")
    sched:wait(hold)
    sched:tween(banner, 0.35, { alpha = 0, y = rest_y + 20 }, "quad_in")
  end, "banner")
  return self.banner_task
end

-- Floating "+score" text: rise and fade are two overlapping tweens.
local function score_popup(self, enemy, points, multiplier)
  local sched, popups = self.sched, self.popups
  local text = "+" .. points .. (multiplier > 1 and (" x" .. multiplier) or "")
  local popup = { text = text, x = enemy.x, y = enemy.y, alpha = 1 }
  popups[#popups + 1] = popup
  sched:spawn(function()
    sched:defer(function()
      for i = #popups, 1, -1 do
        if popups[i] == popup then
          table.remove(popups, i)
        end
      end
    end)
    sched:all({
      function()
        sched:tween(popup, 0.8, { y = popup.y - 36 }, "cubic_out")
      end,
      function()
        sched:wait(0.35)
        sched:tween(popup, 0.45, { alpha = 0 })
      end,
    })
  end, "popup")
end

-- Respawn: fly back in from below with controls locked, then blink while
-- invulnerable. Every flag is restored in defer, even if cancelled.
local function respawn(self)
  local sched, player, config = self.sched, self.world.player, self.config
  self.respawn_task = sched:spawn(function()
    sched:defer(function()
      player.invulnerable, player.visible, player.locked = false, true, false
    end)
    player.invulnerable, player.locked = true, true
    player.x, player.y = config.width / 2, config.height + 30
    sched:tween(player, 0.6, { y = config.height - 60 }, "back_out")
    player.locked = false
    local blinks = math.floor(config.invulnerable_time / 0.1)
    for _ = 1, blinks do
      player.visible = not player.visible
      sched:wait(0.1)
    end
  end, "respawn")
end

function Game:game_over()
  self.state = "game_over"
  self.world.player.alive = false
  -- Every enemy task is a child of the director, so cancelling it clears the
  -- field and stops every pending spawn timer in one call.
  for _, enemy in ipairs(self.world.enemies) do
    self.world:burst(enemy.x, enemy.y, 10, "explosion")
  end
  self.director:cancel()
  self.world.enemy_bullets = {}
  self:announce("GAME OVER", 1e9)
end

function Game:victory()
  self.state = "victory"
  self:announce("VICTORY", 1e9)
end

--- Score multiplier for the next kill, from the current kill chain.
function Game:multiplier()
  return math.min(COMBO_MAX_MULTIPLIER, 1 + math.floor(self.combo / COMBO_STEP))
end

local function handle(self, event)
  if event.kind == "enemy_killed" then
    local enemy = event.enemy
    local multiplier = self:multiplier()
    local points = enemy.score * multiplier
    self.score = self.score + points
    enemy.task:cancel()
    score_popup(self, enemy, points, multiplier)
    self.sched:emit("enemy_killed", enemy)
  elseif event.kind == "player_hit" and self.state == "playing" then
    self.lives = self.lives - 1
    if self.lives <= 0 then
      self:game_over()
    else
      self.world.enemy_bullets = {} -- a moment of mercy on respawn
      respawn(self)
    end
  end
end

--- Advances the whole game by dt seconds. `controls` are ignored once the
-- run has ended.
function Game:update(dt, controls)
  if type(dt) ~= "number" or dt < 0 then
    error("dt must be a non-negative number", 2)
  end
  if self.state ~= "playing" or self.world.player.locked then
    controls = NO_INPUT
  end
  self.sched:update(dt)
  local events = self.world:update(dt, controls or NO_INPUT)
  for _, event in ipairs(events) do
    handle(self, event)
  end
end

Game.COMBO_WINDOW = COMBO_WINDOW

return Game
