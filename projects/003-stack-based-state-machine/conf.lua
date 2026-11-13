function love.conf(t)
  t.identity = "stack-snake" -- save directory name for the best score
  t.version = "11.5"
  t.window.title = "Stack Snake: pushdown state machine"
  t.window.width = 800
  t.window.height = 640
  t.window.resizable = true
  t.window.minwidth = 400
  t.window.minheight = 360
  t.modules.audio = false
  t.modules.sound = false
  t.modules.physics = false
  t.modules.joystick = false
end
