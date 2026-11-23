--- Cooperative task scheduler built on Lua coroutines.
--
-- A *task* is a coroutine that runs game script code top to bottom and pauses
-- by yielding an *awaitable*: "wait 0.4 s", "tween this ship to (200, 80)",
-- "wait until no enemies are left". Each frame the scheduler advances every
-- task's awaitable by dt and resumes the task once it completes. Scripted
-- sequences then read like the design doc instead of a state machine:
--
--   sched:spawn(function()
--     sched:tween(banner, 0.5, { alpha = 1 }, "quad_out")
--     sched:wait(1.5)
--     sched:tween(banner, 0.5, { alpha = 0 })
--   end)
--
-- Timing is exact: when an awaitable finishes partway through a frame, the
-- leftover time is carried into the task's next awaitable in the same frame.
-- `every(0.1, fn)` therefore fires exactly floor(t / 0.1) times by time t at
-- 30 FPS, 144 FPS or with jittery frame times. Naive "reset the timer to zero"
-- code drifts later by up to one frame per tick.
--
-- Awaitables are any object with `advance(dt) -> done, leftover, ...results`.
-- `advance` is called once with the carried time when the task yields it (often
-- 0), then once per update with the frame's dt. Time-based awaitables return
-- the unused part of dt as `leftover`; event-based ones (predicate, signal,
-- join) are marked `event = true`, return 0 leftover, and are re-checked at
-- the end of every update until nothing else wakes ("settling"). So a parent
-- joining a child resumes in the same frame the child finishes, regardless of
-- which of them comes first in the update order.

local Tween = require("core.tween")

local unpack = unpack or table.unpack -- luacheck: ignore (Lua 5.1 / 5.2+)

local Scheduler = {}
Scheduler.__index = Scheduler

-- Thirty frames of 1/30 s don't sum to exactly 1.0 in binary floating point.
-- Time-based awaitables treat anything within this tolerance as "done" so a
-- 1 s wait at 30 FPS doesn't slip a whole frame over a rounding error.
Scheduler.EPSILON = 1e-9

local Task = {}
Task.__index = Task

-- Packs varargs with their count so trailing nils survive (no table.pack in 5.1).
local function pack(...)
  return { n = select("#", ...), ... }
end

---------------------------------------------------------------------------
-- Built-in awaitables
---------------------------------------------------------------------------

-- Wait a fixed amount of time, consuming dt and handing back the excess.
local Delay = {}
Delay.__index = Delay

function Delay.new(seconds)
  return setmetatable({ remaining = seconds }, Delay)
end

function Delay:advance(dt)
  self.remaining = self.remaining - dt
  if self.remaining <= Scheduler.EPSILON then
    return true, math.max(0, -self.remaining)
  end
  return false, 0
end

-- Wait for the next update; resumes with that frame's dt. Carried time is
-- dropped on purpose: "next frame" means a frame boundary, and dropping it is
-- what guarantees a `while true do sched:frame() end` loop yields.
local Frame = {}
Frame.__index = Frame

function Frame.new()
  return setmetatable({ armed = false }, Frame)
end

function Frame:advance(dt)
  if not self.armed then
    self.armed = true
    return false, 0
  end
  return true, 0, dt
end

-- Wait until a predicate holds. Checked as soon as it is yielded, during each
-- update, and again while the update settles.
local Until = {}
Until.__index = Until

function Until.new(predicate)
  return setmetatable({ predicate = predicate, event = true }, Until)
end

function Until:advance()
  if self.predicate() then
    return true, 0
  end
  return false, 0
end

-- Wait for `scheduler:emit(name, ...)`. The payload becomes the results.
local SignalWait = {}
SignalWait.__index = SignalWait

function SignalWait:advance()
  if self.payload then
    return true, 0, unpack(self.payload, 1, self.payload.n)
  end
  return false, 0
end

-- Wait for another task to finish; resumes with that task's return values.
local Join = {}
Join.__index = Join

function Join.new(task)
  return setmetatable({ task = task, event = true }, Join)
end

function Join:advance()
  local task = self.task
  if task.status == "dead" then
    return true, 0, unpack(task.results, 1, task.results.n)
  elseif task.status == "cancelled" then
    return true, 0
  end
  return false, 0
end

Scheduler.Delay = Delay
Scheduler.Frame = Frame
Scheduler.Until = Until
Scheduler.Join = Join

---------------------------------------------------------------------------
-- Tasks
---------------------------------------------------------------------------

-- task.status is "running" (alive, possibly waiting), "dead" (returned
-- normally) or "cancelled".
function Task:cancel()
  self.scheduler:cancel(self)
end

---------------------------------------------------------------------------
-- Scheduler
---------------------------------------------------------------------------

function Scheduler.new()
  return setmetatable({
    tasks = {}, -- in spawn order; also the update order
    by_co = {}, -- coroutine -> task, to find "the current task"
    waiters = {}, -- signal name -> list of SignalWait
    time = 0,
    next_id = 1,
  }, Scheduler)
end

