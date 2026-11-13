-- LÖVE entry point: wires love callbacks to the core state stack. All game
-- logic lives in src/core; this file only moves input in and views out.

love.filesystem.setRequirePath("src/?.lua;src/?/init.lua;" .. love.filesystem.getRequirePath())

local Game = require("core.game")
local Input = require("platform.input")
local Renderer = require("platform.renderer")
local Storage = require("platform.storage")

-- A frame longer than this (window drag, breakpoint) is treated as this long,
-- so the countdown and snake never jump far ahead after a hitch.
local MAX_FRAME = 0.1

local stack, renderer
local show_stack = false

function love.load(args)
  local seed = tonumber(args and args[1]) or os.time()
  stack = Game.new({
    seed = seed,
    best = Storage.load_best(),
    on_new_best = Storage.save_best,
  })
  renderer = Renderer.new()
end

function love.update(dt)
  dt = math.min(dt, MAX_FRAME)
  renderer:update(dt)
  stack:update(dt)
  if stack:is_empty() then
    love.event.quit()
  end
end

function love.keypressed(key, _, isrepeat)
  if key == "f3" then
    show_stack = not show_stack
    return
  end
  if isrepeat then
    return
  end
  local action = Input.action_for_key(key)
  if action then
    stack:handle(action)
  end
end

-- Alt-tabbing away mid-round should pause, not kill the snake.
function love.focus(focused)
  if not focused then
    stack:handle("suspend")
  end
end

function love.draw()
  if stack:is_empty() then
    return
  end
  renderer:draw(stack:views(), show_stack and stack:names() or nil)
end
