"""Headless checks of the pygame layer (SDL dummy video driver).

The core tests above don't need pygame; these skip cleanly if it's missing.
"""
import os

import pytest

os.environ.setdefault("SDL_VIDEODRIVER", "dummy")
os.environ.setdefault("SDL_AUDIODRIVER", "dummy")
pygame = pytest.importorskip("pygame")

from breakout.platform.app import run  # noqa: E402
from breakout.platform.pygame_input import PygameInputTracker  # noqa: E402


def test_tracker_translates_key_events_by_name():
    pygame.init()
    try:
        t = PygameInputTracker()
        t.handle(pygame.event.Event(pygame.KEYDOWN, key=pygame.K_SPACE))
        t.handle(pygame.event.Event(pygame.KEYDOWN, key=pygame.K_LEFT))
        assert t.snapshot().keys == {"space", "left"}
        t.handle(pygame.event.Event(pygame.KEYUP, key=pygame.K_SPACE))
        assert t.snapshot().keys == {"left"}
        t.handle(pygame.event.Event(pygame.WINDOWFOCUSLOST))
        assert t.snapshot().keys == frozenset()
    finally:
        pygame.quit()


def test_events_from_unknown_pads_are_ignored():
    t = PygameInputTracker()
    t.handle(pygame.event.Event(pygame.JOYBUTTONDOWN, instance_id=42, button=0))
    assert t.snapshot().buttons == frozenset()


def test_app_runs_headless_for_a_few_frames(tmp_path):
    assert run(["--frames", "30", "--seed", "1", "--bindings", str(tmp_path / "b.json")]) == 0
