"""Command-line entry point: play, or batch-check that levels are solvable."""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

from .core.game import Game, GameConfig
from .core.level import Level, LevelError
from .core.solver import SolveStatus, solve
from .resources import bundled_levels, load_levels


def check_report(levels: list[Level], max_states: int) -> tuple[list[str], bool]:
    """Solve every level; returns printable lines and whether all were solved."""
    lines: list[str] = []
    all_ok = True
    for i, level in enumerate(levels, start=1):
        result = solve(level, max_states=max_states)
        if result.status is SolveStatus.SOLVED:
            detail = f"{result.pushes} pushes, {len(result.solution or '')} moves  {result.solution}"
        else:
            all_ok = False
            detail = f"{result.status.value} after {result.explored} positions"
        lines.append(f"{i:>3}. {level.name:<20} {result.status.value.upper():<10} {detail}")
    return lines, all_ok


def _positive(text: str) -> int:
    value = int(text)
    if value < 1:
        raise argparse.ArgumentTypeError("must be at least 1")
    return value


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="sokoban-undo", description=__doc__)
    parser.add_argument("--levels", type=Path, help="XSB level collection (default: bundled levels)")
    parser.add_argument("--level", type=_positive, default=1, help="level number to start on")
    parser.add_argument("--history-limit", type=_positive, help="cap undo depth (default: unlimited)")
    parser.add_argument("--max-states", type=_positive, default=GameConfig().solver_max_states,
                        help="solver search budget per hint/check")
    parser.add_argument("--check", action="store_true", help="solve every level, print solutions, and exit")
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    try:
        levels = load_levels(args.levels) if args.levels else bundled_levels()
    except LevelError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2

    if args.check:
        lines, ok = check_report(levels, args.max_states)
        print("\n".join(lines))
        return 0 if ok else 1

    if args.level > len(levels):
        print(f"error: --level {args.level} but the collection has {len(levels)} levels", file=sys.stderr)
        return 2
    config = GameConfig(history_limit=args.history_limit, solver_max_states=args.max_states)
    game = Game(levels, args.level - 1, config)
    try:
        from .terminal.app import run
    except ImportError:  # Windows Python ships without curses
        print("error: the terminal UI needs curses (pip install windows-curses on Windows)", file=sys.stderr)
        return 2
    run(game)
    return 0
