import pytest

from breakout.core.actions import REBINDABLE, Action, Context
from breakout.core.bindings import (
    ActionMap,
    DeviceClass,
    Key,
    PadAxis,
    PadButton,
    PadHat,
    default_bindings,
    describe_source,
    device_class,
    format_source,
    parse_source,
)


@pytest.mark.parametrize(
    "source",
    [Key("space"), Key("f1"), PadButton(0), PadButton(12), PadAxis(0, -1), PadAxis(3, 1), PadHat(0, "up")],
)
def test_sources_round_trip_through_text(source):
    assert parse_source(format_source(source)) == source


@pytest.mark.parametrize(
    "text",
    ["", "key:", "mouse:left", "pad:button:x", "pad:button:-1", "pad:axis:0", "pad:axis:0:*", "pad:hat:0:north"],
)
def test_malformed_sources_are_rejected(text):
    with pytest.raises(ValueError):
        parse_source(text)


def test_parse_normalises_key_case():
    assert parse_source("key:SPACE") == Key("space")


def test_device_class():
    assert device_class(Key("a")) is DeviceClass.KEYBOARD
    assert device_class(PadAxis(0, 1)) is DeviceClass.GAMEPAD
    assert device_class(PadHat(0, "left")) is DeviceClass.GAMEPAD


def test_describe_source_is_human_readable():
    assert describe_source(Key("a")) == "A"
    assert describe_source(Key("space")) == "Space"
    assert describe_source(PadHat(0, "left")) == "D-pad left"
    assert describe_source(PadAxis(1, -1)) == "Stick 1-"


def test_every_action_has_a_default_binding():
    defaults = default_bindings()
    for action in Action:
        assert defaults.sources_for(action), action


def test_same_source_may_be_shared_across_contexts():
    defaults = default_bindings()
    # Space launches in gameplay and confirms in menus; that's not a conflict.
    assert defaults.action_for(Key("space"), Context.GAMEPLAY) is Action.LAUNCH
    assert defaults.action_for(Key("space"), Context.MENU) is Action.CONFIRM


def test_conflicting_bindings_in_one_context_are_rejected():
    with pytest.raises(ValueError, match="both"):
        ActionMap({Action.LAUNCH: [Key("space")], Action.PAUSE: [Key("space")]})


def test_rebind_replaces_only_the_same_device_class():
    m = default_bindings()
    result = m.rebind(Action.MOVE_LEFT, Key("j"))
    assert m.sources_for(Action.MOVE_LEFT) == (PadAxis(0, -1), PadHat(0, "left"), Key("j"))
    assert set(result.replaced) == {Key("left"), Key("a")}
    assert result.stolen_from == ()


def test_rebinding_a_used_source_moves_it_and_reports_it():
    m = default_bindings()
    result = m.rebind(Action.PAUSE, Key("space"))
    assert result.stolen_from == (Action.LAUNCH,)
    assert Key("space") not in m.sources_for(Action.LAUNCH)
    assert m.action_for(Key("space"), Context.GAMEPLAY) is Action.PAUSE
    # Launch still has its gamepad button, so it isn't unbound.
    assert result.now_unbound == ()
    # Menu context is untouched.
    assert m.action_for(Key("space"), Context.MENU) is Action.CONFIRM


def test_rebind_reports_actions_left_unbound():
    m = default_bindings()
    result = m.rebind(Action.LAUNCH, Key("escape"))
    assert result.now_unbound == (Action.QUIT,)
    assert m.sources_for(Action.QUIT) == ()


def test_rebinding_to_the_current_source_is_a_no_op():
    m = default_bindings()
    result = m.rebind(Action.LAUNCH, Key("space"))
    assert result.replaced == () and result.stolen_from == ()
    assert set(m.sources_for(Action.LAUNCH)) == {PadButton(0), Key("space")}


def test_dict_round_trip():
    m = default_bindings()
    m.rebind(Action.LAUNCH, PadButton(3))
    assert ActionMap.from_dict(m.to_dict()) == m


def test_from_dict_fills_missing_actions_from_defaults():
    m = ActionMap.from_dict({"launch": ["key:z"]}, defaults=default_bindings())
    assert m.sources_for(Action.LAUNCH) == (Key("z"),)
    assert m.sources_for(Action.PAUSE) == default_bindings().sources_for(Action.PAUSE)


@pytest.mark.parametrize("data", [{"teleport": ["key:t"]}, {"launch": "key:z"}, {"launch": ["key:"]}])
def test_from_dict_rejects_bad_data(data):
    with pytest.raises(ValueError):
        ActionMap.from_dict(data, defaults=default_bindings())


def test_assign_keeps_identity():
    m = default_bindings()
    other = ActionMap({Action.LAUNCH: [Key("z")]})
    m.assign(other)
    assert m.sources_for(Action.LAUNCH) == (Key("z"),)
    assert m.sources_for(Action.PAUSE) == ()


def test_menu_actions_are_not_rebindable():
    assert all(a.context is Context.GAMEPLAY for a in REBINDABLE)
    assert Action.CONFIRM not in REBINDABLE
