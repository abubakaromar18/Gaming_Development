--- Draws the views produced by StateStack:views(), bottom to top. Each view is
-- plain data with a `kind`; this module is the only place that knows how a
-- kind looks. Everything is procedural: rectangles, circles and LÖVE's
-- built-in font.

local Renderer = {}
Renderer.__index = Renderer

local COLORS = {
  background = { 0.07, 0.08, 0.10 },
  board = { 0.11, 0.13, 0.16 },
  grid = { 0.14, 0.16, 0.20 },
  snake = { 0.36, 0.80, 0.45 },
  head = { 0.55, 0.95, 0.60 },
  eye = { 0.05, 0.08, 0.06 },
  food = { 0.95, 0.35, 0.35 },
  text = { 0.92, 0.94, 0.96 },
  dim_text = { 0.55, 0.60, 0.66 },
  accent = { 0.98, 0.80, 0.30 },
  shade = { 0, 0, 0, 0.6 },
}

local HUD_HEIGHT = 40

function Renderer.new()
  return setmetatable({
    fonts = {
      small = love.graphics.newFont(14),
      medium = love.graphics.newFont(22),
      large = love.graphics.newFont(48),
    },
    time = 0,
  }, Renderer)
end

function Renderer:update(dt)
  self.time = self.time + dt
end

local function set(color)
  love.graphics.setColor(color)
end

function Renderer:text(str, font, color, y)
  local w = love.graphics.getWidth()
  love.graphics.setFont(self.fonts[font])
  set(color)
  love.graphics.printf(str, 0, y, w, "center")
end

function Renderer:shade()
  set(COLORS.shade)
  love.graphics.rectangle("fill", 0, 0, love.graphics.getDimensions())
end

function Renderer:menu(menu, y)
  local step = self.fonts.medium:getHeight() + 10
  for i, label in ipairs(menu.labels) do
    local selected = i == menu.index
    local str = selected and ("> " .. label .. " <") or label
    self:text(str, "medium", selected and COLORS.accent or COLORS.dim_text,
      y + (i - 1) * step)
  end
end

-- Fit the board under the HUD, keeping cells square.
local function layout(view)
  local w, h = love.graphics.getDimensions()
  local cell = math.floor(math.min(w / view.cols, (h - HUD_HEIGHT) / view.rows))
  local bw, bh = cell * view.cols, cell * view.rows
  return cell, math.floor((w - bw) / 2), HUD_HEIGHT + math.floor((h - HUD_HEIGHT - bh) / 2)
end

local EYE_OFFSETS = {
  right = { { 0.7, 0.3 }, { 0.7, 0.7 } },
  left = { { 0.3, 0.3 }, { 0.3, 0.7 } },
  up = { { 0.3, 0.3 }, { 0.7, 0.3 } },
  down = { { 0.3, 0.7 }, { 0.7, 0.7 } },
}

Renderer.draw_kind = {}

function Renderer.draw_kind.menu(self, view)
  local h = love.graphics.getHeight()
  self:text(view.title, "large", COLORS.head, h * 0.22)
  self:menu(view.menu, h * 0.45)
  local scores = "Best " .. view.best
  if view.last then
    scores = "Last " .. view.last .. "   " .. scores
  end
  self:text(scores, "small", COLORS.dim_text, h * 0.75)
  self:text("Arrows/WASD move  |  Enter select  |  P pause  |  F3 stack",
    "small", COLORS.dim_text, h - 30)
end

function Renderer.draw_kind.play(self, view)
  local cell, ox, oy = layout(view)
  set(COLORS.board)
  love.graphics.rectangle("fill", ox, oy, cell * view.cols, cell * view.rows)
  set(COLORS.grid)
  for x = 1, view.cols - 1 do
    love.graphics.line(ox + x * cell, oy, ox + x * cell, oy + cell * view.rows)
  end
  for y = 1, view.rows - 1 do
    love.graphics.line(ox, oy + y * cell, ox + cell * view.cols, oy + y * cell)
  end

  if view.food then
    local pulse = 0.38 + 0.06 * math.sin(self.time * 6)
    set(COLORS.food)
    love.graphics.circle("fill", ox + (view.food.x + 0.5) * cell,
      oy + (view.food.y + 0.5) * cell, cell * pulse)
  end

  local pad = math.max(1, math.floor(cell * 0.08))
  for i = #view.body, 1, -1 do
    local seg = view.body[i]
    set(i == 1 and COLORS.head or COLORS.snake)
    love.graphics.rectangle("fill", ox + seg.x * cell + pad, oy + seg.y * cell + pad,
      cell - 2 * pad, cell - 2 * pad, pad * 2)
  end
  local head = view.body[1]
  set(COLORS.eye)
  for _, eye in ipairs(EYE_OFFSETS[view.dir]) do
    love.graphics.circle("fill", ox + (head.x + eye[1]) * cell,
      oy + (head.y + eye[2]) * cell, math.max(1.5, cell * 0.09))
  end

  love.graphics.setFont(self.fonts.medium)
  set(COLORS.text)
  love.graphics.print("Score " .. view.score, 16, 8)
  set(COLORS.dim_text)
  local best = "Best " .. view.best
  love.graphics.print(best, love.graphics.getWidth() - 16 - self.fonts.medium:getWidth(best), 8)
end

function Renderer.draw_kind.countdown(self, view)
  self:text(tostring(view.value), "large", COLORS.accent,
    love.graphics.getHeight() * 0.4)
end

function Renderer.draw_kind.pause(self, view)
  self:shade()
  local h = love.graphics.getHeight()
  self:text(view.title, "large", COLORS.text, h * 0.25)
  self:menu(view.menu, h * 0.45)
end

function Renderer.draw_kind.game_over(self, view)
  self:shade()
  local h = love.graphics.getHeight()
  self:text(view.title, "large", COLORS.food, h * 0.2)
  self:text("Score " .. view.score .. "   Best " .. view.best, "medium",
    COLORS.text, h * 0.36)
  if view.new_best then
    self:text("New best!", "medium", COLORS.accent, h * 0.43)
  end
  self:menu(view.menu, h * 0.55)
end

--- Draw all visible views, plus an optional debug readout of the stack.
function Renderer:draw(views, stack_names)
  love.graphics.clear(COLORS.background)
  for _, view in ipairs(views) do
    local draw = Renderer.draw_kind[view.kind]
    if not draw then
      error("renderer has no drawing for view kind '" .. tostring(view.kind) .. "'")
    end
    draw(self, view)
  end
  if stack_names then
    love.graphics.setFont(self.fonts.small)
    set(COLORS.accent)
    love.graphics.print("stack: " .. table.concat(stack_names, " > "), 16,
      love.graphics.getHeight() - 24)
  end
end

return Renderer
