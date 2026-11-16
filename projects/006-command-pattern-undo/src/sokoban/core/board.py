"""The mutable game state that commands act on (the command *receiver*).

``Board`` exposes small, dumb primitives (``place_player``, ``move_box``) and
queries. It never decides what a move *means*; that is the commands' job.
Keeping the receiver this simple is what makes every command trivially
reversible.
"""
from __future__ import annotations

from dataclasses import dataclass

from .geometry import Direction, Pos
from .level import Level


@dataclass(frozen=True)
class Snapshot:
    """A memento of everything mutable on a board."""

    player: Pos
    boxes: frozenset[Pos]
    moves: int
    pushes: int


class Board:
    def __init__(self, level: Level) -> None:
        self.level = level
        self.player: Pos = level.start_player
        self._boxes: set[Pos] = set(level.start_boxes)
        self.moves = 0
        self.pushes = 0

    # -- queries -------------------------------------------------------------

    @property
    def boxes(self) -> frozenset[Pos]:
        return frozenset(self._boxes)

    def has_box(self, pos: Pos) -> bool:
        return pos in self._boxes

    def is_free(self, pos: Pos) -> bool:
        """True if a box could be pushed onto ``pos``."""
        return not self.level.is_wall(pos) and pos not in self._boxes

    def can_move(self, direction: Direction) -> bool:
        target = self.player.step(direction)
        if self.level.is_wall(target):
            return False
        if target in self._boxes:
            return self.is_free(target.step(direction))
        return True

    def is_solved(self) -> bool:
        return self._boxes == self.level.goals

    def boxes_on_goals(self) -> int:
        return len(self._boxes & self.level.goals)

    # -- primitives used by commands ------------------------------------------

    def place_player(self, pos: Pos) -> None:
        self.player = pos

    def move_box(self, src: Pos, dst: Pos) -> None:
        if src not in self._boxes:
            raise ValueError(f"no box at {src}")
        if dst in self._boxes:
            raise ValueError(f"box already at {dst}")
        self._boxes.remove(src)
        self._boxes.add(dst)

    def snapshot(self) -> Snapshot:
        return Snapshot(self.player, self.boxes, self.moves, self.pushes)

    def restore(self, snap: Snapshot) -> None:
        self.player = snap.player
        self._boxes = set(snap.boxes)
        self.moves = snap.moves
        self.pushes = snap.pushes
