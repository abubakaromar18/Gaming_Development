--- "3, 2, 1" overlay. Transparent, so the frozen board shows underneath, and
--- it does not set update_below, so the snake stays still until it pops.

local Countdown = {}
Countdown.__index = Countdown

function Countdown.new(ctx)
  return setmetatable({
    name = "countdown",
    transparent = true,
    remaining = ctx.config.countdown,
  }, Countdown)
end

function Countdown:enter(stack)
  self.stack = stack
  if self.remaining <= 0 then
    self.stack:pop()
  end
end

function Countdown:update(dt)
  if self.remaining <= 0 then
    return -- already popping
  end
  self.remaining = self.remaining - dt
  if self.remaining <= 0 then
    self.stack:pop()
  end
end

function Countdown:view()
  return { kind = "countdown", value = math.max(1, math.ceil(self.remaining)) }
end

return Countdown
