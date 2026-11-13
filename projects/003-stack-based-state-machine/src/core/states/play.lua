--- The round in progress. It sits in the middle of the stack and acts like a
--- caller: it pushes Countdown, Pause and GameOver as "subroutines" and reacts
--- to the result each one returns when popped (see resume).

local Snake = require("core.snake")
local Countdown = require("core.states.countdown")
local Pause = require("core.states.pause")
local GameOver = require("core.states.game_over")

local Play = {}
Play.__index = Play

local STEER = { up = true, down = true, left = true, right = true }

function Play.new(ctx)
  return setmetatable({
    name = "play",
    ctx = ctx,
    snake = Snake.new(ctx.config, ctx.rng),
  }, Play)
end

function Play:enter(stack)
  self.stack = stack
  self.stack:push(Countdown.new(self.ctx))
end

function Play:restart()
  self.snake:reset()
  self.stack:push(Countdown.new(self.ctx))
end

--- The state above us was popped. Its result tells us what the player chose.
function Play:resume(result)
  if result == "resume" then
    -- Back from pause: give the player a moment to re-orient.
    self.stack:push(Countdown.new(self.ctx))
  elseif result == "restart" then
    self:restart()
  elseif result == "menu" then
    self.stack:pop()
  end
  -- nil: the countdown finished; carry on.
end

function Play:finish(won)
  local score = self.snake.score
  local new_best = self.ctx.scores:record(score)
  if new_best then
    self.ctx.on_new_best(self.ctx.scores.best)
  end
  self.stack:push(GameOver.new({
    score = score,
    best = self.ctx.scores.best,
    new_best = new_best,
    won = won,
  }))
end

function Play:update(dt)
  local events = self.snake:update(dt)
  for _, event in ipairs(events) do
    if event == "died" or event == "won" then
      self:finish(event == "won")
    end
  end
end

function Play:handle(action)
  if STEER[action] then
    self.snake:steer(action)
  elseif action == "pause" or action == "back" or action == "suspend" then
    self.stack:push(Pause.new())
  end
end

function Play:view()
  local snake = self.snake
  return {
    kind = "play",
    cols = self.ctx.config.cols,
    rows = self.ctx.config.rows,
    body = snake.body, -- read-only for the renderer
    dir = snake.dir,
    food = snake.food,
    score = snake.score,
    best = self.ctx.scores.best,
    status = snake.status,
  }
end

return Play
