--- Tweens: animate numeric fields of a table over time with an easing curve.
--
-- A Tween is a scheduler awaitable, so a script can `sched:tween(...)` and
-- carry on when it finishes, which is what makes tween *sequences* trivial:
-- three tweens in a row are three lines of code. Start values are captured on
-- the first advance, not at construction, so each step of a sequence starts
-- from wherever the previous one left the object.

local Tween = {}
Tween.__index = Tween

-- Same rounding tolerance as Scheduler.EPSILON (see there).
Tween.EPSILON = 1e-9

local PI = math.pi

--- Easing curves map normalized time t in [0, 1] to progress (0 at t=0,
-- 1 at t=1). "Out" curves start fast and settle, which reads as responsive;
-- "in" curves start slow, which reads as wind-up; "back" and "elastic"
-- overshoot for a springy, cartoony feel.
Tween.easing = {
  linear = function(t)
    return t
  end,
  quad_in = function(t)
    return t * t
  end,
  quad_out = function(t)
    return t * (2 - t)
  end,
  quad_in_out = function(t)
    if t < 0.5 then
      return 2 * t * t
    end
    return -1 + (4 - 2 * t) * t
  end,
  cubic_out = function(t)
    local u = t - 1
    return u * u * u + 1
  end,
  sine_in_out = function(t)
    return -(math.cos(PI * t) - 1) / 2
  end,
  back_out = function(t)
    local c1 = 1.70158
    local c3 = c1 + 1
    local u = t - 1
    return 1 + c3 * u * u * u + c1 * u * u
  end,
  elastic_out = function(t)
    if t == 0 or t == 1 then
      return t
    end
    return 2 ^ (-10 * t) * math.sin((t * 10 - 0.75) * (2 * PI / 3)) + 1
  end,
}

local function resolve_easing(easing)
  if easing == nil then
    return Tween.easing.linear
  elseif type(easing) == "function" then
    return easing
  end
  local fn = Tween.easing[easing]
  if not fn then
    error("unknown easing '" .. tostring(easing) .. "'", 3)
  end
  return fn
end

--- target: table to modify; duration: seconds (> 0); props: field -> goal
-- value; easing: name from Tween.easing or a function (default linear).
function Tween.new(target, duration, props, easing)
  if type(target) ~= "table" then
    error("tween target must be a table", 2)
  end
  if type(duration) ~= "number" or not (duration > 0) then -- rejects NaN too
    error("tween duration must be a positive number", 2)
  end
  if type(props) ~= "table" or next(props) == nil then
    error("tween needs at least one property", 2)
  end
  for key, goal in pairs(props) do
    if type(goal) ~= "number" then
      error("tween goal for '" .. tostring(key) .. "' must be a number", 2)
    end
  end
  return setmetatable({
    target = target,
    duration = duration,
    props = props,
    ease = resolve_easing(easing),
    elapsed = 0,
    from = nil,
  }, Tween)
end

local function capture(self)
  local from = {}
  for key in pairs(self.props) do
    local value = self.target[key]
    if type(value) ~= "number" then
      error("tween: target field '" .. tostring(key) .. "' is not a number", 0)
    end
    from[key] = value
  end
  self.from = from
end

--- Awaitable protocol: returns done, leftover. The final step snaps to the
-- exact goal values so floating-point error never accumulates in sequences.
function Tween:advance(dt)
  if not self.from then
    capture(self)
  end
  self.elapsed = self.elapsed + dt
  if self.elapsed >= self.duration - Tween.EPSILON then
    for key, goal in pairs(self.props) do
      self.target[key] = goal
    end
    return true, math.max(0, self.elapsed - self.duration)
  end
  local k = self.ease(self.elapsed / self.duration)
  for key, goal in pairs(self.props) do
    local from = self.from[key]
    self.target[key] = from + (goal - from) * k
  end
  return false, 0
end

return Tween
