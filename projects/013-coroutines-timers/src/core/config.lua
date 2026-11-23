--- Tuning values and their validation. Game.new merges overrides into these
-- defaults, so tests can shrink or speed things up without touching code.

local Config = {}

Config.defaults = {
  width = 480,
  height = 640,
  lives = 3,
  player_speed = 270, -- px/s
  player_radius = 9,
  fire_interval = 0.11, -- seconds between shots while fire is held
  bullet_speed = 640,
  enemy_bullet_speed = 210,
  invulnerable_time = 2.0, -- after being hit
  wave_pause = 1.0, -- seconds between a cleared wave and the next banner
}

local POSITIVE = {
  "width",
  "height",
  "player_speed",
  "player_radius",
  "fire_interval",
  "bullet_speed",
  "enemy_bullet_speed",
  "invulnerable_time",
  "wave_pause",
}

function Config.build(overrides)
  overrides = overrides or {}
  local config = {}
  for key, value in pairs(Config.defaults) do
    config[key] = value
  end
  for key, value in pairs(overrides) do
    if Config.defaults[key] == nil then
      error("unknown config key '" .. tostring(key) .. "'", 2)
    end
    config[key] = value
  end
  for _, key in ipairs(POSITIVE) do
    local value = config[key]
    if type(value) ~= "number" or not (value > 0) then -- rejects NaN too
      error("config." .. key .. " must be a positive number", 2)
    end
  end
  if type(config.lives) ~= "number" or config.lives < 1 or config.lives % 1 ~= 0 then
    error("config.lives must be a positive integer", 2)
  end
  return config
end

return Config
