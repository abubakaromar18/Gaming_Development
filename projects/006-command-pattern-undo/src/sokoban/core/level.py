"""Immutable level definitions parsed from the standard XSB text format.

XSB symbols::

    #  wall           .  goal
    @  player         +  player on goal
    $  box            *  box on goal
    (space, - or _)  floor

A ``Level`` holds only what never changes during play (walls, goals) plus the
starting positions. The mutable part lives in ``Board``.
"""
from __future__ import annotations

from dataclasses import dataclass
from functools import cached_property

from .geometry import Direction, Pos

_FLOOR_CHARS = frozenset(" -_")
_KNOWN_CHARS = frozenset("#.@+$*") | _FLOOR_CHARS


class LevelError(ValueError):
    """A level string that is not a playable Sokoban level."""


@dataclass(frozen=True)
class Level:
    name: str
    width: int
    height: int
    walls: frozenset[Pos]
    goals: frozenset[Pos]
    start_player: Pos
    start_boxes: frozenset[Pos]

    def in_bounds(self, pos: Pos) -> bool:
        return 0 <= pos.row < self.height and 0 <= pos.col < self.width

    def is_wall(self, pos: Pos) -> bool:
        # Out-of-bounds counts as wall so movement code needs no bounds checks.
        return pos in self.walls or not self.in_bounds(pos)

    @cached_property
    def interior(self) -> frozenset[Pos]:
        """Floor cells the player can ever reach (ignoring boxes). Computed once."""
        return _flood(self.start_player, self.is_wall)


def parse_level(text: str, name: str = "Untitled") -> Level:
    """Parse one XSB level. Raises ``LevelError`` with a precise reason."""
    lines = [line.rstrip() for line in text.splitlines()]
    while lines and not lines[0]:
        lines.pop(0)
    while lines and not lines[-1]:
        lines.pop()
    if not lines:
        raise LevelError(f"{name}: level is empty")

    walls: set[Pos] = set()
    goals: set[Pos] = set()
    boxes: set[Pos] = set()
    players: list[Pos] = []
    for r, line in enumerate(lines):
        for c, ch in enumerate(line):
            if ch not in _KNOWN_CHARS:
                raise LevelError(f"{name}: unknown symbol {ch!r} at line {r + 1}, column {c + 1}")
            pos = Pos(r, c)
            if ch == "#":
                walls.add(pos)
            if ch in ".+*":
                goals.add(pos)
            if ch in "$*":
                boxes.add(pos)
            if ch in "@+":
                players.append(pos)

    if len(players) != 1:
        raise LevelError(f"{name}: expected exactly one player, found {len(players)}")
    if not boxes:
        raise LevelError(f"{name}: level has no boxes")
    if len(boxes) != len(goals):
        raise LevelError(f"{name}: {len(boxes)} boxes but {len(goals)} goals")
    if boxes == goals:
        raise LevelError(f"{name}: every box already starts on a goal")

    level = Level(
        name=name,
        width=max(len(line) for line in lines),
        height=len(lines),
        walls=frozenset(walls),
        goals=frozenset(goals),
        start_player=players[0],
        start_boxes=frozenset(boxes),
    )
    _check_enclosed(level)
    return level


def parse_collection(text: str) -> list[Level]:
    """Parse a file of XSB levels separated by blank lines.

    Lines starting with ``;`` are comments; the first comment before a level
    becomes its name (``; Level 3: The Corridor``).
    """
    levels: list[Level] = []
    block: list[str] = []
    title: str | None = None

    def flush() -> None:
        nonlocal block, title
        if block:
            levels.append(parse_level("\n".join(block), title or f"Level {len(levels) + 1}"))
        block, title = [], None

    for line in text.splitlines():
        stripped = line.strip()
        if stripped.startswith(";"):
            if block:
                flush()
            if title is None:
                title = stripped.lstrip(";").strip() or None
        elif not stripped:
            flush()
        else:
            block.append(line)
    flush()
    if not levels:
        raise LevelError("collection contains no levels")
    return levels


def _flood(start: Pos, blocked) -> frozenset[Pos]:
    seen = {start}
    stack = [start]
    while stack:
        pos = stack.pop()
        for d in Direction:
            nxt = pos.step(d)
            if nxt not in seen and not blocked(nxt):
                seen.add(nxt)
                stack.append(nxt)
    return frozenset(seen)


def _check_enclosed(level: Level) -> None:
    """Reject levels whose playable area leaks off the edge of the map.

    Flood fill from the player treating only walls as blocking. If it steps
    outside the map, the player could walk off the level.
    """
    def wall_or_leak(pos: Pos) -> bool:
        if not level.in_bounds(pos):
            raise LevelError(f"{level.name}: level is not enclosed by walls")
        return pos in level.walls

    reachable = _flood(level.start_player, wall_or_leak)
    stray = sorted((level.start_boxes | level.goals) - reachable)
    if stray:
        r, c = stray[0]
        raise LevelError(f"{level.name}: box or goal at line {r + 1}, column {c + 1} is outside the playable area")
