local StateStack = require("core.state_stack")

-- A state that records every hook call into a shared log.
local function recorder(name, log, extra)
  local state = { name = name }
  for _, hook in ipairs({ "enter", "exit", "pause", "resume", "update", "handle" }) do
    state[hook] = function(self, arg)
      local entry = name .. ":" .. hook
      if hook == "resume" or hook == "handle" then
        entry = entry .. "(" .. tostring(arg) .. ")"
      end
      log[#log + 1] = entry
      if hook == "enter" then
        self.stack = arg
      end
    end
  end
  state.view = function() return { kind = name } end
  for k, v in pairs(extra or {}) do
    state[k] = v
  end
  return state
end

local function clear(t)
  for i = #t, 1, -1 do t[i] = nil end
end

local function kinds(views)
  local out = {}
  for i, v in ipairs(views) do out[i] = v.kind end
  return out
end

describe("StateStack", function()
  local stack, log

  before_each(function()
    stack = StateStack.new()
    log = {}
  end)

  it("starts empty", function()
    assert.is_true(stack:is_empty())
    assert.is_nil(stack:top())
    assert.same({}, stack:views())
  end)

  it("pauses the state below on push and resumes it with the pop result", function()
    stack:push(recorder("a", log))
    stack:push(recorder("b", log))
    stack:pop("done")
    assert.same({ "a:enter", "a:pause", "b:enter", "b:exit", "a:resume(done)" }, log)
    assert.same({ "a" }, stack:names())
  end)

  it("replace swaps the top without pausing or resuming the state below", function()
    stack:push(recorder("a", log))
    stack:push(recorder("b", log))
    stack:replace(recorder("c", log))
    assert.same({ "a:enter", "a:pause", "b:enter", "b:exit", "c:enter" }, log)
    assert.same({ "a", "c" }, stack:names())
  end)

  it("reset exits every state top-first and starts over", function()
    stack:push(recorder("a", log))
    stack:push(recorder("b", log))
    clear(log)
    stack:reset(recorder("c", log))
    assert.same({ "b:exit", "a:exit", "c:enter" }, log)
    assert.same({ "c" }, stack:names())
  end)

  it("defers transitions requested during update until the update finishes", function()
    local below = recorder("below", log)
    local top = recorder("top", log, {
      update_below = true,
      update = function(self)
        log[#log + 1] = "top:update"
        self.stack:pop()
        -- Still on the stack until the frame's update pass is over.
        log[#log + 1] = "size=" .. self.stack:size()
      end,
    })
    stack:push(below)
    stack:push(top)
    clear(log)
    stack:update(0.016)
    assert.same({ "top:update", "size=2", "below:update", "top:exit", "below:resume(nil)" }, log)
  end)

  it("updates only the top state unless it opts into update_below", function()
    stack:push(recorder("a", log))
    stack:push(recorder("b", log))
    clear(log)
    stack:update(1)
    assert.same({ "b:update" }, log)
  end)

  it("routes input to the top state only", function()
    stack:push(recorder("a", log))
    stack:push(recorder("b", log))
    clear(log)
    stack:handle("confirm")
    assert.same({ "b:handle(confirm)" }, log)
  end)

  it("draws down to the first opaque state, bottom to top", function()
    stack:push(recorder("menu", log))
    stack:push(recorder("play", log))
    stack:push(recorder("pause", log, { transparent = true }))
    stack:push(recorder("confirm", log, { transparent = true }))
    assert.same({ "play", "pause", "confirm" }, kinds(stack:views()))
  end)

  it("applies transitions queued by lifecycle hooks in request order", function()
    local a = recorder("a", log)
    a.resume = function(self, result)
      log[#log + 1] = "a:resume(" .. tostring(result) .. ")"
      if result == "chain" then
        self.stack:push(recorder("c", log))
      end
    end
    stack:push(a)
    stack:push(recorder("b", log))
    stack:pop("chain")
    assert.same({ "a", "c" }, stack:names())
    assert.same("c:enter", log[#log])
  end)

  it("errors when popping an empty stack", function()
    assert.has_error(function() stack:pop() end, "pop on an empty state stack")
  end)

  it("rejects states without a name", function()
    assert.has_error(function() stack:push({}) end)
    assert.has_error(function() stack:push("menu") end)
  end)

  it("detects states that bounce forever and stays usable afterwards", function()
    local bouncer
    bouncer = {
      name = "bouncer",
      enter = function(self, s) s:replace({ name = "bouncer", enter = bouncer.enter }) end,
    }
    assert.has_error(function() stack:push(bouncer) end)
    stack:reset(recorder("ok", log))
    assert.same({ "ok" }, stack:names())
  end)

  it("tolerates states that implement no hooks at all", function()
    stack:push({ name = "bare" })
    stack:update(1)
    stack:handle("confirm")
    assert.same({}, stack:views())
    stack:pop()
    assert.is_true(stack:is_empty())
  end)
end)
