-- LÖVE entry point: wires love callbacks to the core. All game logic lives in
-- src/core; this file turns the keyboard into controls, the frame time into
-- dt, and the game state into pixels.

love.filesystem.setRequirePath("src/?.lua;src/?/init.lua;" .. love.filesystem.getRequirePath())

local Game = require("core.game")
local Input = require("platform.input")
local Renderer = require("platform.renderer")

-- A frame longer than this (window drag, breakpoint) counts as this long, so
-- scripts and bullets never leap ahead after a hitch.
local MAX_FRAME = 0.1
local FIXED_DT = 1 / 60 -- used by --frames smoke runs
local NO_INPUT = { move_x = 0, move_y = 0, fire = false }

local options, game, renderer
local paused, show_tasks = false, false
local frames_left

local function new_game()
  game = Game.new({ seed = options.seed })
  renderer = Renderer.new(game.config.width, game.config.height, options.seed)
  paused = false
end

function love.load(args)
  options = Input.parse_args(args, os.time())
  frames_left = options.frames
  new_game()
end

function love.update(dt)
  if frames_left then
    game:update(FIXED_DT, NO_INPUT)
    renderer:update(FIXED_DT)
    frames_left = frames_left - 1
    if frames_left <= 0 then
      print(string.format("ran %d frames: state=%s wave=%d score=%d tasks=%d",
        options.frames, game.state, game.wave, game.score, game.sched:count()))
      love.event.quit()
    end
    return
  end
  if paused then
    return -- game time stops, and with it every script, timer and tween
  end
  dt = math.min(dt, MAX_FRAME)
  renderer:update(dt)
  game:update(dt, Input.controls(love.keyboard.isDown))
end

function love.keypressed(key, _, isrepeat)
  if isrepeat then
    return
  end
  if key == "escape" then
    love.event.quit()
  elseif key == "f3" then
    show_tasks = not show_tasks
  elseif key == "p" and game.state == "playing" then
    paused = not paused
  elseif key == "r" and game.state ~= "playing" then
    options.seed = options.seed + 1
    new_game()
  end
end

-- Alt-tabbing away pauses instead of letting bullets fly unseen.
function love.focus(focused)
  if not focused and game.state == "playing" then
    paused = true
  end
end

function love.draw()
  renderer:draw(game, paused, show_tasks and game.sched:names() or nil)
end
