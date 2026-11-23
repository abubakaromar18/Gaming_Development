--- Draws the game state with LÖVE primitives only: polygons, circles, lines
-- and the built-in font. The arena has a fixed logical size and is scaled to
-- fit the window with letterboxing, so the core never sees window pixels.
-- Purely cosmetic animation (star scroll) lives here, driven by its own time.

local World = require("core.world")

local Renderer = {}
Renderer.__index = Renderer

local COLORS = {
  void = { 0.02, 0.02, 0.04 },
  arena = { 0.04, 0.05, 0.09 },
  star = { 0.75, 0.8, 1.0 },
  player = { 0.45, 0.9, 1.0 },
  bullet = { 1.0, 0.95, 0.6 },
  enemy_bullet = { 1.0, 0.4, 0.55 },
  drone = { 0.95, 0.55, 0.25 },
  gunner = { 0.7, 0.5, 1.0 },
  boss = { 0.95, 0.3, 0.35 },
  shield = { 0.5, 0.8, 1.0 },
  text = { 0.92, 0.94, 0.97 },
  dim = { 0.55, 0.6, 0.7 },
  accent = { 1.0, 0.82, 0.3 },
}

local PARTICLE_COLORS = {
  explosion = { 1.0, 0.6, 0.25 },
  spark = { 1.0, 0.95, 0.7 },
  player = { 0.45, 0.9, 1.0 },
}

local STAR_COUNT = 90

function Renderer.new(width, height, seed)
  -- Presentation-only randomness gets its own generator so it never perturbs
  -- the simulation's RNG sequence.
  local random = love.math.newRandomGenerator(seed)
  local stars = {}
  for i = 1, STAR_COUNT do
    stars[i] = {
      x = random:random() * width,
      y = random:random() * height,
      speed = 20 + random:random() * 80,
      size = random:random() < 0.15 and 2 or 1,
    }
  end
  return setmetatable({
    width = width,
    height = height,
    stars = stars,
    time = 0,
    fonts = {
      small = love.graphics.newFont(12),
      medium = love.graphics.newFont(18),
      large = love.graphics.newFont(40),
    },
  }, Renderer)
end

function Renderer:update(dt)
  self.time = self.time + dt
end

local function color(c, alpha)
  love.graphics.setColor(c[1], c[2], c[3], alpha or 1)
end

-- Blends a base color toward white by `amount` (used for telegraph flashes).
local function flashed(c, amount)
  return { c[1] + (1 - c[1]) * amount, c[2] + (1 - c[2]) * amount, c[3] + (1 - c[3]) * amount }
end

function Renderer:draw_stars()
  for _, s in ipairs(self.stars) do
    local y = (s.y + s.speed * self.time) % self.height
    color(COLORS.star, s.size == 2 and 0.9 or 0.45)
    love.graphics.rectangle("fill", s.x, y, s.size, s.size)
  end
end

local function draw_player(p)
  if not p.alive or not p.visible then
    return
  end
  color(COLORS.player)
  local r = p.r
  love.graphics.polygon("fill", p.x, p.y - r * 1.6, p.x + r, p.y + r, p.x, p.y + r * 0.4, p.x - r, p.y + r)
end

local function draw_enemy(e, time)
  local base = flashed(COLORS[e.kind], e.flash)
  color(base)
  local r = e.r
  if e.kind == "drone" then
    love.graphics.polygon("fill", e.x, e.y + r, e.x + r, e.y, e.x, e.y - r, e.x - r, e.y)
  elseif e.kind == "gunner" then
    love.graphics.circle("fill", e.x, e.y, r, 6)
    color(COLORS.arena)
    love.graphics.circle("fill", e.x, e.y + r * 0.3, r * 0.35)
  else
    love.graphics.circle("fill", e.x, e.y, r, 10)
    color(COLORS.arena)
    love.graphics.circle("fill", e.x, e.y + 8, r * 0.35)
    -- Health bar under the boss.
    local w = 120
    color(COLORS.dim, 0.6)
    love.graphics.rectangle("fill", e.x - w / 2, e.y + r + 10, w, 5)
    color(base)
    love.graphics.rectangle("fill", e.x - w / 2, e.y + r + 10, w * e.hp / e.max_hp, 5)
  end
  if e.shielded then
    color(COLORS.shield, 0.35 + 0.2 * math.sin(time * 12))
    love.graphics.setLineWidth(2)
    love.graphics.circle("line", e.x, e.y, r + 6)
    love.graphics.setLineWidth(1)
  end
