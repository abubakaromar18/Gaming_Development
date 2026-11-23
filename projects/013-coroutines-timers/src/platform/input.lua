--- Keyboard -> abstract controls. The core only ever sees
-- { move_x, move_y, fire }, so remapping keys (or adding a gamepad) only
-- touches this module. `is_down` is injected (love.keyboard.isDown in the
-- game, a table lookup in tests).

local Input = {}

Input.BINDINGS = {
  left = { "left", "a" },
  right = { "right", "d" },
  up = { "up", "w" },
  down = { "down", "s" },
  fire = { "space", "z", "j" },
}

local function any_down(is_down, keys)
  for _, key in ipairs(keys) do
    if is_down(key) then
      return true
    end
  end
  return false
end

function Input.controls(is_down)
  local b = Input.BINDINGS
  local x, y = 0, 0
  if any_down(is_down, b.left) then
    x = x - 1
  end
  if any_down(is_down, b.right) then
    x = x + 1
  end
  if any_down(is_down, b.up) then
    y = y - 1
  end
  if any_down(is_down, b.down) then
    y = y + 1
  end
  return { move_x = x, move_y = y, fire = any_down(is_down, b.fire) }
end

--- Parses LÖVE's command-line args: an optional numeric seed and
-- `--frames N` (run N fixed 1/60 s frames with no input, then quit; used for
-- headless smoke runs).
function Input.parse_args(args, default_seed)
  local options = { seed = default_seed }
  local i = 1
  while args and i <= #args do
    local arg = args[i]
    if arg == "--frames" then
      local n = tonumber(args[i + 1])
      if not n or n < 1 or n % 1 ~= 0 then
        error("--frames needs a positive integer", 2)
      end
      options.frames = n
      i = i + 1
    elseif tonumber(arg) then
      options.seed = tonumber(arg)
    else
      error("unknown argument '" .. tostring(arg) .. "'", 2)
    end
    i = i + 1
  end
  return options
end

return Input
