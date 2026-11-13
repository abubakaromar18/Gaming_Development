local Config = require("core.config")
local Rng = require("core.rng")
local Snake = require("core.snake")

local function make(overrides, seed)
  return Snake.new(Config.build(overrides), Rng.new(seed or 7))
end

-- Run exactly n moves regardless of the current tick interval.
local function steps(snake, n)
  local events = {}
  for _ = 1, n do
    local e = snake:step()
    if e then events[#events + 1] = e end
  end
  return events
end

local function place_food(snake, x, y)
  snake.food = { x = x, y = y }
end

describe("Snake", function()
  it("starts centred, heading right, with food on a free cell", function()
    local s = make()
    assert.same({ x = 12, y = 9 }, s:head())
    assert.equals(4, s:length())
    assert.equals("right", s.dir)
    assert.is_nil(s.occupied[s:key(s.food.x, s.food.y)])
  end)

  it("moves one cell per tick interval and keeps the remainder", function()
    -- Binary-exact times so the assertion is about the rule, not rounding.
    local s = make({ base_tick = 0.25, min_tick = 0.25 })
    place_food(s, 0, 0)
    s:update(0.625)
    assert.same({ x = 14, y = 9 }, s:head())
    s:update(0.125)
    assert.same({ x = 15, y = 9 }, s:head())
  end)

  it("caps catch-up moves after a long frame", function()
    local s = make({ base_tick = 0.1, min_tick = 0.1, max_steps_per_update = 3 })
    place_food(s, 0, 0)
    s:update(5)
    assert.same({ x = 15, y = 9 }, s:head())
    assert.equals(0, s.accumulator)
  end)

  it("refuses to reverse into its own neck, even via a quick double tap", function()
    local s = make()
    assert.is_false(s:steer("left"))
    assert.is_true(s:steer("up"))
    assert.is_false(s:steer("down")) -- opposite of the queued "up"
    assert.is_true(s:steer("left"))
    place_food(s, 0, 0)
    steps(s, 2)
    assert.same({ x = 11, y = 8 }, s:head())
  end)

  it("buffers at most turn_buffer turns", function()
    local s = make({ turn_buffer = 2 })
    assert.is_true(s:steer("up"))
    assert.is_true(s:steer("left"))
    assert.is_false(s:steer("down"))
  end)

  it("rejects unknown directions", function()
    assert.has_error(function() make():steer("sideways") end)
  end)

  it("grows, scores and speeds up when it eats", function()
    local s = make()
    local before = s:tick_interval()
    place_food(s, 13, 9)
    assert.same({ "ate" }, steps(s, 1))
    assert.equals(1, s.score)
    assert.equals(5, s:length())
    assert.is_true(s:tick_interval() < before)
    assert.is_nil(s.occupied[s:key(s.food.x, s.food.y)])
  end)

  it("never moves faster than min_tick", function()
    local s = make({ base_tick = 0.1, min_tick = 0.05, speedup = 0.5 })
    s.score = 10
    assert.equals(0.05, s:tick_interval())
  end)

  it("dies on hitting a wall", function()
    local s = make()
    place_food(s, 0, 0)
    local events = steps(s, 12)
    assert.same({ "died" }, events)
    assert.equals("dead", s.status)
    assert.is_false(s:steer("up"))
    assert.same({}, s:update(10))
  end)

  it("wraps around the edges when configured to", function()
    local s = make({ wrap = true })
    place_food(s, 0, 0)
    assert.same({}, steps(s, 12))
    assert.same({ x = 0, y = 9 }, s:head())
  end)

  it("dies on running into itself", function()
    local s = make({ start_length = 5 })
    place_food(s, 0, 0)
    s:steer("up"); steps(s, 1)
    s:steer("left"); steps(s, 1)
    s:steer("down")
    assert.same({ "died" }, steps(s, 1))
  end)

  it("may move into the cell its tail is vacating", function()
    local s = make({ start_length = 4 })
    place_food(s, 0, 0)
    -- A 2x2 loop: the head chases the tail around the square forever.
    s.body = { { x = 5, y = 5 }, { x = 5, y = 6 }, { x = 6, y = 6 }, { x = 6, y = 5 } }
    s.occupied = {}
    for _, seg in ipairs(s.body) do s.occupied[s:key(seg.x, seg.y)] = true end
    s.dir = "up"
    s.turns = { "right" }
    assert.same({}, steps(s, 1))
    assert.same({ x = 6, y = 5 }, s:head())
    assert.equals("alive", s.status)
  end)

  it("wins when the last free cell is eaten", function()
    local s = make({ cols = 5, rows = 5, start_length = 1 })
    s.body, s.occupied = {}, {}
    -- Fill the board with a serpentine body, head at (3,4), leaving only (4,4) free.
    for y = 4, 0, -1 do
      local leftward = (4 - y) % 2 == 0
      for i = 0, 4 do
        local x = leftward and (4 - i) or i
        if not (x == 4 and y == 4) then
          s.body[#s.body + 1] = { x = x, y = y }
        end
      end
    end
    for _, seg in ipairs(s.body) do s.occupied[s:key(seg.x, seg.y)] = true end
    s.dir = "right" -- head at (3,4), moving onto (4,4)
    place_food(s, 4, 4)
    assert.same({ "won" }, steps(s, 1))
    assert.equals("won", s.status)
    assert.is_nil(s.food)
  end)

  it("places food identically for the same seed", function()
    local a, b = make(nil, 99), make(nil, 99)
    for _ = 1, 20 do
      assert.same(a.food, b.food)
      a:spawn_food(); b:spawn_food()
    end
  end)

  it("reset starts a fresh round", function()
    local s = make()
    place_food(s, 13, 9)
    steps(s, 1)
    s:steer("up")
    s:reset()
    assert.equals(0, s.score)
    assert.equals(4, s:length())
    assert.equals("right", s.dir)
    assert.same({}, s.turns)
  end)
end)
