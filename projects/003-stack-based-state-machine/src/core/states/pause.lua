--- Pause overlay. It never touches the round directly: it pops with a result
--- ("resume", "restart", "menu") and lets Play decide what that means.

local MenuList = require("core.menu_list")

local Pause = {}
Pause.__index = Pause

function Pause.new()
  return setmetatable({
    name = "pause",
    transparent = true,
    list = MenuList.new({
      { id = "resume", label = "Resume" },
      { id = "restart", label = "Restart" },
      { id = "menu", label = "Quit to menu" },
    }),
  }, Pause)
end

function Pause:enter(stack)
  self.stack = stack
end

function Pause:handle(action)
  if action == "pause" or action == "back" then
    self.stack:pop("resume")
    return
  end
  local choice = self.list:handle(action)
  if choice then
    self.stack:pop(choice)
  end
end

function Pause:view()
  return { kind = "pause", title = "PAUSED", menu = self.list:view() }
end

return Pause
