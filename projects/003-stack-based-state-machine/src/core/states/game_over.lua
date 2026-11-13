--- End-of-round overlay over the final board. Like Pause, it reports the
--- player's choice as a pop result instead of reaching into other states.

local MenuList = require("core.menu_list")

local GameOver = {}
GameOver.__index = GameOver

--- @param result table { score, best, new_best, won }
function GameOver.new(result)
  return setmetatable({
    name = "game_over",
    transparent = true,
    result = result,
    list = MenuList.new({
      { id = "restart", label = "Play again" },
      { id = "menu", label = "Menu" },
    }),
  }, GameOver)
end

function GameOver:enter(stack)
  self.stack = stack
end

function GameOver:handle(action)
  if action == "back" then
    self.stack:pop("menu")
    return
  end
  local choice = self.list:handle(action)
  if choice then
    self.stack:pop(choice)
  end
end

function GameOver:view()
  local r = self.result
  return {
    kind = "game_over",
    title = r.won and "YOU WIN" or "GAME OVER",
    score = r.score,
    best = r.best,
    new_best = r.new_best,
    menu = self.list:view(),
  }
end

return GameOver
