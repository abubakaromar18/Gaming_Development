--- Composition root for the core: builds the shared context and a state
--- stack that starts on the title menu. The platform layer and the tests both
--- go through here, so they run exactly the same game.

local Config = require("core.config")
local Rng = require("core.rng")
local Scores = require("core.scores")
local StateStack = require("core.state_stack")
local Menu = require("core.states.menu")

local Game = {}

--- @param opts table|nil
---   seed         number  RNG seed (default 1)
---   best         number  previously saved best score (default 0)
---   config       table   overrides for core.config defaults
---   on_new_best  function(best) called when a round sets a new best
--- @return table stack, table ctx
function Game.new(opts)
  opts = opts or {}
  local ctx = {
    config = Config.build(opts.config),
    rng = Rng.new(opts.seed or 1),
    scores = Scores.new(opts.best),
    on_new_best = opts.on_new_best or function() end,
  }
  local stack = StateStack.new()
  stack:push(Menu.new(ctx))
  return stack, ctx
end

return Game
