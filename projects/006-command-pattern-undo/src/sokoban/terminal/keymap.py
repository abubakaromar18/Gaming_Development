"""Terminal key names -> UI actions.

Keys arrive as curses key names (``curses.keyname``), e.g. ``"KEY_UP"``,
``"a"``, ``"^R"``. Keeping the table as plain strings means it can be tested,
and edited, without a terminal.
"""
from __future__ import annotations

from enum import Enum, auto

from ..core.geometry import Direction


class Action(Enum):
    UP = auto()
    DOWN = auto()
    LEFT = auto()
    RIGHT = auto()
    UNDO = auto()
    REDO = auto()
    RESTART = auto()
    HINT = auto()
    CHECK = auto()
    AUTOPLAY = auto()
    NEXT_LEVEL = auto()
    PREV_LEVEL = auto()
    QUIT = auto()


MOVE_DIRECTIONS = {
    Action.UP: Direction.UP,
    Action.DOWN: Direction.DOWN,
    Action.LEFT: Direction.LEFT,
    Action.RIGHT: Direction.RIGHT,
}

DEFAULT_KEYMAP: dict[str, Action] = {
    "KEY_UP": Action.UP, "w": Action.UP, "k": Action.UP,
    "KEY_DOWN": Action.DOWN, "s": Action.DOWN, "j": Action.DOWN,
    "KEY_LEFT": Action.LEFT, "a": Action.LEFT, "h": Action.LEFT,
    "KEY_RIGHT": Action.RIGHT, "d": Action.RIGHT, "l": Action.RIGHT,
    "z": Action.UNDO, "u": Action.UNDO, "KEY_BACKSPACE": Action.UNDO, "^H": Action.UNDO,
    "y": Action.REDO, "^R": Action.REDO,
    "r": Action.RESTART,
    "?": Action.HINT,
    "c": Action.CHECK,
    "p": Action.AUTOPLAY,
    "n": Action.NEXT_LEVEL, "KEY_NPAGE": Action.NEXT_LEVEL,
    "b": Action.PREV_LEVEL, "KEY_PPAGE": Action.PREV_LEVEL,
    "q": Action.QUIT, "^[": Action.QUIT,
}

HELP = "Move: arrows/WASD/hjkl  z undo  y redo  r restart  ? hint  c check  p solve  n/b level  q quit"


def action_for(key_name: str, keymap: dict[str, Action] = DEFAULT_KEYMAP) -> Action | None:
    return keymap.get(key_name)