end

function Renderer:draw_world(game)
  local world = game.world
  self:draw_stars()
  for _, part in ipairs(world.particles) do
    color(PARTICLE_COLORS[part.kind], part.life / World.PARTICLE_LIFE)
    love.graphics.rectangle("fill", part.x - 1.5, part.y - 1.5, 3, 3)
  end
  for _, e in ipairs(world.enemies) do
    draw_enemy(e, self.time)
  end
  color(COLORS.bullet)
  for _, b in ipairs(world.bullets) do
    love.graphics.rectangle("fill", b.x - 1.5, b.y - 7, 3, 10)
  end
  color(COLORS.enemy_bullet)
  for _, b in ipairs(world.enemy_bullets) do
    love.graphics.circle("fill", b.x, b.y, b.r)
  end
  draw_player(world.player)
end

function Renderer:draw_hud(game, paused, debug_names)
  local w, h = self.width, self.height
  love.graphics.setFont(self.fonts.medium)
  color(COLORS.text)
  love.graphics.print(string.format("%07d", game.score), 12, 10)
  love.graphics.printf("WAVE " .. game.wave .. "/" .. game.wave_count, 0, 10, w - 12, "right")
  color(COLORS.player)
  for i = 1, game.lives do
    local x = 16 + (i - 1) * 18
    love.graphics.polygon("fill", x, h - 26, x + 6, h - 12, x - 6, h - 12)
  end
  if game.combo > 1 then
    color(COLORS.accent)
    love.graphics.printf(game.combo .. " chain  x" .. game:multiplier(), 0, h - 28, w - 12, "right")
  end

  love.graphics.setFont(self.fonts.small)
  for _, popup in ipairs(game.popups) do
    color(COLORS.accent, popup.alpha)
    love.graphics.printf(popup.text, popup.x - 50, popup.y, 100, "center")
  end

  local banner = game.banner
  if banner.alpha > 0 then
    love.graphics.setFont(self.fonts.large)
    color(COLORS.text, banner.alpha)
    love.graphics.printf(banner.text, 0, banner.y, w, "center")
    if game.state ~= "playing" then
      love.graphics.setFont(self.fonts.medium)
      color(COLORS.dim, banner.alpha)
      love.graphics.printf("R to play again   Esc to quit", 0, banner.y + 56, w, "center")
    end
  end

  if paused then
    love.graphics.setFont(self.fonts.large)
    color(COLORS.text)
    love.graphics.printf("PAUSED", 0, h * 0.45, w, "center")
  end

  if debug_names then
    love.graphics.setFont(self.fonts.small)
    local lines = { "tasks: " .. #debug_names }
    local counts, order = {}, {}
    for _, name in ipairs(debug_names) do
      if not counts[name] then
        order[#order + 1] = name
      end
      counts[name] = (counts[name] or 0) + 1
    end
    for _, name in ipairs(order) do
      lines[#lines + 1] = string.format("  %-14s x%d", name, counts[name])
    end
    local text = table.concat(lines, "\n")
    local font = self.fonts.small
    local _, wrapped = font:getWrap(text, w)
    love.graphics.setColor(0, 0, 0, 0.55)
    love.graphics.rectangle("fill", 6, 36, 150, #wrapped * font:getHeight() + 8)
    color(COLORS.dim, 0.9)
    love.graphics.print(text, 12, 40)
  end
end

--- Draws one frame. `debug_names` (scheduler task names) enables the overlay.
function Renderer:draw(game, paused, debug_names)
  local ww, wh = love.graphics.getDimensions()
  local scale = math.min(ww / self.width, wh / self.height)
  local ox, oy = (ww - self.width * scale) / 2, (wh - self.height * scale) / 2
  love.graphics.clear(COLORS.void)
  love.graphics.push()
  love.graphics.translate(ox, oy)
  love.graphics.scale(scale)
  love.graphics.setScissor(ox, oy, self.width * scale, self.height * scale)
  color(COLORS.arena)
  love.graphics.rectangle("fill", 0, 0, self.width, self.height)
  self:draw_world(game)
  self:draw_hud(game, paused, debug_names)
  love.graphics.setScissor()
  love.graphics.pop()
end

return Renderer
