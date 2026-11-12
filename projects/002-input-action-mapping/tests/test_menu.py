from breakout.core.actions import REBINDABLE, Action
from breakout.core.bindings import Key, default_bindings
from breakout.core.menu import DONE_ITEM, RESET_ITEM, BindingsMenu, MenuOutcome
from breakout.core.resolver import NO_INPUT, InputResolver
from helpers import keys


class Driver:
    """Feeds raw frames through a real resolver into the menu, like the app does."""

    def __init__(self):
        self.map = default_bindings()
        self.resolver = InputResolver(self.map)
        self.menu = BindingsMenu(self.map, default_bindings)

    def frame(self, raw=NO_INPUT, dt=1 / 60):
        return self.menu.update(self.resolver.resolve(raw), raw, dt)

    def tap(self, *names):
        out = self.frame(keys(*names))
        self.frame()
        return out


def test_cursor_wraps_both_ways():
    d = Driver()
    d.tap("up")
    assert d.menu.rows()[d.menu.cursor].label == DONE_ITEM
    d.tap("down")
    assert d.menu.cursor == 0


def test_full_rebind_flow_through_the_menu():
    d = Driver()
    assert d.menu.rows()[0].label == REBINDABLE[0].label  # Move Left
    d.frame(keys("return"))  # open prompt, Enter still held next frame
    assert d.menu.capture is not None
    d.frame(keys("return"))
    assert d.menu.capture is not None, "held confirm key must not be captured"
    d.frame()
    d.frame(keys("j"))
    assert d.menu.capture is None
    assert Key("j") in d.map.sources_for(Action.MOVE_LEFT)
    assert d.menu.dirty
    assert "Move Left -> J" in d.menu.message


def test_menu_ignores_navigation_while_capturing():
    d = Driver()
    d.tap("return")
    d.frame(keys("down"))
    assert d.menu.cursor == 0
    assert Key("down") in d.map.sources_for(Action.MOVE_LEFT)


def test_stealing_is_reported():
    d = Driver()
    d.menu.cursor = REBINDABLE.index(Action.PAUSE)
    d.tap("return")
    d.tap("space")
    assert "Moved from Launch" in d.menu.message


def test_back_cancels_a_capture_without_closing_the_menu():
    d = Driver()
    d.tap("return")
    assert d.tap("escape") is MenuOutcome.STAY
    assert d.menu.capture is None
    assert "cancelled" in d.menu.message
    assert not d.menu.dirty


def test_back_closes_when_idle():
    assert Driver().tap("escape") is MenuOutcome.CLOSE


def test_done_item_closes():
    d = Driver()
    d.menu.cursor = len(d.menu.rows()) - 1
    assert d.tap("return") is MenuOutcome.CLOSE


def test_reset_restores_defaults_in_place():
    d = Driver()
    d.map.rebind(Action.LAUNCH, Key("z"))
    d.menu.cursor = [r.label for r in d.menu.rows()].index(RESET_ITEM)
    d.tap("return")
    assert d.map == default_bindings()
    assert d.resolver.action_map is d.map
    assert d.menu.dirty


def test_unbound_actions_are_labelled():
    d = Driver()
    d.map.rebind(Action.LAUNCH, Key("escape"))
    quit_row = next(r for r in d.menu.rows() if r.label == Action.QUIT.label)
    assert quit_row.bindings == "(unbound)"
