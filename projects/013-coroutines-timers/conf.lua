function love.conf(t)
  t.identity = "coroutine-waves"
  t.version = "11.5"
  t.window.title = "Coroutine Waves: scripted shooter with timers and tweens"
  t.window.width = 480
  t.window.height = 640
  t.window.resizable = true
  t.window.minwidth = 240
  t.window.minheight = 320
  t.modules.audio = false
  t.modules.sound = false
  t.modules.physics = false
  t.modules.joystick = false
end
