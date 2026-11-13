--- Best-score persistence in LÖVE's per-game save directory. Any failure
--- (missing file, unreadable data, read-only disk) degrades to "no saved
--- score" instead of crashing the game.

local Storage = {}

local FILE = "best_score.txt"

function Storage.load_best()
  if not love.filesystem.getInfo(FILE) then
    return 0
  end
  local contents = love.filesystem.read(FILE)
  return tonumber(contents) or 0
end

function Storage.save_best(best)
  local ok, err = love.filesystem.write(FILE, tostring(best))
  if not ok then
    print("could not save best score: " .. tostring(err))
  end
end

return Storage
