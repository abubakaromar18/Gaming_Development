"""Commands: every change to a ``Board`` is an object that can be undone.

Two undo strategies are shown side by side:

* ``MoveCommand`` stores the *minimum* needed to reverse itself (the direction
  and whether it pushed a box). Undo is the inverse operation. Cheap: a few
  bytes per move, so unlimited history is fine.
* ``RestartCommand`` stores a *memento* (a full ``Snapshot``) because its
  inverse cannot be computed from the command alone. Restart therefore becomes
  just another undoable step instead of a history-destroying reset.
"""
from __future__ import annotations

from typing import Protocol

from .board import Board, Snapshot
from .geometry import Direction


class Command(Protocol):
    def execute(self, board: Board) -> None: ...

    def undo(self, board: Board) -> None: ...

    @property
    def notation(self) -> str:
        """How this command appears in a move log ("" if it is not a move)."""
        ...


class IllegalMoveError(ValueError):
    pass


class MoveCommand:
    """Walk one cell, pushing a box if one is in the way."""

    __slots__ = ("direction", "pushed")

    def __init__(self, direction: Direction) -> None:
        self.direction = direction
        # Recorded on execute; undo needs it to know whether to pull a box back.
        self.pushed = False

    def execute(self, board: Board) -> None:
        if not board.can_move(self.direction):
            raise IllegalMoveError(f"cannot move {self.direction.name.lower()}")
        target = board.player.step(self.direction)
        self.pushed = board.has_box(target)
        if self.pushed:
            board.move_box(target, target.step(self.direction))
            board.pushes += 1
        board.place_player(target)
        board.moves += 1

    def undo(self, board: Board) -> None:
        origin = board.player.step(self.direction.opposite)
        if self.pushed:
            box = board.player.step(self.direction)
            board.move_box(box, board.player)
            board.pushes -= 1
        board.place_player(origin)
        board.moves -= 1

    @property
    def notation(self) -> str:
        letter = self.direction.letter
        return letter.upper() if self.pushed else letter

    def __repr__(self) -> str:
        return f"MoveCommand({self.direction.name}, pushed={self.pushed})"


class RestartCommand:
    """Reset the board to the level start, remembering where it was."""

    __slots__ = ("_before",)

    def __init__(self) -> None:
        self._before: Snapshot | None = None

    def execute(self, board: Board) -> None:
        self._before = board.snapshot()
        level = board.level
        board.restore(Snapshot(level.start_player, level.start_boxes, 0, 0))

    def undo(self, board: Board) -> None:
        if self._before is None:
            raise RuntimeError("undo called before execute")
        board.restore(self._before)

    @property
    def notation(self) -> str:
        return ""

    def __repr__(self) -> str:
        return "RestartCommand()"
