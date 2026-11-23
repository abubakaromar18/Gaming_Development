--- The level script. Each wave is a function that runs inside the director
-- task and reads like a timeline: "stream eight drones in, 0.3 s apart" is a
-- loop with a wait in it. The director shows a banner, runs the wave, waits
-- for the field to clear, pauses, and moves on.

local Waves = {}

local SLOT_SPACING = 48

-- Formation slot `i` of a row of `n`, centered on the arena.
local function slot(config, row_y, i, n)
  local left = config.width / 2 - (n - 1) * SLOT_SPACING / 2
  return { left + (i - 1) * SLOT_SPACING, row_y }
end

-- Streams `count` drones in from one side, `interval` seconds apart, each
-- curving through a waypoint to its slot. Blocks until the last one spawned.
local function drone_stream(game, side, count, interval, row_y, dive_base)
  local sched, config = game.sched, game.config
  local from_x = side == "left" and -20 or config.width + 20
  local via_x = side == "left" and config.width * 0.25 or config.width * 0.75
  for i = 1, count do
    local index = side == "left" and i or (count - i + 1)
    game:spawn_enemy("drone", from_x, 60, "drone", {
      path = { { via_x, row_y + 120 }, slot(config, row_y, index, count) },
      dive_after = dive_base + game.rng:range(0, 4),
    })
    if i < count then
      sched:wait(interval)
    end
  end
end

local function gunner(game, side, slot_x, slot_y)
  local x = side == "left" and -30 or game.config.width + 30
  game:spawn_enemy("gunner", x, slot_y, "gunner", { slot = { slot_x, slot_y } })
end

Waves.list = {
  {
    name = "WAVE 1",
    run = function(game)
      drone_stream(game, "left", 8, 0.3, 100, 3)
    end,
  },
  {
    name = "WAVE 2",
    run = function(game)
      local sched, w = game.sched, game.config.width
      -- Two streams at once: all() runs them as parallel child tasks.
      sched:all({
        function()
          drone_stream(game, "left", 6, 0.3, 90, 4)
        end,
        function()
          drone_stream(game, "right", 6, 0.3, 135, 4)
        end,
      })
      sched:wait(1.0)
      gunner(game, "left", w * 0.3, 200)
      gunner(game, "right", w * 0.7, 200)
    end,
  },
  {
    name = "WAVE 3",
    run = function(game)
      local sched, w = game.sched, game.config.width
      gunner(game, "left", w * 0.2, 180)
      sched:wait(0.6)
      gunner(game, "right", w * 0.8, 180)
      sched:wait(0.6)
      gunner(game, "left", w * 0.5, 230)
      sched:wait(1.5)
      drone_stream(game, "right", 8, 0.25, 95, 2)
    end,
  },
  {
    name = "BOSS",
    run = function(game)
      game:spawn_enemy("boss", game.config.width / 2, -60, "boss")
    end,
  },
}

--- Returns the director's task function for a list of waves.
function Waves.director(game, waves)
  local sched, world = game.sched, game.world
  local function field_clear()
    return world:enemy_count() == 0
  end
  return function()
    for i, wave in ipairs(waves) do
      game.wave = i
      sched:join(game:announce(wave.name, 1.0))
      wave.run(game)
      sched:wait_until(field_clear)
      sched:wait(game.config.wave_pause)
    end
    game:victory()
  end
end

return Waves
