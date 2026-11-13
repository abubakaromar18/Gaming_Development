--- A pushdown automaton for game screens.
--
-- A flat state machine has one current state; a *pushdown* automaton keeps a
-- stack of them. Pushing a state suspends the one below instead of destroying
-- it, and popping returns control to it, optionally with a result value, much
-- like returning from a function call. That is exactly how game screens
-- behave: opening the pause menu must not throw away the round in progress,
-- and closing it must resume that round where it left off.
--
-- Lifecycle hooks (all optional on a state):
--   state:enter(stack)        pushed on the stack (or switched in via replace)
--   state:exit()              removed from the stack
--   state:pause()             another state was pushed on top of it
--   state:resume(result)      the state above was popped; result is what it
--                             returned via stack:pop(result)
--   state:update(dt)          per-frame simulation
--   state:handle(action)      an abstract input action (top state only)
--   state:view()              plain-data description for the renderer
--
-- Flags (optional, default false):
--   state.transparent         the renderer also draws the state below
--   state.update_below        the state below keeps updating too
--
-- Transitions requested while the stack is busy (inside update, handle, or a
-- lifecycle hook) are queued and applied afterwards in request order. Mutating
-- a stack while iterating over it is a classic source of "state updated after
-- it was popped" bugs, and deferring makes that impossible.

local StateStack = {}
StateStack.__index = StateStack

--- Upper bound on transitions applied in one commit. A state whose enter hook
-- pushes another state whose enter hook pops back, forever, is a programming
-- error; failing loudly beats freezing the game.
local MAX_TRANSITIONS_PER_COMMIT = 64

local function call(state, hook, ...)
  local fn = state[hook]
  if fn then
    return fn(state, ...)
  end
end

local function validate_state(state)
  if type(state) ~= "table" then
    error("state must be a table, got " .. type(state), 3)
  end
  if type(state.name) ~= "string" or state.name == "" then
    error("state must have a non-empty string 'name'", 3)
  end
end

function StateStack.new()
  return setmetatable({
    states = {},
    pending = {},
    busy = 0,
  }, StateStack)
end

-- Queue a transition, or apply it right away when nothing is in progress.
function StateStack:request(op)
  self.pending[#self.pending + 1] = op
  if self.busy == 0 then
    self:commit()
  end
end

--- Push a state on top. The current top receives pause().
function StateStack:push(state)
  validate_state(state)
  self:request({ kind = "push", state = state })
end

--- Pop the top state. The new top receives resume(result).
function StateStack:pop(result)
  self:request({ kind = "pop", result = result })
end

--- Swap the top state for another without pausing/resuming the one below.
function StateStack:replace(state)
  validate_state(state)
  self:request({ kind = "replace", state = state })
end

--- Exit every state (top first) and start over with a single state.
function StateStack:reset(state)
  validate_state(state)
  self:request({ kind = "reset", state = state })
end

function StateStack:apply(op)
  local states = self.states
  if op.kind == "push" then
    local below = states[#states]
    if below then
      call(below, "pause")
    end
    states[#states + 1] = op.state
    call(op.state, "enter", self)
  elseif op.kind == "pop" then
    local top = states[#states]
    if not top then
      error("pop on an empty state stack", 0)
    end
    states[#states] = nil
    call(top, "exit")
    local below = states[#states]
    if below then
      call(below, "resume", op.result)
    end
  elseif op.kind == "replace" then
    local top = states[#states]
    if top then
      states[#states] = nil
      call(top, "exit")
    end
    states[#states + 1] = op.state
    call(op.state, "enter", self)
  elseif op.kind == "reset" then
    while #states > 0 do
      local top = states[#states]
      states[#states] = nil
      call(top, "exit")
    end
    states[1] = op.state
    call(op.state, "enter", self)
  else
    error("unknown transition '" .. tostring(op.kind) .. "'", 0)
  end
end

--- Apply every queued transition, including ones queued by the hooks that the
--- transitions themselves trigger.
function StateStack:commit()
  self.busy = self.busy + 1
  local applied = 0
  local ok, err = pcall(function()
    while #self.pending > 0 do
      applied = applied + 1
      if applied > MAX_TRANSITIONS_PER_COMMIT then
        error("more than " .. MAX_TRANSITIONS_PER_COMMIT
          .. " transitions in one commit; states are probably bouncing", 0)
      end
      local op = table.remove(self.pending, 1)
      self:apply(op)
    end
  end)
  self.busy = self.busy - 1
  if not ok then
    self.pending = {}
    error(err, 0)
  end
end

-- Run fn while the stack is marked busy, then flush queued transitions.
function StateStack:guarded(fn)
  self.busy = self.busy + 1
  local ok, err = pcall(fn)
  self.busy = self.busy - 1
  if not ok then
    error(err, 0)
  end
  if self.busy == 0 then
    self:commit()
  end
end

--- Update the top state and, while states allow it, the ones below it.
function StateStack:update(dt)
  self:guarded(function()
    local states = self.states
    for i = #states, 1, -1 do
      local state = states[i]
      call(state, "update", dt)
      if not state.update_below then
        break
      end
    end
  end)
end

--- Route an input action to the top state only. Lower states are suspended
--- and must not react to input meant for an overlay.
function StateStack:handle(action)
  self:guarded(function()
    local top = self.states[#self.states]
    if top then
      call(top, "handle", action)
    end
  end)
end

--- Views to draw, bottom to top: the topmost opaque state and everything
--- above it. States hidden under an opaque state are skipped entirely.
function StateStack:views()
  local states = self.states
  local first = #states
  while first > 1 and states[first].transparent do
    first = first - 1
  end
  local views = {}
  for i = math.max(first, 1), #states do
    local view = call(states[i], "view")
    if view then
      views[#views + 1] = view
    end
  end
  return views
end

function StateStack:top()
  return self.states[#self.states]
end

function StateStack:size()
  return #self.states
end

function StateStack:is_empty()
  return #self.states == 0
end

--- Names from bottom to top, e.g. { "menu", "play", "pause" }. Handy for
--- debugging overlays and for tests.
function StateStack:names()
  local names = {}
  for i, state in ipairs(self.states) do
    names[i] = state.name
  end
  return names
end

return StateStack
