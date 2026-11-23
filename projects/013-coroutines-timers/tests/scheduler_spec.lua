local Scheduler = require("core.scheduler")

-- Steps a scheduler by `total` seconds in frames of `dt`.
local function run(sched, total, dt)
  local steps = math.floor(total / dt + 0.5)
  for _ = 1, steps do
    sched:update(dt)
  end
end

describe("Scheduler tasks", function()
  it("runs a spawned task immediately until its first wait", function()
    local sched = Scheduler.new()
    local log = {}
    sched:spawn(function()
      log[#log + 1] = "start"
      sched:wait(1)
      log[#log + 1] = "end"
    end)
    assert.same({ "start" }, log)
    sched:update(0.5)
    assert.same({ "start" }, log)
    sched:update(0.5)
    assert.same({ "start", "end" }, log)
  end)

  it("passes spawn arguments to the task and keeps its return values", function()
    local sched = Scheduler.new()
    local task = sched:spawn(function(a, b)
      return a + b, "done"
    end, "adder", 2, 3)
    assert.equal("dead", task.status)
    assert.same({ n = 2, 5, "done" }, task.results)
    assert.equal(1, sched:count()) -- pruned on the next update
    sched:update(0)
    assert.equal(0, sched:count())
  end)

  it("carries leftover time into the next wait in the same frame", function()
    local sched = Scheduler.new()
    local log = {}
    sched:spawn(function()
      for i = 1, 5 do
        sched:wait(0.1)
        log[#log + 1] = i
      end
    end)
    sched:update(0.35) -- 3.5 waits' worth of time in a single frame
    assert.same({ 1, 2, 3 }, log)
    sched:update(0.05)
    assert.same({ 1, 2, 3, 4 }, log)
  end)

  it("gives a task spawned mid-frame its parent's leftover time", function()
    local sched = Scheduler.new()
    local obj = { x = 0 }
    sched:spawn(function()
      sched:wait(0.25)
      sched:spawn(function()
        sched:tween(obj, 1, { x = 100 })
      end)
    end)
    sched:update(0.5) -- the child starts 0.25 s into this frame
    assert.is_near(25, obj.x, 1e-9)
  end)

  it("fires a repeating timer floor(t / interval) times at any frame rate", function()
    for _, dt in ipairs({ 1 / 30, 1 / 60, 1 / 144, 0.25 }) do
      local sched = Scheduler.new()
      local fired = 0
      sched:every(0.1, function()
        fired = fired + 1
      end)
      run(sched, 3, dt)
      assert.equal(30, fired, "dt = " .. dt)
    end
  end)

  it("stays drift-free under jittery frame times", function()
    local sched = Scheduler.new()
    local fired = 0
    sched:every(0.05, function()
      fired = fired + 1
    end)
    local frames = { 0.013, 0.021, 0.008, 0.034, 0.017 }
    local t = 0
    for i = 1, 1000 do
      local dt = frames[(i % #frames) + 1]
      t = t + dt
      sched:update(dt)
    end
    assert.equal(math.floor(t / 0.05 + 1e-9), fired)
  end)

  it("limits every() to count calls and passes the call index", function()
    local sched = Scheduler.new()
    local seen = {}
    local task = sched:every(0.2, function(i)
      seen[#seen + 1] = i
    end, 3)
    run(sched, 2, 0.1)
    assert.same({ 1, 2, 3 }, seen)
    assert.equal("dead", task.status)
  end)

  it("calls after() once and lets the caller cancel it first", function()
    local sched = Scheduler.new()
    local a, b = 0, 0
    sched:after(0.5, function()
      a = a + 1
    end)
    local cancelled = sched:after(0.5, function()
      b = b + 1
    end)
    sched:update(0.3)
    cancelled:cancel()
    run(sched, 2, 0.1)
    assert.equal(1, a)
    assert.equal(0, b)
  end)

  it("frame() waits exactly one update and returns its dt", function()
    local sched = Scheduler.new()
    local seen = {}
    sched:spawn(function()
      for _ = 1, 3 do
        seen[#seen + 1] = sched:frame()
      end
    end)
    sched:update(0.5) -- big frame: still only one resume, carry is dropped
    sched:update(0.25)
    assert.same({ 0.5, 0.25 }, seen)
  end)

  it("wait_until resumes on the first update where the predicate holds", function()
    local sched = Scheduler.new()
    local open, passed = false, false
    sched:spawn(function()
      sched:wait_until(function()
        return open
      end)
      passed = true
    end)
    sched:update(1)
    assert.is_false(passed)
    open = true
    sched:update(0)
    assert.is_true(passed)
  end)

  it("wakes signal waiters with the emitted payload", function()
    local sched = Scheduler.new()
    local got = {}
    for i = 1, 2 do
      sched:spawn(function()
        local kind, amount = sched:wait_signal("hit")
        got[i] = kind .. amount
      end)
    end
    assert.equal(2, sched:emit("hit", "fire", 3))
    assert.equal(0, sched:emit("hit", "late", 0)) -- waiters were consumed
    sched:update(0)
    assert.same({ "fire3", "fire3" }, got)
  end)

  it("join resumes the parent in the frame the child finishes", function()
    local sched = Scheduler.new()
    local finished_at
    sched:spawn(function()
      local child = sched:spawn(function()
        sched:wait(0.3)
        return "loot"
      end)
      local value = sched:join(child)
      finished_at = sched.time
      assert.equal("loot", value)
    end)
    run(sched, 0.3, 0.1)
    assert.is_near(0.3, finished_at, 1e-9)
  end)

  it("all() waits for every child and collects first return values", function()
    local sched = Scheduler.new()
    local results
    sched:spawn(function()
      results = sched:all({
        function()
          sched:wait(0.2)
          return "a"
        end,
        function()
          sched:wait(0.5)
          return "b"
        end,
      })
    end)
    run(sched, 0.4, 0.1)
    assert.is_nil(results)
    run(sched, 0.1, 0.1)
    assert.same({ "a", "b" }, results)
  end)

  it("race() returns the first finisher and cancels the rest", function()
    local sched = Scheduler.new()
    local winner, value
    local slow_ran = false
    sched:spawn(function()
      winner, value = sched:race({
        function()
          sched:wait(1)
          slow_ran = true
        end,
        function()
          sched:wait(0.2)
          return "fast"
        end,
      })
    end)
    run(sched, 2, 0.1)
    assert.equal(2, winner)
    assert.equal("fast", value)
    assert.is_false(slow_ran)
    assert.equal(0, sched:count())
  end)
end)

describe("Scheduler cancellation", function()
  it("cancels children recursively and runs deferred cleanups newest first", function()
    local sched = Scheduler.new()
    local log = {}
    local parent = sched:spawn(function()
      sched:defer(function()
        log[#log + 1] = "parent-outer"
      end)
      sched:defer(function()
        log[#log + 1] = "parent-inner"
      end)
      sched:spawn(function()
        sched:defer(function()
          log[#log + 1] = "child"
        end)
        sched:wait(10)
      end)
      sched:wait(10)
    end)
    parent:cancel()
    assert.same({ "child", "parent-inner", "parent-outer" }, log)
    parent:cancel() -- idempotent
    assert.equal(3, #log)
    sched:update(0.1)
    assert.equal(0, sched:count())
  end)

  it("runs cleanups when a task returns normally too", function()
    local sched = Scheduler.new()
    local cleaned = false
    sched:spawn(function()
      sched:defer(function()
        cleaned = true
      end)
      sched:wait(0.1)
    end)
    sched:update(0.1)
    assert.is_true(cleaned)
  end)

  it("hands a finished helper's live children to its parent", function()
    local sched = Scheduler.new()
    local grandchild
    local root = sched:spawn(function()
      sched:all({
        function()
          grandchild = sched:spawn(function()
            sched:wait(10)
          end)
        end,
      })
      sched:wait(10)
    end)
    sched:update(0.1)
    assert.equal(root, grandchild.parent)
    root:cancel()
    assert.equal("cancelled", grandchild.status)
  end)

  it("lets a task cancel itself; it stops at its next yield", function()
    local sched = Scheduler.new()
    local after = false
    local task
    task = sched:spawn(function()
      sched:wait(0.1)
      task:cancel()
      sched:wait(0.1)
      after = true
    end)
    run(sched, 1, 0.1)
    assert.equal("cancelled", task.status)
    assert.is_false(after)
  end)

  it("cancel_all stops every task", function()
    local sched = Scheduler.new()
    local fired = 0
    sched:every(0.1, function()
      fired = fired + 1
    end)
    sched:cancel_all()
    run(sched, 1, 0.1)
    assert.equal(0, fired)
    assert.equal(0, sched:count())
  end)
end)

describe("Scheduler errors", function()
  it("reports script errors with the task name and a traceback", function()
    local sched = Scheduler.new()
    sched:spawn(function()
      sched:wait(0.1)
      error("boom")
    end, "exploder")
    local ok, err = pcall(sched.update, sched, 0.1)
    assert.is_false(ok)
    assert.matches("task 'exploder' failed", err)
    assert.matches("boom", err)
    assert.matches("traceback", err)
  end)

  it("rejects yielding something that is not an awaitable", function()
    local sched = Scheduler.new()
    local ok, err = pcall(sched.spawn, sched, function()
      coroutine.yield(42)
    end, "bad")
    assert.is_false(ok)
    assert.matches("non%-awaitable", err)
  end)

  it("refuses waits outside a task and invalid arguments", function()
    local sched = Scheduler.new()
    assert.has_error(function()
      sched:wait(1)
    end)
    assert.has_error(function()
      sched:frame()
    end)
    assert.has_error(function()
      sched:spawn("not a function")
    end)
    assert.has_error(function()
      sched:update(-1)
    end)
    sched:spawn(function()
      assert.has_error(function()
        sched:wait(0)
      end)
      assert.has_error(function()
        sched:wait_until(true)
      end)
    end)
  end)

  it("accepts custom awaitables through await()", function()
    local sched = Scheduler.new()
    -- Resolves after N updates, returning how many it saw.
    local Countdown = {}
    Countdown.__index = Countdown
    function Countdown:advance()
      self.seen = self.seen + 1
      return self.seen > self.n, 0, self.seen - 1
    end
    local got
    sched:spawn(function()
      got = sched:await(setmetatable({ n = 2, seen = 0 }, Countdown))
    end)
    sched:update(0)
    sched:update(0)
    assert.equal(2, got)
  end)
end)
