local Config = require("core.config")
local MenuList = require("core.menu_list")
local Rng = require("core.rng")
local Scores = require("core.scores")
local Input = require("platform.input")

describe("Rng", function()
  it("is deterministic per seed", function()
    local a, b = Rng.new(42), Rng.new(42)
    for _ = 1, 100 do assert.equals(a:next(), b:next()) end
  end)

  it("stays in range and covers it", function()
    local r, seen = Rng.new(5), {}
    for _ = 1, 2000 do
      local v = r:int(1, 6)
      assert.is_true(v >= 1 and v <= 6)
      seen[v] = true
    end
    for v = 1, 6 do assert.is_true(seen[v]) end
  end)

  it("handles a zero seed and rejects bad input", function()
    assert.is_true(Rng.new(0):next() > 0)
    assert.has_error(function() Rng.new("x") end)
    assert.has_error(function() Rng.new(1):int(3, 2) end)
  end)
end)

describe("Config", function()
  it("fills in defaults", function()
    assert.same(Config.DEFAULTS, Config.build())
  end)

  it("applies valid overrides", function()
    assert.equals(true, Config.build({ wrap = true }).wrap)
  end)

  it("rejects unknown keys, wrong types and bad values", function()
    assert.has_error(function() Config.build({ speeed = 1 }) end)
    assert.has_error(function() Config.build({ cols = "24" }) end)
    assert.has_error(function() Config.build({ cols = 3 }) end)
    assert.has_error(function() Config.build({ min_tick = 1, base_tick = 0.5 }) end)
    assert.has_error(function() Config.build({ start_length = 20 }) end)
  end)
end)

describe("MenuList", function()
  it("wraps the cursor and returns the chosen id on confirm", function()
    local m = MenuList.new({ { id = "a", label = "A" }, { id = "b", label = "B" } })
    assert.is_nil(m:handle("up"))
    assert.equals("b", m:handle("confirm"))
    m:handle("down")
    assert.equals("a", m:selected())
    assert.same({ labels = { "A", "B" }, index = 1 }, m:view())
  end)

  it("needs at least one item", function()
    assert.has_error(function() MenuList.new({}) end)
  end)
end)

describe("Scores", function()
  it("tracks the best and reports new bests", function()
    local s = Scores.new(3)
    assert.is_false(s:record(2))
    assert.is_true(s:record(5))
    assert.equals(5, s.best)
    assert.equals(5, s.last)
  end)

  it("treats corrupt saved values as zero", function()
    assert.equals(0, Scores.new("garbage").best)
    assert.equals(0, Scores.new(-4).best)
    assert.equals(0, Scores.new(1.5).best)
  end)
end)

describe("platform.input", function()
  it("maps keys to abstract actions", function()
    assert.equals("up", Input.action_for_key("w"))
    assert.equals("confirm", Input.action_for_key("return"))
    assert.equals("back", Input.action_for_key("escape"))
    assert.is_nil(Input.action_for_key("f12"))
  end)
end)
