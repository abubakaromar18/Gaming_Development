--- A vertical list of options with a wrapping cursor, shared by every menu
--- screen. Each item is { id = "play", label = "Play" }.

local MenuList = {}
MenuList.__index = MenuList

function MenuList.new(items)
  if type(items) ~= "table" or #items == 0 then
    error("a menu needs at least one item", 2)
  end
  return setmetatable({ items = items, index = 1 }, MenuList)
end

--- Move the cursor; wraps around at both ends.
function MenuList:move(delta)
  local n = #self.items
  self.index = (self.index - 1 + delta) % n + 1
end

function MenuList:selected()
  return self.items[self.index].id
end

--- Handle a navigation action. Returns the chosen item id on "confirm".
function MenuList:handle(action)
  if action == "up" then
    self:move(-1)
  elseif action == "down" then
    self:move(1)
  elseif action == "confirm" then
    return self:selected()
  end
  return nil
end

--- Plain data for the renderer.
function MenuList:view()
  local labels = {}
  for i, item in ipairs(self.items) do
    labels[i] = item.label
  end
  return { labels = labels, index = self.index }
end

return MenuList
