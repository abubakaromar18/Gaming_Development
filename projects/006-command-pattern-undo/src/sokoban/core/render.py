"""Board -> text. Pure, so the terminal UI and the tests share it."""
from __future__ import annotations

from .board import Board
from .geometry import Pos


def to_xsb(board: Board) -> list[str]:
    """The current board in XSB notation; ``parse_level`` round-trips it."""
    level = board.level
    rows: list[str] = []
    for r in range(level.height):
        chars: list[str] = []
        for c in range(level.width):
            pos = Pos(r, c)
            goal = pos in level.goals
            if pos in level.walls:
                chars.append("#")
            elif pos == board.player:
                chars.append("+" if goal else "@")
            elif board.has_box(pos):
                chars.append("*" if goal else "$")
            else:
                chars.append("." if goal else " ")
        rows.append("".join(chars).rstrip())
    return rows
