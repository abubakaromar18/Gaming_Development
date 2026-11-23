--- The per-frame simulation: player movement and firing, bullets, particles,
-- and collisions. Deliberately *not* coroutine-driven: continuous physics is
-- a plain update(dt) loop, while discrete scripted sequences (enemy entrances,
-- attack patterns, waves) live in tasks that move the entities this module
-- owns. The world knows nothing about scripts; it reports what happened as
-- events and the game decides what they mean.

local World = {}
World.__index = World

--- Enemy archetypes. Behavior lives in core.behaviors; these are the stats.
World.KINDS = {
  drone = { radius = 11, hp = 1, score = 100 },
  gunner = { radius = 14, hp = 4, score = 250 },
  boss = { radius = 46, hp = 90, score = 5000 },
}

local PARTICLE_LIFE = 0.6
local atan2 = math.atan2 or math.atan -- Lua 5.3+ folded atan2 into atan(y, x)

function World.new(config, rng)
  local self = setmetatable({
    config = config,
    rng = rng,
    enemies = {},
    bullets = {},
    enemy_bullets = {},
    particles = {},
  }, World)
  self:reset_player()
  return self
end

function World:reset_player()
  local c = self.config
  self.player = {
    x = c.width / 2,
    y = c.height - 60,
    r = c.player_radius,
    alive = true,
    invulnerable = false,
    visible = true,
    locked = false, -- set by the respawn script while the ship flies in
    cooldown = 0,
  }
end

function World:spawn_enemy(kind, x, y)
  local stats = World.KINDS[kind]
  if not stats then
    error("unknown enemy kind '" .. tostring(kind) .. "'", 2)
  end
  local enemy = {
    kind = kind,
    x = x,
    y = y,
    r = stats.radius,
    hp = stats.hp,
    max_hp = stats.hp,
    score = stats.score,
    alive = true,
    shielded = false, -- scripts raise this during entrances and phase changes
    flash = 0, -- 0..1 telegraph glow, tweened by scripts
  }
  self.enemies[#self.enemies + 1] = enemy
  return enemy
end

--- Removes an enemy without scoring it (it flew off screen). Idempotent.
function World:remove_enemy(enemy)
  enemy.alive = false
  for i = #self.enemies, 1, -1 do
    if self.enemies[i] == enemy then
      table.remove(self.enemies, i)
    end
  end
end

function World:enemy_count()
  return #self.enemies
end

--- Fires an enemy bullet from (x, y) at `angle` radians (0 = right, pi/2 = down).
function World:fire(x, y, angle, speed)
  speed = speed or self.config.enemy_bullet_speed
  self.enemy_bullets[#self.enemy_bullets + 1] = {
    x = x,
    y = y,
    vx = math.cos(angle) * speed,
    vy = math.sin(angle) * speed,
    r = 4,
  }
end

--- Angle from (x, y) toward the player.
function World:aim(x, y)
  local p = self.player
  return atan2(p.y - y, p.x - x)
end

function World:burst(x, y, count, kind)
  local rng = self.rng
  for _ = 1, count do
    local angle = rng:range(0, 2 * math.pi)
    local speed = rng:range(40, 220)
    self.particles[#self.particles + 1] = {
      x = x,
      y = y,
      vx = math.cos(angle) * speed,
      vy = math.sin(angle) * speed,
      life = PARTICLE_LIFE,
      kind = kind,
    }
  end
end

local function overlaps(a, b)
  local dx, dy, r = a.x - b.x, a.y - b.y, a.r + b.r
  return dx * dx + dy * dy <= r * r
end

-- Moves a list of {x, y, vx, vy} and drops anything far outside the arena.
local function move_and_cull(list, dt, w, h)
  local margin = 32
  for i = #list, 1, -1 do
    local b = list[i]
    b.x = b.x + b.vx * dt
    b.y = b.y + b.vy * dt
    if b.x < -margin or b.x > w + margin or b.y < -margin or b.y > h + margin then
      table.remove(list, i)
    end
  end
end

local function update_player(self, dt, controls)
  local p, c = self.player, self.config
  if not p.alive then
    return
  end
  local mx, my = controls.move_x or 0, controls.move_y or 0
  local len = math.sqrt(mx * mx + my * my)
  if len > 1 then -- diagonals aren't faster
    mx, my = mx / len, my / len
  end
  p.x = math.max(p.r, math.min(c.width - p.r, p.x + mx * c.player_speed * dt))
  p.y = math.max(c.height * 0.45, math.min(c.height - p.r, p.y + my * c.player_speed * dt))

  -- Fire-rate accumulator: holding fire for t seconds yields exactly
  -- 1 + floor(t / interval) shots at any frame rate.
  p.cooldown = p.cooldown - dt
  if controls.fire then
    while p.cooldown <= 0 do
      self.bullets[#self.bullets + 1] = { x = p.x, y = p.y - p.r, vx = 0, vy = -c.bullet_speed, r = 3 }
      p.cooldown = p.cooldown + c.fire_interval
    end
  elseif p.cooldown < 0 then
    p.cooldown = 0
  end
end

local function hit_enemies(self, events)
  for bi = #self.bullets, 1, -1 do
    local bullet = self.bullets[bi]
    for ei = #self.enemies, 1, -1 do
      local enemy = self.enemies[ei]
      if overlaps(bullet, enemy) then
        table.remove(self.bullets, bi)
        if enemy.shielded then
          self:burst(bullet.x, bullet.y, 2, "spark")
        else
          enemy.hp = enemy.hp - 1
          if enemy.hp <= 0 then
            table.remove(self.enemies, ei)
            enemy.alive = false
            self:burst(enemy.x, enemy.y, enemy.kind == "boss" and 60 or 14, "explosion")
            events[#events + 1] = { kind = "enemy_killed", enemy = enemy }
          else
            self:burst(bullet.x, bullet.y, 3, "spark")
          end
        end
        break
      end
    end
  end
end

local function hit_player(self, events)
  local p = self.player
  if not p.alive or p.invulnerable then
    return
  end
  local hit = false
  for i = #self.enemy_bullets, 1, -1 do
    if overlaps(self.enemy_bullets[i], p) then
      table.remove(self.enemy_bullets, i)
      hit = true
    end
  end
  for _, enemy in ipairs(self.enemies) do
    if overlaps(enemy, p) then
      hit = true
    end
  end
  if hit then
    self:burst(p.x, p.y, 24, "player")
    events[#events + 1] = { kind = "player_hit" }
  end
end

--- Advances the simulation. `controls` = { move_x, move_y in [-1, 1], fire }.
-- Returns a list of events: { kind = "enemy_killed", enemy } and
-- { kind = "player_hit" }.
function World:update(dt, controls)
  local c = self.config
  local events = {}
  update_player(self, dt, controls)
  move_and_cull(self.bullets, dt, c.width, c.height)
  move_and_cull(self.enemy_bullets, dt, c.width, c.height)
  for i = #self.particles, 1, -1 do
    local part = self.particles[i]
    part.life = part.life - dt
    if part.life <= 0 then
      table.remove(self.particles, i)
    else
      part.x = part.x + part.vx * dt
      part.y = part.y + part.vy * dt
    end
  end
  hit_enemies(self, events)
  hit_player(self, events)
  return events
end

World.PARTICLE_LIFE = PARTICLE_LIFE

return World
