"""Grid positions and the four movement directions."""
from __future__ import annotations

from enum import Enum
from typing import NamedTuple


class Pos(NamedTuple):
    row: int
    col: int

    def step(self, direction: "Direction") -> "Pos":
        return Pos(self.row + direction.dr, self.col + direction.dc)


class Direction(Enum):
    # value = (row delta, col delta, LURD letter)
    UP = (-1, 0, "u")
    DOWN = (1, 0, "d")
    LEFT = (0, -1, "l")
    RIGHT = (0, 1, "r")

    @property
    def dr(self) -> int:
        return self.value[0]

    @property
    def dc(self) -> int:
        return self.value[1]

    @property
    def letter(self) -> str:
        """Lowercase LURD letter; Sokoban notation uppercases it for a push."""
        return self.value[2]

    @property
    def opposite(self) -> "Direction":
        return _OPPOSITE[self]

    @classmethod
    def from_letter(cls, letter: str) -> "Direction":
        try:
            return _BY_LETTER[letter.lower()]
        except KeyError:
            raise ValueError(f"not a LURD move letter: {letter!r}") from None


_OPPOSITE = {
    Direction.UP: Direction.DOWN,
    Direction.DOWN: Direction.UP,
    Direction.LEFT: Direction.RIGHT,
    Direction.RIGHT: Direction.LEFT,
}
_BY_LETTER = {d.letter: d for d in Direction}
