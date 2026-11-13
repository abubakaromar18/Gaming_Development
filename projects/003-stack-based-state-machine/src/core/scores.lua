--- Session and best scores. Persistence is the platform layer's job: it reads
--- the stored best into new() and saves when record() reports a new best.

local Scores = {}
Scores.__index = Scores

function Scores.new(best)
  best = tonumber(best) or 0
  if best < 0 or best ~= math.floor(best) then
    best = 0 -- corrupt save data: start fresh rather than crash
  end
  return setmetatable({ best = best, last = nil }, Scores)
end

--- Record a finished round. Returns true when it set a new best.
function Scores:record(score)
  self.last = score
  if score > self.best then
    self.best = score
    return true
  end
  return false
end

return Scores
