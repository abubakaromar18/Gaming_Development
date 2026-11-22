"""pygame shell: window, input, main loop, CLI.

This is the only module that reads the wall clock (via ``pygame.time.Clock``)
or keyboard state. It converts them into ``dt`` and ``Controls`` for the core.
"""
from __future__ import annotations

import argparse
import math
import random
import sys

import pygame

from ..core import FEATURES, Camera, CameraConfig, Controls, Vec2, World, WorldConfig, trauma_for
from .renderer import Renderer

MAX_DT = 1 / 20          # clamp long frames (window drag, breakpoint) so physics stays sane
HEADLESS_DT = 1 / 60
MANUAL_TRAUMA = 0.3
ZOOM_STEP = 1.1

FEATURE_KEYS = dict(zip((pygame.K_1, pygame.K_2, pygame.K_3, pygame.K_4, pygame.K_5), FEATURES))


def parse_args(argv: list[str] | None) -> argparse.Namespace:
    p = argparse.ArgumentParser(prog="camera2d", description="2D camera systems demo")
    p.add_argument("--seed", type=int, default=12, help="world generation and shake noise seed")
    p.add_argument("--width", type=int, default=1280)
    p.add_argument("--height", type=int, default=720)
    p.add_argument(
        "--frames",
        type=int,
        default=0,
        help="run N frames at a fixed 1/60 s with an autopilot, then exit (headless smoke test)",
    )
    args = p.parse_args(argv)
    if args.width <= 0 or args.height <= 0:
        p.error("--width and --height must be positive")
    if args.frames < 0:
        p.error("--frames must be >= 0")
    return args


def keyboard_controls(fire: bool) -> Controls:
    keys = pygame.key.get_pressed()
    x = (keys[pygame.K_RIGHT] or keys[pygame.K_d]) - (keys[pygame.K_LEFT] or keys[pygame.K_a])
    y = (keys[pygame.K_DOWN] or keys[pygame.K_s]) - (keys[pygame.K_UP] or keys[pygame.K_w])
    return Controls(Vec2(float(x), float(y)), fire)


def autopilot_controls(frame: int) -> Controls:
    """Deterministic figure-eight with periodic shockwaves, for headless runs."""
    t = frame * HEADLESS_DT
    return Controls(Vec2(math.cos(t * 1.3), math.sin(t * 2.6)), fire=frame % 90 == 45)


def run(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    pygame.init()
    try:
        screen = pygame.display.set_mode((args.width, args.height))
        pygame.display.set_caption("2D Camera Systems")
        rng = random.Random(args.seed)
        world_config = WorldConfig()
        world = World.generate(world_config, rng)
        camera = Camera(args.width, args.height, CameraConfig(), rng, world.bounds)
        camera.snap_to(world.position)
        renderer = Renderer(screen)
        clock = pygame.time.Clock()
        show_debug = True
        frame = 0

        while args.frames == 0 or frame < args.frames:
            fire = False
            for event in pygame.event.get():
                if event.type == pygame.QUIT or (event.type == pygame.KEYDOWN and event.key == pygame.K_ESCAPE):
                    return 0
                if event.type == pygame.KEYDOWN:
                    if event.key == pygame.K_SPACE:
                        fire = True
                    elif event.key in FEATURE_KEYS:
                        camera.config = camera.config.toggled(FEATURE_KEYS[event.key])
                    elif event.key == pygame.K_TAB:
                        show_debug = not show_debug
                    elif event.key == pygame.K_t:
                        camera.add_trauma(MANUAL_TRAUMA)
                    elif event.key == pygame.K_r:
                        world = World.generate(world_config, random.Random(args.seed))
                        camera.trauma = 0.0
                        camera.set_zoom(1.0)
                        camera.snap_to(world.position)
                elif event.type == pygame.MOUSEWHEEL and event.y:
                    mx, my = pygame.mouse.get_pos()
                    camera.zoom_at(Vec2(float(mx), float(my)), camera.zoom * ZOOM_STEP ** event.y)

            if args.frames:
                dt = HEADLESS_DT
                controls = autopilot_controls(frame)
            else:
                dt = min(clock.tick(120) / 1000.0, MAX_DT)
                controls = keyboard_controls(fire)

            for gameplay_event in world.update(dt, controls):
                amount = trauma_for(gameplay_event)
                if amount > 0:
                    camera.add_trauma(amount)
            camera.update(dt, world.position, world.velocity)

            renderer.draw(world, camera, show_debug, clock.get_fps())
            pygame.display.flip()
            frame += 1
        return 0
    finally:
        pygame.quit()


def main() -> None:
    sys.exit(run())
