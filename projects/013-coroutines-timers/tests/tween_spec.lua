local Tween = require("core.tween")
local Scheduler = require("core.scheduler")

describe("Easing curves", function()
  it("start at 0 and end at 1", function()
    for name, ease in pairs(Tween.easing) do
      assert.is_near(0, ease(0), 1e-9, name)
      assert.is_near(1, ease(1), 1e-9, name)
    end
  end)

  it("have the expected shapes at the midpoint", function()
    local e = Tween.easing
    assert.is_near(0.5, e.linear(0.5), 1e-12)
    assert.is_near(0.25, e.quad_in(0.5), 1e-12)
    assert.is_near(0.75, e.quad_out(0.5), 1e-12)
    assert.is_near(0.5, e.quad_in_out(0.5), 1e-12)
    assert.is_near(0.5, e.sine_in_out(0.5), 1e-12)
    assert.is_true(e.cubic_out(0.5) > e.quad_out(0.5))
  end)

  it("back_out overshoots past 1 before settling", function()
    local peak = 0
    for i = 0, 100 do
      peak = math.max(peak, Tween.easing.back_out(i / 100))
    end
    assert.is_true(peak > 1.05)
  end)
end)

describe("Tween", function()
  it("interpolates every property with the easing and snaps to the goal", function()
    local obj = { x = 0, y = 100 }
    local tween = Tween.new(obj, 1, { x = 10, y = 0 }, "quad_out")
    assert.is_false((tween:advance(0.5)))
    assert.is_near(7.5, obj.x, 1e-9)
    assert.is_near(25, obj.y, 1e-9)
    local done, leftover = tween:advance(0.7)
    assert.is_true(done)
    assert.is_near(0.2, leftover, 1e-9)
    assert.equal(10, obj.x)
    assert.equal(0, obj.y)
  end)

  it("captures start values on the first advance, not at construction", function()
    local obj = { x = 0 }
    local tween = Tween.new(obj, 1, { x = 10 })
    obj.x = 5 -- moved by an earlier step of a sequence
    tween:advance(0.5)
    assert.is_near(7.5, obj.x, 1e-9)
  end)

  it("accepts a custom easing function", function()
    local obj = { a = 0 }
    local tween = Tween.new(obj, 2, { a = 1 }, function(t)
      return t * t * t
    end)
    tween:advance(1)
    assert.is_near(0.125, obj.a, 1e-12)
  end)

  it("validates its arguments", function()
    assert.has_error(function()
      Tween.new(nil, 1, { x = 1 })
    end)
    assert.has_error(function()
      Tween.new({ x = 0 }, 0, { x = 1 })
    end)
    assert.has_error(function()
      Tween.new({ x = 0 }, 1, {})
    end)
    assert.has_error(function()
      Tween.new({ x = 0 }, 1, { x = "far" })
    end)
    assert.has_error(function()
      Tween.new({ x = 0 }, 1, { x = 1 }, "wobbly")
    end)
    local bad = Tween.new({ x = "text" }, 1, { x = 1 })
    assert.has_error(function()
      bad:advance(0.1)
    end)
  end)
end)

describe("Tween sequences in tasks", function()
  it("chains steps exactly, carrying time across step boundaries", function()
    local sched = Scheduler.new()
    local obj = { x = 0 }
    local task = sched:spawn(function()
      sched:tween(obj, 0.25, { x = 100 })
      sched:tween(obj, 0.25, { x = 0 })
    end)
    sched:update(0.375) -- whole first step plus half of the second
    assert.is_near(50, obj.x, 1e-9)
    sched:update(0.125)
    assert.equal(0, obj.x)
    assert.equal("dead", task.status)
  end)

  it("ends a sequence at the same time and place at any frame rate", function()
    local function finish_time(dt)
      local sched = Scheduler.new()
      local obj = { x = 0, y = 0 }
      local done_at
      sched:spawn(function()
        sched:tween(obj, 0.4, { x = 50 }, "back_out")
        sched:wait(0.3)
        sched:all({
          function()
            sched:tween(obj, 0.5, { x = 0 }, "sine_in_out")
          end,
          function()
            sched:tween(obj, 0.7, { y = 30 }, "quad_in")
          end,
        })
        done_at = sched.time
      end)
      while not done_at do
        sched:update(dt)
      end
      return done_at, obj
    end
    for _, dt in ipairs({ 1 / 30, 1 / 60, 1 / 144 }) do
      local t, obj = finish_time(dt)
      -- 0.4 + 0.3 + 0.7 = 1.4 s, finishing within the frame it lands in.
      assert.is_true(t >= 1.4 - 1e-9 and t < 1.4 + dt, "dt = " .. dt)
      assert.equal(0, obj.x)
      assert.equal(30, obj.y)
    end
  end)
end)
