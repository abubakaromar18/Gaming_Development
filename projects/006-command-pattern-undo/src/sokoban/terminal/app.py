"""curses main loop: read keys, tick auto-play with real dt, draw."""
from __future__ import annotations

import curses
import time

from ..core.game import Game
from ..core.geometry import Pos
from .controller import Controller
from .keymap import HELP, action_for

FRAME_SECONDS = 1 / 30
MAX_DT = 0.25  # clamp after a stall so auto-play doesn't jump ahead

# Two characters per cell keeps cells roughly square in a terminal.
GLYPHS = {"wall": "██", "floor": "  ", "goal": "··", "box": "[]", "box_goal": "{}", "player": "@@"}


def run(game: Game) -> None:
    curses.wrapper(_loop, Controller(game))


def _loop(screen: "curses.window", controller: Controller) -> None:
    curses.curs_set(0)
    screen.keypad(True)
    screen.timeout(int(FRAME_SECONDS * 1000))
    colors = _init_colors()
    last = time.monotonic()
    while True:
        _draw(screen, controller, colors)
        key = screen.getch()
        now = time.monotonic()
        controller.tick(min(now - last, MAX_DT))
        last = now
        if key == -1 or key == curses.KEY_RESIZE:
            continue
        action = action_for(curses.keyname(key).decode("ascii", "replace"))
        if action is not None and not controller.handle(action):
            return


def _init_colors() -> dict[str, int]:
    if not curses.has_colors():
        return {name: curses.A_NORMAL for name in GLYPHS}
    curses.start_color()
    curses.use_default_colors()
    pairs = {"wall": curses.COLOR_BLUE, "goal": curses.COLOR_RED, "box": curses.COLOR_YELLOW,
             "box_goal": curses.COLOR_GREEN, "player": curses.COLOR_CYAN}
    attrs = {"floor": curses.A_NORMAL}
    for i, (name, color) in enumerate(pairs.items(), start=1):
        curses.init_pair(i, color, -1)
        attrs[name] = curses.color_pair(i) | curses.A_BOLD
    return attrs


def _cell(game: Game, pos: Pos) -> str:
    level, board = game.level, game.board
    if pos in level.walls:
        return "wall"
    if pos == board.player:
        return "player"
    if board.has_box(pos):
        return "box_goal" if pos in level.goals else "box"
    return "goal" if pos in level.goals else "floor"


def _draw(screen: "curses.window", controller: Controller, colors: dict[str, int]) -> None:
    game = controller.game
    board, level, history = game.board, game.level, game.history
    screen.erase()
    rows, cols = screen.getmaxyx()

    def put(y: int, x: int, text: str, attr: int = curses.A_NORMAL) -> None:
        # Clip instead of crashing when the terminal is smaller than the level.
        if 0 <= y < rows - 1 and x < cols:
            screen.addstr(y, x, text[: max(cols - x - 1, 0)], attr)

    title = f" {game.index + 1}/{len(game.levels)}  {level.name} "
    put(0, 0, title, curses.A_REVERSE)
    stats = (f"moves {board.moves}  pushes {board.pushes}  goals {board.boxes_on_goals()}/{len(level.goals)}"
             f"  undo {history.undo_depth}  redo {history.redo_depth}")
    put(1, 0, stats)
    for r in range(level.height):
        for c in range(level.width):
            kind = _cell(game, Pos(r, c))
            if kind != "floor" or Pos(r, c) in level.interior:
                put(3 + r, 2 + c * 2, GLYPHS[kind], colors[kind])
    put(4 + level.height, 0, controller.message, curses.A_BOLD)
    put(6 + level.height, 0, HELP, curses.A_DIM)
    screen.refresh()
