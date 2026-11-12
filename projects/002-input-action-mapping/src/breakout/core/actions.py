"""The vocabulary of player intent.

Game code never asks "is the space bar down?". It asks "was LAUNCH just
pressed?". Actions are grouped into contexts so the same physical key can mean
different things on different screens (Enter launches nothing in gameplay but
confirms in the bindings menu) without counting as a conflict.
"""
from __future__ import annotations

from enum import Enum


class Context(Enum):
    GAMEPLAY = "gameplay"
    MENU = "menu"


class Action(Enum):
    # Gameplay
    MOVE_LEFT = "move_left"
    MOVE_RIGHT = "move_right"
    LAUNCH = "launch"
    PAUSE = "pause"
    OPEN_BINDINGS = "open_bindings"
    QUIT = "quit"
    # Menu navigation
    MENU_UP = "menu_up"
    MENU_DOWN = "menu_down"
    CONFIRM = "confirm"
    BACK = "back"

    @property
    def context(self) -> Context:
        return _CONTEXT[self]

    @property
    def label(self) -> str:
        return self.value.replace("_", " ").title()


_CONTEXT: dict[Action, Context] = {
    Action.MOVE_LEFT: Context.GAMEPLAY,
    Action.MOVE_RIGHT: Context.GAMEPLAY,
    Action.LAUNCH: Context.GAMEPLAY,
    Action.PAUSE: Context.GAMEPLAY,
    Action.OPEN_BINDINGS: Context.GAMEPLAY,
    Action.QUIT: Context.GAMEPLAY,
    Action.MENU_UP: Context.MENU,
    Action.MENU_DOWN: Context.MENU,
    Action.CONFIRM: Context.MENU,
    Action.BACK: Context.MENU,
}

#: Actions the player may rebind from the in-game menu. Menu navigation stays
#: fixed so a bad rebind can never lock the player out of the menu itself.
REBINDABLE: tuple[Action, ...] = tuple(a for a in Action if a.context is Context.GAMEPLAY)


def actions_in(context: Context) -> tuple[Action, ...]:
    return tuple(a for a in Action if a.context is context)
