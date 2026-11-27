import pytest

from gamesettings.core import BoolField, KeyField, Schema, UnknownSettingError, unique_values


def test_keys_are_dotted_and_ordered(schema):
    assert schema.keys[0] == "graphics.resolution"
    assert set(schema.sections) == {"graphics", "audio", "controls"}
    assert "audio.subtitles" in schema


def test_unknown_key_suggests_close_matches(schema):
    with pytest.raises(UnknownSettingError) as info:
        schema.field("graphics.vsnyc")
    assert "graphics.vsync" in info.value.suggestions
    assert "did you mean" in str(info.value)


def test_defaults_obey_rules(schema):
    assert schema.check(schema.defaults()) == ()


def test_duplicate_bindings_violate(schema):
    values = schema.defaults() | {"controls.bind_jump": "E"}
    (violation,) = schema.check(values)
    assert set(violation.keys) == {"controls.bind_jump", "controls.bind_interact"}


def test_fps_rule(schema):
    assert schema.check(schema.defaults() | {"graphics.fps_limit": 20})
    assert not schema.check(schema.defaults() | {"graphics.fps_limit": 30})


def test_schema_rejects_defaults_that_break_its_rules():
    with pytest.raises(ValueError, match="break its own rules"):
        Schema({"c": {"a": KeyField("E", "A"), "b": KeyField("E", "B")}}, version=1,
               rules=[unique_values(["c.a", "c.b"], "key")])


@pytest.mark.parametrize("bad", ["", "a.b", "has space"])
def test_schema_rejects_bad_names(bad):
    with pytest.raises(ValueError):
        Schema({bad: {"x": BoolField(True, "X")}}, version=1)


def test_version_must_be_positive():
    with pytest.raises(ValueError):
        Schema({}, version=0)
