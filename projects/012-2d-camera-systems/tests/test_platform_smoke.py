"""Headless checks of the pygame layer (SDL dummy drivers). Skipped if pygame is missing."""
import os

import pytest

os.environ.setdefault("SDL_VIDEODRIVER", "dummy")
os.environ.setdefault("SDL_AUDIODRIVER", "dummy")
pytest.importorskip("pygame")

from camera2d.platform.app import autopilot_controls, parse_args, run  # noqa: E402


def test_app_runs_headless_with_autopilot():
    assert run(["--frames", "240", "--seed", "3", "--width", "640", "--height", "360"]) == 0


def test_autopilot_is_deterministic_and_fires():
    assert autopilot_controls(10) == autopilot_controls(10)
    assert any(autopilot_controls(f).fire for f in range(120))


def test_cli_rejects_bad_sizes():
    with pytest.raises(SystemExit):
        parse_args(["--width", "0"])
    with pytest.raises(SystemExit):
        parse_args(["--frames", "-1"])
