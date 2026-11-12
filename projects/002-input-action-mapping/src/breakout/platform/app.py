"""The imperative shell: window, clock, event pump, and screen switching.

Per frame:  pygame events -> tracker -> RawInput -> resolver -> ActionState
            -> (gameplay) Commands -> Breakout.update
            -> (controls) BindingsMenu.update
            -> renderer

Everything after ``RawInput`` is pure core code.
"""
from __future__ import annotations

import argparse
import random
import sys
from enum import Enum
from pathlib import Path

import pygame

from ..core.actions import Action
from ..core.bindings import default_bindings
from ..core.breakout import Breakout, Event, GameConfig
from ..core.commands import gameplay_commands
from ..core.menu import BindingsMenu, MenuOutcome
from ..core.resolver import InputResolver
from ..persistence import default_bindings_path, load_bindings, save_bindings
from .pygame_input import PygameInputTracker
from .renderer import Renderer

#: Longest frame the simulation will accept. A stall (window drag, breakpoint)
#: becomes a brief slowdown instead of a teleporting ball.
MAX_FRAME_SECONDS = 1 / 20
FLASH_SECONDS = 0.35


class Screen(Enum):
    PLAY = "play"
    CONTROLS = "controls"


def parse_args(argv: list[str] | None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Breakout with rebindable keyboard/gamepad controls.")
    parser.add_argument("--bindings", type=Path, default=default_bindings_path(), help="bindings JSON file")
    parser.add_argument("--seed", type=int, default=None, help="RNG seed for reproducible serves")
    parser.add_argument("--frames", type=int, default=None, help="quit after N frames (smoke tests)")
    return parser.parse_args(argv)


def run(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    pygame.init()
    pygame.joystick.init()
    try:
        cfg = GameConfig()
        surface = pygame.display.set_mode((int(cfg.width), int(cfg.height)))
        pygame.display.set_caption("Breakout: Input Action Mapping")
        clock = pygame.time.Clock()

        loaded = load_bindings(args.bindings)
        action_map = loaded.action_map
        resolver = InputResolver(action_map)
        tracker = PygameInputTracker()
        game = Breakout(random.Random(args.seed), cfg)
        renderer = Renderer(surface)
        menu: BindingsMenu | None = None
        screen = Screen.PLAY
        flash = 0.0
        status = loaded.warning
        if status:
            print(f"warning: {status}", file=sys.stderr)

        frame = 0
        while args.frames is None or frame < args.frames:
            frame += 1
            dt = min(clock.tick(120) / 1000.0, MAX_FRAME_SECONDS)

            for event in pygame.event.get():
                if event.type == pygame.QUIT:
                    return 0
                tracker.handle(event)
            raw = tracker.snapshot()
            actions = resolver.resolve(raw)

            if screen is Screen.PLAY:
                if actions.just_pressed(Action.QUIT):
                    return 0
                if actions.just_pressed(Action.OPEN_BINDINGS):
                    screen = Screen.CONTROLS
                    menu = BindingsMenu(action_map, default_bindings)
                    if status:
                        menu.message = status
                        status = None
                    game.paused = True
                    continue
                events = game.update(gameplay_commands(actions), dt)
                if Event.LIFE_LOST in events:
                    flash = FLASH_SECONDS
                flash = max(0.0, flash - dt)
                renderer.draw_game(game, action_map, flash / FLASH_SECONDS)
            else:
                assert menu is not None
                if menu.update(actions, raw, dt) is MenuOutcome.CLOSE:
                    if menu.dirty:
                        try:
                            save_bindings(args.bindings, action_map)
                        except OSError as exc:
                            print(f"warning: could not save bindings: {exc}", file=sys.stderr)
                    # The game stays paused; the player resumes with their
                    # (possibly new) pause binding, which the HUD now shows.
                    screen, menu = Screen.PLAY, None
                    continue
                renderer.draw_menu(menu, tracker.pad_names)

            pygame.display.flip()
        return 0
    finally:
        pygame.quit()


def main() -> None:
    sys.exit(run())
