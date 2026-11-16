"""Game session: a level collection, the current board, and its history.

This is the façade the front end talks to. Every board change it makes goes
through ``CommandHistory``, so hints, auto-play, and restarts are all
undoable exactly like the player's own moves.
"""
from __future__ import annotations

from dataclasses import dataclass

from .board import Board
from .commands import MoveCommand, RestartCommand
from .geometry import Direction, Pos
from .history import CommandHistory
from .level import Level
from .solver import DEFAULT_MAX_STATES, SolveResult, SolveStatus, solve_board


@dataclass(frozen=True)
class GameConfig:
    history_limit: int | None = None  # None = unlimited undo
    solver_max_states: int = DEFAULT_MAX_STATES

    def __post_init__(self) -> None:
        if self.history_limit is not None and self.history_limit < 1:
            raise ValueError("history_limit must be at least 1 or None")
        if self.solver_max_states < 1:
            raise ValueError("solver_max_states must be positive")


@dataclass(frozen=True)
class HintResult:
    status: SolveStatus
    moved: Direction | None  # the step the hint just played, if any
    pushes_left: int | None


class Game:
    def __init__(self, levels: list[Level], start_index: int = 0, config: GameConfig | None = None) -> None:
        if not levels:
            raise ValueError("a game needs at least one level")
        self.levels = list(levels)
        self.config = config or GameConfig()
        # Cached solver line: (player, boxes) it applies to, and the moves left from there.
        self._plan: tuple[Pos, frozenset[Pos], str] | None = None
        self.select(start_index)

    # -- level navigation ---------------------------------------------------

    @property
    def level(self) -> Level:
        return self.board.level

    def select(self, index: int) -> None:
        if not 0 <= index < len(self.levels):
            raise IndexError(f"level index {index} out of range 0..{len(self.levels) - 1}")
        self.index = index
        self.board = Board(self.levels[index])
        # History is per attempt; switching levels starts a fresh one.
        self.history = CommandHistory(self.board, self.config.history_limit)
        self._plan = None

    def next_level(self) -> bool:
        if self.index + 1 >= len(self.levels):
            return False
        self.select(self.index + 1)
        return True

    def prev_level(self) -> bool:
        if self.index == 0:
            return False
        self.select(self.index - 1)
        return True

    # -- player actions -----------------------------------------------------

    @property
    def solved(self) -> bool:
        return self.board.is_solved()

    def move(self, direction: Direction) -> bool:
        """Try a step. Returns False for a blocked move or a finished level."""
        if self.solved or not self.board.can_move(direction):
            return False
        self.history.execute(MoveCommand(direction))
        return True

    def undo(self) -> bool:
        return self.history.undo() is not None

    def redo(self) -> bool:
        return self.history.redo() is not None

    def restart(self) -> bool:
        level = self.level
        if self.board.player == level.start_player and self.board.boxes == level.start_boxes:
            return False  # already at the start; don't pad history with a no-op
        self.history.execute(RestartCommand())
        return True

    # -- solver-backed helpers ----------------------------------------------

    def check(self) -> SolveResult:
        """Is the current position still solvable?"""
        return solve_board(self.board, self.config.solver_max_states)

    def solution(self) -> SolveResult:
        """A solution from the current position, reusing the cached plan when the player followed it."""
        cached = self._cached_plan()
        if cached is not None:
            return SolveResult(SolveStatus.SOLVED, cached, 0)
        result = self.check()
        if result.status is SolveStatus.SOLVED:
            self._plan = (self.board.player, self.board.boxes, result.solution or "")
        return result

    def hint(self) -> HintResult:
        """Play the next move of a solution (undoable like any other move)."""
        if self.solved:
            return HintResult(SolveStatus.SOLVED, None, 0)
        result = self.solution()
        if result.status is not SolveStatus.SOLVED or not result.solution:
            return HintResult(result.status, None, None)
        plan = result.solution
        step = Direction.from_letter(plan[0])
        self.move(step)
        rest = plan[1:]
        self._plan = (self.board.player, self.board.boxes, rest)
        return HintResult(SolveStatus.SOLVED, step, sum(ch.isupper() for ch in rest))

    def _cached_plan(self) -> str | None:
        if self._plan is None:
            return None
        player, boxes, rest = self._plan
        if player == self.board.player and boxes == self.board.boxes:
            return rest
        return None


class SolutionPlayer:
    """Plays a LURD solution back one step per ``step_interval`` seconds.

    Time is injected through ``update(dt)``: an accumulator converts arbitrary
    frame times into whole steps, so playback speed is independent of the
    frame rate and tests can drive it with exact numbers.
    """

    def __init__(self, game: Game, solution: str, step_interval: float = 0.12) -> None:
        if step_interval <= 0:
            raise ValueError("step_interval must be positive")
        self._game = game
        self._steps = [Direction.from_letter(ch) for ch in solution if not ch.isspace()]
        self._next = 0
        self._interval = step_interval
        self._accumulator = 0.0

    @property
    def finished(self) -> bool:
        return self._next >= len(self._steps)

    @property
    def remaining(self) -> int:
        return len(self._steps) - self._next

    def update(self, dt: float) -> int:
        """Advance by ``dt`` seconds; returns how many moves were played."""
        if dt < 0:
            raise ValueError("dt must be non-negative")
        self._accumulator += dt
        played = 0
        while self._accumulator >= self._interval and not self.finished:
            self._accumulator -= self._interval
            if not self._game.move(self._steps[self._next]):
                # The board no longer matches the plan; stop instead of desyncing.
                self._next = len(self._steps)
                break
            self._next += 1
            played += 1
        return played