-- Runs the task's `defer`red cleanups, newest first, exactly once.
local function run_cleanups(task)
  local cleanups = task.cleanups
  task.cleanups = {}
  for i = #cleanups, 1, -1 do
    cleanups[i]()
  end
end

-- A task that returns normally hands its still-running children to its own
-- parent, so ownership is never lost: cancelling the director still reaches
-- an enemy spawned by a short-lived helper task that the director started.
local function adopt_orphans(task)
  local parent = task.parent
  for _, child in ipairs(task.children) do
    if child.status == "running" then
      child.parent = parent
      if parent then
        parent.children[#parent.children + 1] = child
      end
    end
  end
  task.children = {}
end

local function finish(self, task, status)
  task.status = status
  task.awaiting = nil
  self.by_co[task.co] = nil
  if status == "dead" then
    adopt_orphans(task)
  end
  run_cleanups(task)
end

-- Resumes `task` with `args`, then keeps advancing each newly yielded
-- awaitable by the carried time until one of them needs to wait.
local function step(self, task, carry, args)
  while true do
    task.carry = carry -- inherited by tasks spawned during this slice
    local out = pack(coroutine.resume(task.co, unpack(args, 1, args.n)))
    if task.status ~= "running" then
      return -- cancelled itself (or by a child) during this slice
    end
    if not out[1] then
      local message = debug.traceback(task.co, tostring(out[2]))
      finish(self, task, "dead")
      error("task '" .. task.name .. "' failed: " .. message, 0)
    end
    if coroutine.status(task.co) == "dead" then
      task.results = pack(unpack(out, 2, out.n))
      finish(self, task, "dead")
      return
    end
    local awaitable = out[2]
    if type(awaitable) ~= "table" or type(awaitable.advance) ~= "function" then
      finish(self, task, "dead")
      error("task '" .. task.name .. "' yielded a non-awaitable (" .. type(awaitable) .. ")", 0)
    end
    local result = pack(awaitable:advance(carry))
    if not result[1] then
      task.awaiting = awaitable
      return
    end
    carry = result[2]
    args = pack(unpack(result, 3, result.n))
  end
end

--- Starts `fn(...)` as a new task and runs it until its first yield, so a
-- spawned script takes effect immediately (inheriting any time its parent
-- still has left in this frame). A task spawned from inside another
-- task becomes its child: cancelling the parent cancels the child.
-- `name` is optional and only used in errors and the debug overlay.
function Scheduler:spawn(fn, name, ...)
  if type(fn) ~= "function" then
    error("spawn expects a function", 2)
  end
  local id = self.next_id
  self.next_id = id + 1
  local task = setmetatable({
    id = id,
    name = name or ("task#" .. id),
    co = coroutine.create(fn),
    status = "running",
    scheduler = self,
    children = {},
    cleanups = {},
    results = pack(),
  }, Task)
  local parent = self:current()
  if parent then
    task.parent = parent
    parent.children[#parent.children + 1] = task
  end
  self.tasks[#self.tasks + 1] = task
  self.by_co[task.co] = task
  -- A task spawned right after its parent's wait finished mid-frame starts
  -- with the parent's leftover time, so it isn't late by a fraction of a frame.
  step(self, task, parent and parent.carry or 0, pack(...))
  return task
end

--- The task whose coroutine is running right now, or nil outside tasks.
function Scheduler:current()
  local co = coroutine.running()
  return co and self.by_co[co] or nil
end

local function require_task(self, what)
  local task = self:current()
  if not task then
    error(what .. " can only be called from inside a scheduler task", 3)
  end
  return task
end

--- Pauses the current task until `awaitable` completes; returns its results.
function Scheduler:await(awaitable)
  require_task(self, "await")
  return coroutine.yield(awaitable)
end

--- Waits `seconds` of game time (must be > 0; use frame() for one frame).
function Scheduler:wait(seconds)
  if type(seconds) ~= "number" or not (seconds > 0) then -- "not >" also rejects NaN
    error("wait expects a positive number of seconds, got " .. tostring(seconds), 2)
  end
  require_task(self, "wait")
  return coroutine.yield(Delay.new(seconds))
end

--- Waits until the next update; returns that frame's dt.
function Scheduler:frame()
  require_task(self, "frame")
  return coroutine.yield(Frame.new())
end

function Scheduler:wait_until(predicate)
  if type(predicate) ~= "function" then
    error("wait_until expects a function", 2)
  end
  require_task(self, "wait_until")
  return coroutine.yield(Until.new(predicate))
end

--- Waits for emit(name, ...); returns the emitted values.
function Scheduler:wait_signal(name)
  require_task(self, "wait_signal")
  local wait = setmetatable({ event = true }, SignalWait)
  local list = self.waiters[name] or {}
  list[#list + 1] = wait
  self.waiters[name] = list
  return coroutine.yield(wait)
end

--- Wakes every task currently waiting on `name`, within the current update
-- (or the next one when emitted from outside the scheduler).
function Scheduler:emit(name, ...)
  local list = self.waiters[name]
  if not list then
    return 0
  end
  self.waiters[name] = nil
  local payload = pack(...)
  for _, wait in ipairs(list) do
    wait.payload = payload
  end
  return #list
end

--- Tweens fields of `target` and waits for it to finish (see core.tween).
function Scheduler:tween(target, duration, props, easing)
  require_task(self, "tween")
  return coroutine.yield(Tween.new(target, duration, props, easing))
end

function Scheduler:join(task)
  require_task(self, "join")
  return coroutine.yield(Join.new(task))
end

--- Runs each function as a child task and waits for all of them. Returns a
-- list with the first return value of each.
function Scheduler:all(fns)
  require_task(self, "all")
  local children = {}
  for i, fn in ipairs(fns) do
    children[i] = self:spawn(fn, "all#" .. i)
  end
  local results = {}
  for i, child in ipairs(children) do
    results[i] = self:join(child)
  end
  return results
end

--- Runs each function as a child task until the first one finishes, cancels
-- the rest, and returns the winner's index and return values. This is how a
-- boss runs an attack loop "until health drops below half".
function Scheduler:race(fns)
  require_task(self, "race")
  local children = {}
  for i, fn in ipairs(fns) do
    children[i] = self:spawn(fn, "race#" .. i)
  end
  local winner
  self:wait_until(function()
    for i, child in ipairs(children) do
      if child.status == "dead" then
        winner = i
        return true
      end
    end
    return false
  end)
  for i, child in ipairs(children) do
    if i ~= winner then
      self:cancel(child)
    end
  end
  local results = children[winner].results
  return winner, unpack(results, 1, results.n)
end

--- Registers `fn` to run when the current task ends, whether it returns or is
-- cancelled. A plain coroutine that is never resumed again cannot clean up
-- after itself, so this is the scheduler's equivalent of `finally`.
function Scheduler:defer(fn)
  local task = require_task(self, "defer")
  task.cleanups[#task.cleanups + 1] = fn
end

--- Stops a task and, recursively, all of its children. Safe to call on a
-- finished task, and from inside the task itself (it stops at its next yield).
function Scheduler:cancel(task)
  if task.status ~= "running" then
    return
  end
  for _, child in ipairs(task.children) do
    self:cancel(child)
  end
  finish(self, task, "cancelled")
end

function Scheduler:cancel_all()
  for _, task in ipairs(self.tasks) do
    self:cancel(task)
  end
  self.tasks = {}
  self.waiters = {}
end

--- One-shot timer: calls fn after `delay` seconds. Returns the task, so the
-- caller can cancel it.
function Scheduler:after(delay, fn, name)
  return self:spawn(function()
    self:wait(delay)
    fn()
  end, name or "after")
end

--- Repeating timer: calls fn(i) every `interval` seconds, `count` times (or
-- forever when count is nil). Drift-free thanks to carried leftover time.
function Scheduler:every(interval, fn, count, name)
  if count ~= nil and count < 1 then
    error("every: count must be >= 1", 2)
  end
  return self:spawn(function()
    local i = 0
    while count == nil or i < count do
      self:wait(interval)
      i = i + 1
      fn(i)
    end
  end, name or "every")
end

--- Advances every live task by dt. Tasks run in spawn order; tasks spawned
-- during this pass already ran their first slice and wait for the next pass.
function Scheduler:update(dt)
  if type(dt) ~= "number" or dt < 0 then
    error("update expects dt >= 0", 2)
  end
  self.time = self.time + dt
  local snapshot = {}
  for i, task in ipairs(self.tasks) do
    snapshot[i] = task
  end
  for _, task in ipairs(snapshot) do
    local awaitable = task.status == "running" and task.awaiting
    if awaitable then
      local result = pack(awaitable:advance(dt))
      if result[1] and task.status == "running" then
        task.awaiting = nil
        step(self, task, result[2], pack(unpack(result, 3, result.n)))
      end
    end
  end
  self:settle()
  local live = {}
  for _, task in ipairs(self.tasks) do
    if task.status == "running" then
      live[#live + 1] = task
    else
      -- Drop finished children so long-lived parents don't accumulate them.
      local parent = task.parent
      if parent then
        for i = #parent.children, 1, -1 do
          if parent.children[i] == task then
            table.remove(parent.children, i)
          end
        end
      end
    end
  end
  self.tasks = live
end

-- Re-checks event awaitables until no task wakes. Each wake runs some script
-- code forward, so this terminates unless a script loops on conditions that
-- are already true (which would hang as a plain loop too).
function Scheduler:settle()
  local woke = true
  while woke do
    woke = false
    for _, task in ipairs(self.tasks) do
      local awaitable = task.status == "running" and task.awaiting
      if awaitable and awaitable.event then
        local result = pack(awaitable:advance(0))
        if result[1] and task.status == "running" then
          task.awaiting = nil
          step(self, task, 0, pack(unpack(result, 3, result.n)))
          woke = true
        end
      end
    end
  end
end

function Scheduler:count()
  return #self.tasks
end

--- Names of live tasks, for the debug overlay.
function Scheduler:names()
  local names = {}
  for i, task in ipairs(self.tasks) do
    names[i] = task.name
  end
  return names
end

return Scheduler
