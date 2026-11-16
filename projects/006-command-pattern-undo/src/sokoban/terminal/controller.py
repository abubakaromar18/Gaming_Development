"""Turns UI actions into game calls and status messages.

No curses here: the controller is plain Python, so the whole interactive flow
(hints, auto-play, level changes) is testable by feeding it actions and dt.
"""
from __future__ import annotations

from ..core.game import Game, SolutionPlayer
from ..core.solver import SolveStatus
from .keymap import MOVE_DIRECTIONS, Action

AUTOPLAY_STEP_SECONDS = 0.12


class Controller:
    def __init__(self, game: Game, autoplay_step: float = AUTOPLAY_STEP_SECONDS) -> None:
        self.game = game
        self.message = f"{game.level.name}. Push every box onto a goal."
        self._autoplay_step = autoplay_step
        self._autoplay: SolutionPlayer | None = None
        self._was_solved = False

    @property
    def autoplaying(self) -> bool:
        return self._autoplay is not None

    def handle(self, action: Action) -> bool:
        """Apply one action. Returns False when the player asked to quit."""
        if action is Action.QUIT:
            return False
        # Any input takes control back from auto-play (and is then applied).
        if self._autoplay is not None:
            self._autoplay = None
            self.message = "Auto-play stopped."
            if action is Action.AUTOPLAY:
                return True

        game = self.game
        if action in MOVE_DIRECTIONS:
            if not game.move(MOVE_DIRECTIONS[action]) and not game.solved:
                self.message = "Blocked."
        elif action is Action.UNDO:
            self.message = "Undone." if game.undo() else "Nothing to undo."
        elif action is Action.REDO:
            self.message = "Redone." if game.redo() else "Nothing to redo."
        elif action is Action.RESTART:
            self.message = "Restarted (z to undo the restart)." if game.restart() else "Already at the start."
        elif action is Action.HINT:
            self._hint()
        elif action is Action.CHECK:
            self._check()
        elif action is Action.AUTOPLAY:
            self._start_autoplay()
        elif action is Action.NEXT_LEVEL:
            self._change_level(game.next_level(), "This is the last level.")
        elif action is Action.PREV_LEVEL:
            self._change_level(game.prev_level(), "This is the first level.")
        self._announce_win()
        return True

    def tick(self, dt: float) -> None:
        if self._autoplay is None:
            return
        self._autoplay.update(dt)
        if self._autoplay.finished:
            self._autoplay = None
            if not self.game.solved:
                self.message = "Auto-play lost sync with the board and stopped."
        self._announce_win()

    def _hint(self) -> None:
        result = self.game.hint()
        if result.moved is not None:
            self.message = f"Hint: {result.moved.name.lower()}. {result.pushes_left} pushes to go."
        else:
            self.message = _status_text(result.status)

    def _check(self) -> None:
        result = self.game.check()
        if result.status is SolveStatus.SOLVED:
            self.message = f"Still solvable in {result.pushes} pushes."
        else:
            self.message = _status_text(result.status)

    def _start_autoplay(self) -> None:
        if self.game.solved:
            return
        result = self.game.solution()
        if result.status is SolveStatus.SOLVED and result.solution:
            self._autoplay = SolutionPlayer(self.game, result.solution, self._autoplay_step)
            self.message = f"Auto-playing {len(result.solution)} moves (any key stops, z undoes)."
        else:
            self.message = _status_text(result.status)

    def _change_level(self, changed: bool, at_edge: str) -> None:
        self.message = f"{self.game.level.name}." if changed else at_edge

    def _announce_win(self) -> None:
        solved = self.game.solved
        if solved and not self._was_solved:
            board = self.game.board
            self.message = f"Solved in {board.moves} moves, {board.pushes} pushes! n for next level."
        self._was_solved = solved


def _status_text(status: SolveStatus) -> str:
    if status is SolveStatus.UNSOLVABLE:
        return "Deadlocked: no solution from here. Undo!"
    if status is SolveStatus.GAVE_UP:
        return "Too many positions to decide; keep going."
    return "Already solved."
