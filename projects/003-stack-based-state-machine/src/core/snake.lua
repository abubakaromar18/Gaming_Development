--- Snake rules: grid movement on a fixed tick, buffered turns, growth, food.
--
-- This module knows nothing about screens or states. The Play state owns one
-- Snake and turns its events ("ate", "died", "won") into stack transitions.

local Snake = {}
Snake.__index = Snake

Snake.DIRECTIONS = {
  up = { x = 0, y = -1 },
  down = { x = 0, y = 1 },
  left = { x = -1, y = 0 },
  right = { x = 1, y = 0 },
}

local OPPOSITE = { up = "down", down = "up", left = "right", right = "left" }

--- @param cfg table built by core.config
--- @param rng table core.rng instance (food placement)
function Snake.new(cfg, rng)
  local self = setmetatable({ cfg = cfg, rng = rng }, Snake)
  self:reset()
  return self
end

function Snake:key(x, y)
  return y * self.cfg.cols + x
end

--- Start a fresh round: snake centred, heading right, new food.
function Snake:reset()
  local cfg = self.cfg
  local hx = math.floor(cfg.cols / 2)
  local hy = math.floor(cfg.rows / 2)
  self.body = {}     -- head first
  self.occupied = {} -- cell key -> true, for O(1) collision checks
  for i = 0, cfg.start_length - 1 do
    local seg = { x = hx - i, y = hy }
    self.body[#self.body + 1] = seg
    self.occupied[self:key(seg.x, seg.y)] = true
  end
  self.dir = "right"
  self.turns = {}
  self.score = 0
  self.status = "alive" -- "alive" | "dead" | "won"
  self.accumulator = 0
  self.steps = 0
  self:spawn_food()
end

function Snake:head()
  return self.body[1]
end

function Snake:length()
  return #self.body
end

function Snake:is_alive()
  return self.status == "alive"
end

--- Seconds per move; the snake speeds up as it eats.
function Snake:tick_interval()
  local cfg = self.cfg
  return math.max(cfg.min_tick, cfg.base_tick * cfg.speedup ^ self.score)
end

--- Queue a turn. Turns are buffered so that a quick "up, left" between two
--- ticks performs both moves instead of dropping the second key. A turn is
--- checked against the last *queued* direction, so no sequence of fast taps
--- can reverse the snake into its own neck.
--- @return boolean whether the turn was accepted
function Snake:steer(dir)
  if not Snake.DIRECTIONS[dir] then
    error("unknown direction '" .. tostring(dir) .. "'", 2)
  end
  if not self:is_alive() or #self.turns >= self.cfg.turn_buffer then
    return false
  end
  local last = self.turns[#self.turns] or self.dir
  if dir == last or dir == OPPOSITE[last] then
    return false
  end
  self.turns[#self.turns + 1] = dir
  return true
end

--- Place food on a uniformly random free cell, or nowhere if the board is full.
function Snake:spawn_food()
  local free = {}
  for y = 0, self.cfg.rows - 1 do
    for x = 0, self.cfg.cols - 1 do
      if not self.occupied[self:key(x, y)] then
        free[#free + 1] = { x = x, y = y }
      end
    end
  end
  if #free == 0 then
    self.food = nil
  else
    self.food = free[self.rng:int(1, #free)]
  end
end

--- Advance exactly one cell. Returns an event name or nil.
function Snake:step()
  if not self:is_alive() then
    return nil
  end
  local cfg = self.cfg
  if #self.turns > 0 then
    self.dir = table.remove(self.turns, 1)
  end
  local d = Snake.DIRECTIONS[self.dir]
  local head = self.body[1]
  local nx, ny = head.x + d.x, head.y + d.y

  if cfg.wrap then
    nx, ny = nx % cfg.cols, ny % cfg.rows
  elseif nx < 0 or ny < 0 or nx >= cfg.cols or ny >= cfg.rows then
    self.status = "dead"
    return "died"
  end

  local grows = self.food ~= nil and nx == self.food.x and ny == self.food.y
  local tail = self.body[#self.body]
  local target = self:key(nx, ny)
  -- Moving into the cell the tail is leaving this tick is legal.
  local tail_vacates = not grows and target == self:key(tail.x, tail.y)
  if self.occupied[target] and not tail_vacates then
    self.status = "dead"
    return "died"
  end

  if not grows then
    self.body[#self.body] = nil
    self.occupied[self:key(tail.x, tail.y)] = nil
  end
  table.insert(self.body, 1, { x = nx, y = ny })
  self.occupied[target] = true
  self.steps = self.steps + 1

  if grows then
    self.score = self.score + 1
    self:spawn_food()
    if self.food == nil then
      self.status = "won"
      return "won"
    end
    return "ate"
  end
  return nil
end

--- Accumulate dt and move once per tick interval. Catch-up after a long frame
--- is capped so a hitch never teleports the snake several cells into a wall.
--- @return table list of events that happened this update, in order
function Snake:update(dt)
  local events = {}
  if not self:is_alive() then
    return events
  end
  self.accumulator = self.accumulator + dt
  local moves = 0
  while self:is_alive() and self.accumulator >= self:tick_interval() do
    if moves >= self.cfg.max_steps_per_update then
      self.accumulator = 0
      break
    end
    self.accumulator = self.accumulator - self:tick_interval()
    moves = moves + 1
    local event = self:step()
    if event then
      events[#events + 1] = event
    end
  end
  return events
end

return Snake
