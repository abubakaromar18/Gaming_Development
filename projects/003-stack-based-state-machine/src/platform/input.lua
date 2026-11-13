--- Translate LÖVE key names into the abstract actions the core understands.
-- States never see key codes, so remapping keys only touches this table.

local Input = {}

Input.KEYMAP = {
  up = "up", w = "up",
  down = "down", s = "down",
  left = "left", a = "left",
  right = "right", d = "right",
  ["return"] = "confirm", kpenter = "confirm", space = "confirm",
  escape = "back", backspace = "back",
  p = "pause",
}

--- @return string|nil action for a pressed key, or nil if the key is unbound
function Input.action_for_key(key)
  return Input.KEYMAP[key]
end

return Input
