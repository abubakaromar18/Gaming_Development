import pytest

from breakout.core.actions import Action
from breakout.core.bindings import ActionMap, Key, PadAxis, default_bindings
from breakout.core.resolver import NO_INPUT, InputResolver, ResolverConfig, axis_value
from helpers import keys, pad, raw


def test_key_press_produces_held_and_edge():
    r = InputResolver(default_bindings())
    s = r.resolve(keys("space"))
    assert s.held(Action.LAUNCH) and s.just_pressed(Action.LAUNCH)
    s = r.resolve(keys("space"))
    assert s.held(Action.LAUNCH) and not s.just_pressed(Action.LAUNCH)
    s = r.resolve(NO_INPUT)
    assert not s.held(Action.LAUNCH) and s.just_released(Action.LAUNCH)


def test_any_bound_source_drives_the_action():
    r = InputResolver(default_bindings())
    assert r.resolve(keys("a")).held(Action.MOVE_LEFT)
    assert r.resolve(keys("left")).held(Action.MOVE_LEFT)
    assert r.resolve(pad(hats=[(0, "left")])).held(Action.MOVE_LEFT)
    assert r.resolve(pad(buttons=[0])).held(Action.LAUNCH)


def test_holding_a_second_source_does_not_refire_the_edge():
    r = InputResolver(default_bindings())
    r.resolve(keys("left"))
    s = r.resolve(keys("left", "a"))
    assert not s.just_pressed(Action.MOVE_LEFT)


def test_one_key_can_drive_actions_in_different_contexts():
    s = InputResolver(default_bindings()).resolve(keys("space"))
    assert s.just_pressed(Action.LAUNCH) and s.just_pressed(Action.CONFIRM)


@pytest.mark.parametrize(
    "raw, direction, expected",
    [(0.1, 1, 0.0), (0.2, 1, 0.0), (0.6, 1, 0.5), (1.0, 1, 1.0), (-1.0, 1, 0.0), (-1.0, -1, 1.0), (1.3, 1, 1.0)],
)
def test_axis_deadzone_is_rescaled(raw, direction, expected):
    assert axis_value(raw, direction, 0.2) == pytest.approx(expected)


def test_stick_gives_analog_value():
    s = InputResolver(default_bindings()).resolve(pad(axes={0: 0.6}))
    assert s.value(Action.MOVE_RIGHT) == pytest.approx(0.5)
    assert s.value(Action.MOVE_LEFT) == 0.0


def test_value_is_max_over_sources():
    s = InputResolver(default_bindings()).resolve(raw(keys=["d"], axes={0: 0.4}))
    assert s.value(Action.MOVE_RIGHT) == 1.0


def test_hysteresis_prevents_chatter_near_threshold():
    m = ActionMap({Action.MOVE_RIGHT: [PadAxis(0, 1)]})
    r = InputResolver(m, ResolverConfig(deadzone=0.0, press_threshold=0.5, release_threshold=0.3))
    assert not r.resolve(pad(axes={0: 0.45})).held(Action.MOVE_RIGHT)
    assert r.resolve(pad(axes={0: 0.55})).just_pressed(Action.MOVE_RIGHT)
    # Dips below the press threshold but above release: still held, no new edges.
    s = r.resolve(pad(axes={0: 0.4}))
    assert s.held(Action.MOVE_RIGHT) and not s.just_released(Action.MOVE_RIGHT)
    s = r.resolve(pad(axes={0: 0.52}))
    assert not s.just_pressed(Action.MOVE_RIGHT)
    assert r.resolve(pad(axes={0: 0.2})).just_released(Action.MOVE_RIGHT)


def test_rebinding_is_seen_by_the_resolver_immediately():
    m = default_bindings()
    r = InputResolver(m)
    m.rebind(Action.LAUNCH, Key("z"))
    assert r.resolve(keys("z")).just_pressed(Action.LAUNCH)
    assert not r.resolve(keys("space")).held(Action.LAUNCH)


def test_escape_held_across_a_context_switch_does_not_fire_twice():
    # Escape is BACK in menus and QUIT in gameplay. Leaving the menu with it
    # must not quit on the next frame: the shared held-set suppresses the edge.
    r = InputResolver(default_bindings())
    assert r.resolve(keys("escape")).just_pressed(Action.BACK)
    assert not r.resolve(keys("escape")).just_pressed(Action.QUIT)


@pytest.mark.parametrize(
    "kwargs", [{"deadzone": 1.0}, {"deadzone": -0.1}, {"press_threshold": 0.3, "release_threshold": 0.5}]
)
def test_invalid_config_is_rejected(kwargs):
    with pytest.raises(ValueError):
        ResolverConfig(**kwargs)
