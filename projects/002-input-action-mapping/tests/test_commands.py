import pytest

from breakout.core.bindings import default_bindings
from breakout.core.commands import gameplay_commands
from breakout.core.resolver import InputResolver
from helpers import keys, pad, raw


def commands_for(*frames):
    r = InputResolver(default_bindings())
    return [gameplay_commands(r.resolve(f)) for f in frames][-1]


def test_digital_move():
    assert commands_for(keys("left")).move == -1.0
    assert commands_for(keys("d")).move == 1.0


def test_opposite_directions_cancel():
    assert commands_for(keys("left", "right")).move == 0.0


def test_analog_move_is_proportional():
    assert commands_for(pad(axes={0: -0.6})).move == pytest.approx(-0.5)


def test_mixed_devices_clamp_to_unit_range():
    assert commands_for(raw(keys=["right"], axes={0: 1.0})).move == 1.0


def test_launch_and_pause_are_edges_not_levels():
    assert commands_for(keys("space")).launch
    assert not commands_for(keys("space"), keys("space")).launch
    assert commands_for(pad(buttons=[7])).toggle_pause
