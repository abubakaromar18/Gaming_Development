--- Game configuration with defaults and validation.
--
-- Everything tunable lives here so states and rules read named values instead
-- of magic numbers, and a bad override fails at startup, not mid-game.

local Config = {}

Config.DEFAULTS = {
  cols = 24,             -- board width in cells
  rows = 18,             -- board height in cells
  start_length = 4,      -- initial snake length (cells)
  base_tick = 0.14,      -- seconds per move at score 0
  min_tick = 0.06,       -- fastest the snake will ever move
  speedup = 0.97,        -- tick multiplier per food eaten
  max_steps_per_update = 4, -- cap catch-up after a long frame
  turn_buffer = 2,       -- turns queued between moves (fast double-taps)
  wrap = false,          -- true: leave one edge, enter the opposite one
  countdown = 3,         -- seconds of "3, 2, 1" before play (re)starts
}

local function check(cond, message)
  if not cond then
    error("invalid config: " .. message, 3)
  end
end

local function is_int(v)
  return type(v) == "number" and v == math.floor(v)
end

--- Merge overrides onto the defaults and validate the result.
function Config.build(overrides)
  overrides = overrides or {}
  local cfg = {}
  for key, value in pairs(Config.DEFAULTS) do
    cfg[key] = value
  end
  for key, value in pairs(overrides) do
    check(Config.DEFAULTS[key] ~= nil, "unknown key '" .. tostring(key) .. "'")
    check(type(value) == type(Config.DEFAULTS[key]),
      "'" .. key .. "' must be a " .. type(Config.DEFAULTS[key]))
    cfg[key] = value
  end

  check(is_int(cfg.cols) and cfg.cols >= 5, "cols must be an integer >= 5")
  check(is_int(cfg.rows) and cfg.rows >= 5, "rows must be an integer >= 5")
  check(is_int(cfg.start_length) and cfg.start_length >= 1
    and cfg.start_length <= math.floor(cfg.cols / 2),
    "start_length must fit in half the board width")
  check(cfg.base_tick > 0 and cfg.min_tick > 0 and cfg.min_tick <= cfg.base_tick,
    "need 0 < min_tick <= base_tick")
  check(cfg.speedup > 0 and cfg.speedup <= 1, "speedup must be in (0, 1]")
  check(is_int(cfg.max_steps_per_update) and cfg.max_steps_per_update >= 1,
    "max_steps_per_update must be a positive integer")
  check(is_int(cfg.turn_buffer) and cfg.turn_buffer >= 1,
    "turn_buffer must be a positive integer")
  check(cfg.countdown >= 0, "countdown must be >= 0")
  return cfg
end

return Config
