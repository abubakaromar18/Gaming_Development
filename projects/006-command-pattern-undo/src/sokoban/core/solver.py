"""Level solver and solution checker.

The solver answers two questions the game asks:

* *Is this level (or this position) still solvable?* Used to validate bundled
  levels and to tell players they have deadlocked themselves.
* *What is a solution?* Used for hints and auto-play.

It is a breadth-first search over **pushes**, not single steps. Walking around
without touching a box never changes what is possible, so each search node is
"box layout + which region the player is standing in". The region is
normalized to its smallest cell, which collapses thousands of equivalent
player positions into one node. BFS over pushes returns a push-optimal
solution.

Simple deadlocks are pruned with precomputed *dead squares*: floor cells from
which no box can ever reach a goal (most corners and walls without goals).
"""
from __future__ import annotations

from collections import deque
from dataclasses import dataclass
from enum import Enum

from .board import Board
from .commands import IllegalMoveError, MoveCommand
from .geometry import Direction, Pos
from .level import Level

DEFAULT_MAX_STATES = 200_000


class SolveStatus(Enum):
    SOLVED = "solved"
    UNSOLVABLE = "unsolvable"
    GAVE_UP = "gave up"  # hit the state budget before deciding


@dataclass(frozen=True)
class SolveResult:
    status: SolveStatus
    solution: str | None  # LURD; uppercase letters are pushes
    explored: int

    @property
    def pushes(self) -> int | None:
        if self.solution is None:
            return None
        return sum(ch.isupper() for ch in self.solution)


def dead_squares(level: Level) -> frozenset[Pos]:
    """Floor cells a box can never be pushed from onto any goal.

    Computed backwards: start from every goal and *pull* boxes outward. A pull
    in direction ``d`` moves a box from ``b`` to ``b+d`` and needs the player to
    have room at ``b+d`` and ``b+2d``. Every cell reached is live; the rest of
    the interior is dead. Boxes are ignored, so this is a safe
    under-approximation of real deadlocks.
    """
    live = set(level.goals)
    frontier = list(level.goals)
    while frontier:
        box = frontier.pop()
        for d in Direction:
            dst = box.step(d)
            if dst in live or level.is_wall(dst) or level.is_wall(dst.step(d)):
                continue
            live.add(dst)
            frontier.append(dst)
    return level.interior - live


def solve(
    level: Level,
    player: Pos | None = None,
    boxes: frozenset[Pos] | None = None,
    max_states: int = DEFAULT_MAX_STATES,
) -> SolveResult:
    """Search for a push-optimal solution from the given position (default: level start)."""
    if max_states < 1:
        raise ValueError("max_states must be positive")
    player = level.start_player if player is None else player
    boxes = level.start_boxes if boxes is None else frozenset(boxes)
    goals = level.goals
    dead = dead_squares(level)

    if boxes == goals:
        return SolveResult(SolveStatus.SOLVED, "", 0)
    if boxes & dead:
        return SolveResult(SolveStatus.UNSOLVABLE, None, 0)

    graph = _Graph(level)
    goal_ids = frozenset(graph.index[g] for g in goals)
    dead_ids = frozenset(graph.index[c] for c in dead)
    start_layout = frozenset(graph.index[b] for b in boxes)
    start_at = graph.index[player]
    nbr = graph.neighbours

    # A state is (smallest cell of the player's region, box layout).
    start_key = (min(graph.region(start_at, start_layout)), start_layout)
    # parent[child] = (parent key, cell the player pushed from, direction index)
    parent: dict = {start_key: None}
    queue = deque([(start_key, start_at)])

    while queue:
        key, at = queue.popleft()
        layout = key[1]
        reach = graph.region(at, layout)
        for box in layout:
            for d in range(4):
                stand = nbr[box][_OPPOSITE_INDEX[d]]
                dst = nbr[box][d]
                if dst < 0 or stand not in reach or dst in layout or dst in dead_ids:
                    continue
                new_layout = layout - {box} | {dst}
                new_key = (min(graph.region(box, new_layout)), new_layout)
                if new_key in parent:
                    continue
                parent[new_key] = (key, stand, d)
                if new_layout == goal_ids:
                    pushes = [(graph.cells[c], _DIRS[i]) for c, i in _unwind(parent, new_key)]
                    return SolveResult(SolveStatus.SOLVED, _to_lurd(level, player, boxes, pushes), len(parent))
                if len(parent) >= max_states:
                    return SolveResult(SolveStatus.GAVE_UP, None, len(parent))
                queue.append((new_key, box))
    return SolveResult(SolveStatus.UNSOLVABLE, None, len(parent))


