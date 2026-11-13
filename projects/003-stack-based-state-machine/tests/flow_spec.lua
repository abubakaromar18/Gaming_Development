-- End-to-end screen flow through the real states, driven the same way the
-- LÖVE shell drives them: abstract actions in, stack:update(dt), views out.
local Game = require("core.game")

local function kinds(stack)
  local out = {}
  for i, v in ipairs(stack:views()) do out[i] = v.kind end
  return out
end

local function play_state(stack)
  for _, s in ipairs(stack.states) do
    if s.name == "play" then return s end
  end
end

-- Steer the snake straight into the right wall (12 cells from the start).
local function crash(stack)
  for _ = 1, 100 do
    if stack:top().name == "game_over" then return end
    stack:update(0.2)
  end
  error("snake never crashed")
end

describe("screen flow", function()
  local stack, ctx, saved

  before_each(function()
    saved = {}
    stack, ctx = Game.new({
      seed = 3,
      best = 2,
      config = { countdown = 1, wrap = false },
      on_new_best = function(best) saved[#saved + 1] = best end,
    })
  end)

  it("boots into the title menu", function()
    assert.same({ "menu" }, stack:names())
    assert.same({ "menu" }, kinds(stack))
  end)

  it("Play pushes the round with a countdown overlay on top", function()
    stack:handle("confirm")
    assert.same({ "menu", "play", "countdown" }, stack:names())
    -- The menu is hidden under the opaque play state.
    assert.same({ "play", "countdown" }, kinds(stack))
  end)

  it("freezes the snake during the countdown, then lets it move", function()
    stack:handle("confirm")
    local play = play_state(stack)
    local start = play.snake:head().x
    stack:update(0.5)
    assert.equals(start, play.snake:head().x)
    stack:update(0.6) -- countdown ends; pops at the end of this update
    assert.same({ "menu", "play" }, stack:names())
    stack:update(0.2)
    assert.is_true(play.snake:head().x > start)
  end)

  it("pause suspends the round; resuming adds a fresh countdown", function()
    stack:handle("confirm")
    stack:update(1.1)
    local play = play_state(stack)
    stack:handle("pause")
    assert.same({ "menu", "play", "pause" }, stack:names())
    local frozen = play.snake:head().x
    stack:update(5)
    assert.equals(frozen, play.snake:head().x)
    -- Steering while paused goes to the pause menu, not the snake.
    stack:handle("up")
    assert.same({}, play.snake.turns)
    stack:handle("pause")
    assert.same({ "menu", "play", "countdown" }, stack:names())
  end)

  it("Quit to menu unwinds two states in one action", function()
    stack:handle("confirm")
    stack:update(1.1)
    stack:handle("back")   -- open pause
    stack:handle("down")
    stack:handle("down")   -- "Quit to menu"
    stack:handle("confirm")
    assert.same({ "menu" }, stack:names())
  end)

  it("Restart from pause resets the round", function()
    stack:handle("confirm")
    stack:update(1.1)
    local play = play_state(stack)
    play.snake.score = 5
    stack:handle("pause")
    stack:handle("down")   -- "Restart"
    stack:handle("confirm")
    assert.equals(0, play.snake.score)
    assert.same({ "menu", "play", "countdown" }, stack:names())
  end)

  it("a crash shows game over over the final board and records the score", function()
    stack:handle("confirm")
    stack:update(1.1)
    local play = play_state(stack)
    play.snake.food = { x = 0, y = 0 }
    play.snake.score = 4
    crash(stack)
    assert.same({ "menu", "play", "game_over" }, stack:names())
    assert.same({ "play", "game_over" }, kinds(stack))
    local view = stack:views()[2]
    assert.equals(4, view.score)
    assert.is_true(view.new_best)
    assert.equals(4, ctx.scores.best)
    assert.same({ 4 }, saved)
  end)

  it("a score below the best is not saved", function()
    stack:handle("confirm")
    stack:update(1.1)
    play_state(stack).snake.food = { x = 0, y = 0 }
    crash(stack)
    assert.same({}, saved)
    assert.is_false(stack:views()[2].new_best)
  end)

  it("Play again restarts in place; Menu returns to the title", function()
    stack:handle("confirm")
    stack:update(1.1)
    play_state(stack).snake.food = { x = 0, y = 0 }
    crash(stack)
    stack:handle("confirm") -- "Play again"
    assert.same({ "menu", "play", "countdown" }, stack:names())
    assert.is_true(play_state(stack).snake:is_alive())

    stack:update(1.1)
    play_state(stack).snake.food = { x = 0, y = 0 }
    crash(stack)
    stack:handle("down")
    stack:handle("confirm") -- "Menu"
    assert.same({ "menu" }, stack:names())
    assert.equals(0, stack:views()[1].last)
  end)

  it("alt-tab (suspend) pauses a round but is ignored elsewhere", function()
    stack:handle("suspend")
    assert.same({ "menu" }, stack:names())
    stack:handle("confirm")
    stack:update(1.1)
    stack:handle("suspend")
    assert.same({ "menu", "play", "pause" }, stack:names())
    stack:handle("suspend")
    assert.same({ "menu", "play", "pause" }, stack:names())
  end)

  it("quitting from the title menu empties the stack", function()
    stack:handle("down")
    stack:handle("confirm")
    assert.is_true(stack:is_empty())
  end)

  it("replays identically from the same seed and inputs", function()
    local function run(seed)
      local s = Game.new({ seed = seed, config = { countdown = 0, wrap = true } })
      s:handle("confirm")
      local script = { "up", "left", "down", "right" }
      for frame = 1, 600 do
        if frame % 37 == 0 then s:handle(script[(frame / 37) % 4 + 1]) end
        s:update(1 / 60)
      end
      local snake = play_state(s).snake
      return { snake.score, snake:head().x, snake:head().y, snake.food and snake.food.x }
    end
    assert.same(run(11), run(11))
  end)
end)
