--- Title screen: the bottom of the stack. Popping it empties the stack, and
--- an empty stack is how the game says "quit". No global quit flag needed.

local MenuList = require("core.menu_list")
local Play = require("core.states.play")

local Menu = {}
Menu.__index = Menu

function Menu.new(ctx)
  return setmetatable({
    name = "menu",
    ctx = ctx,
    list = MenuList.new({
      { id = "play", label = "Play" },
      { id = "quit", label = "Quit" },
    }),
  }, Menu)
end

function Menu:enter(stack)
  self.stack = stack
end

function Menu:handle(action)
  if action == "back" then
    self.stack:pop()
    return
  end
  local choice = self.list:handle(action)
  if choice == "play" then
    self.stack:push(Play.new(self.ctx))
  elseif choice == "quit" then
    self.stack:pop()
  end
end

function Menu:view()
  return {
    kind = "menu",
    title = "STACK SNAKE",
    menu = self.list:view(),
    best = self.ctx.scores.best,
    last = self.ctx.scores.last,
  }
end

return Menu