_DIRS = list(Direction)
_OPPOSITE_INDEX = [_DIRS.index(d.opposite) for d in _DIRS]


class _Graph:
    """The level interior as integer cell ids with a neighbour table.

    The search touches millions of cells on bigger levels; small ints in sets
    and a precomputed adjacency list are several times faster in Python than
    hashing ``Pos`` tuples and re-checking walls. Cells are numbered in
    row-major order, so the smallest id is also the smallest ``Pos``.
    """

    def __init__(self, level: Level) -> None:
        self.cells: list[Pos] = sorted(level.interior)
        self.index: dict[Pos, int] = {pos: i for i, pos in enumerate(self.cells)}
        self.neighbours: list[tuple[int, ...]] = [
            tuple(self.index.get(pos.step(d), -1) for d in _DIRS) for pos in self.cells
        ]

    def region(self, start: int, layout: frozenset[int]) -> set[int]:
        nbr = self.neighbours
        seen = {start}
        stack = [start]
        while stack:
            for nxt in nbr[stack.pop()]:
                if nxt >= 0 and nxt not in seen and nxt not in layout:
                    seen.add(nxt)
                    stack.append(nxt)
        return seen


def solve_board(board: Board, max_states: int = DEFAULT_MAX_STATES) -> SolveResult:
    return solve(board.level, board.player, board.boxes, max_states)


def _unwind(parent, key) -> list:
    pushes = []
    while (link := parent[key]) is not None:
        key, stand, d = link
        pushes.append((stand, d))
    pushes.reverse()
    return pushes


def _to_lurd(level: Level, player: Pos, boxes: frozenset[Pos], pushes: list[tuple[Pos, Direction]]) -> str:
    """Expand a list of pushes into full LURD, adding the walks between them."""
    out: list[str] = []
    layout = set(boxes)
    for stand, d in pushes:
        out.append(_walk(level, player, stand, layout))
        box = stand.step(d)
        layout.remove(box)
        layout.add(box.step(d))
        player = box
        out.append(d.letter.upper())
    return "".join(out)


def _walk(level: Level, src: Pos, dst: Pos, boxes: set[Pos]) -> str:
    """Shortest non-pushing walk between two cells, as lowercase LURD."""
    came: dict[Pos, tuple[Pos, Direction] | None] = {src: None}
    queue = deque([src])
    while queue:
        pos = queue.popleft()
        if pos == dst:
            break
        for d in Direction:
            nxt = pos.step(d)
            if nxt not in came and nxt not in boxes and not level.is_wall(nxt):
                came[nxt] = (pos, d)
                queue.append(nxt)
    if dst not in came:
        raise RuntimeError("solver produced an unreachable push; search invariant broken")
    steps: list[str] = []
    pos = dst
    while (link := came[pos]) is not None:
        pos, d = link
        steps.append(d.letter)
    return "".join(reversed(steps))


@dataclass(frozen=True)
class CheckResult:
    solved: bool
    reason: str
    moves_applied: int


def check_solution(level: Level, lurd: str) -> CheckResult:
    """Replay a LURD string through the real game rules and report the outcome.

    Moves go through ``MoveCommand``, the same code the player uses, so the
    checker can never disagree with the game. Letter case is accepted either
    way; the board decides whether a step pushes. Whitespace is ignored, and
    so is anything after the last box lands.
    """
    board = Board(level)
    applied = 0
    for ch in lurd:
        if ch.isspace():
            continue
        try:
            MoveCommand(Direction.from_letter(ch)).execute(board)
        except ValueError as exc:  # unknown letter or IllegalMoveError
            kind = "blocked" if isinstance(exc, IllegalMoveError) else "invalid"
            return CheckResult(False, f"{kind} move {ch!r} at step {applied + 1}", applied)
        applied += 1
        if board.is_solved():
            # Like the game, the level ends the moment the last box lands.
            return CheckResult(True, "solved", applied)
    return CheckResult(False, f"{board.boxes_on_goals()}/{len(level.goals)} boxes on goals after all moves", applied)
