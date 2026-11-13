--- Seeded pseudo-random generator (Park–Miller "minimal standard", MINSTD).
--
-- The game never touches math.random: the RNG is created from a seed and
-- injected, so a test (or a replay) with the same seed and inputs sees the
-- same food positions. MINSTD is tiny and its products stay below 2^53, so it
-- gives identical sequences on LuaJIT (LÖVE), Lua 5.1 and Lua 5.4.

local Rng = {}
Rng.__index = Rng

local MODULUS = 2147483647 -- 2^31 - 1
local MULTIPLIER = 48271

function Rng.new(seed)
  if type(seed) ~= "number" or seed ~= seed then
    error("seed must be a number", 2)
  end
  local state = math.floor(math.abs(seed)) % MODULUS
  if state == 0 then
    state = 1 -- zero is a fixed point of the generator
  end
  return setmetatable({ state = state }, Rng)
end

--- Next float in [0, 1).
function Rng:next()
  self.state = (self.state * MULTIPLIER) % MODULUS
  return (self.state - 1) / (MODULUS - 1)
end

--- Uniform integer in [lo, hi], inclusive.
function Rng:int(lo, hi)
  if hi < lo then
    error("empty range [" .. lo .. ", " .. hi .. "]", 2)
  end
  return lo + math.floor(self:next() * (hi - lo + 1))
end

return Rng
