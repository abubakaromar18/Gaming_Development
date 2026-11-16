"""Linear undo/redo history (the command *invoker*).

Classic two-stack design: executing a new command pushes it onto the undo
stack and clears the redo stack, because the redo branch no longer follows
from the current state. Undo moves a command across; redo moves it back and
re-executes it.
"""
from __future__ import annotations

from collections import deque

from .board import Board
from .commands import Command, RestartCommand


class CommandHistory:
    def __init__(self, board: Board, limit: int | None = None) -> None:
        """``limit`` caps undo depth (oldest steps are forgotten); ``None`` is unlimited."""
        if limit is not None and limit < 1:
            raise ValueError("history limit must be at least 1")
        self.board = board
        self._undo: deque[Command] = deque(maxlen=limit)
        self._redo: list[Command] = []

    def execute(self, command: Command) -> None:
        # Execute first: a command that raises (an illegal move) never enters history.
        command.execute(self.board)
        self._undo.append(command)
        self._redo.clear()

    def undo(self) -> Command | None:
        if not self._undo:
            return None
        command = self._undo.pop()
        command.undo(self.board)
        self._redo.append(command)
        return command

    def redo(self) -> Command | None:
        if not self._redo:
            return None
        command = self._redo.pop()
        command.execute(self.board)
        self._undo.append(command)
        return command

    def clear(self) -> None:
        self._undo.clear()
        self._redo.clear()

    @property
    def can_undo(self) -> bool:
        return bool(self._undo)

    @property
    def can_redo(self) -> bool:
        return bool(self._redo)

    @property
    def undo_depth(self) -> int:
        return len(self._undo)

    @property
    def redo_depth(self) -> int:
        return len(self._redo)

    def notation(self) -> str:
        """LURD log of the moves made since the most recent restart.

        With a ``limit``, forgotten steps are missing from the log, so it is a
        replayable solution only when the history was never truncated.
        """
        letters: list[str] = []
        for command in self._undo:
            if isinstance(command, RestartCommand):
                letters.clear()
            else:
                letters.append(command.notation)
        return "".join(letters)
