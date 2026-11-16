"""Pure Sokoban rules, commands, history, and solver. No I/O, no terminal code."""
from .board import Board, Snapshot
from .commands import Command, IllegalMoveError, MoveCommand, RestartCommand
from .game import Game, GameConfig, HintResult, SolutionPlayer
from .geometry import Direction, Pos
from .history import CommandHistory
from .level import Level, LevelError, parse_collection, parse_level
from .render import to_xsb
from .solver import CheckResult, SolveResult, SolveStatus, check_solution, dead_squares, solve, solve_board

__all__ = [
    "Board", "Snapshot", "Command", "IllegalMoveError", "MoveCommand", "RestartCommand",
    "Game", "GameConfig", "HintResult", "SolutionPlayer", "Direction", "Pos", "CommandHistory",
    "Level", "LevelError", "parse_collection", "parse_level", "to_xsb", "CheckResult",
    "SolveResult", "SolveStatus", "check_solution", "dead_squares", "solve", "solve_board",
]
