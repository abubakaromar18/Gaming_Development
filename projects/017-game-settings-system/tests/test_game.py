import pytest

from gamesettings.core import PRESETS, SettingsStore, bindings, detect_preset, effective_volume, render_resolution


def test_defaults_match_high_preset(schema):
    assert detect_preset(schema.defaults()) == "high"


@pytest.mark.parametrize("name", list(PRESETS))
def test_every_preset_is_valid_and_detected(store, name):
    store.stage_many(PRESETS[name])
    store.apply()
    assert detect_preset(store.values) == name


def test_tweaking_one_option_becomes_custom(store):
    store.stage_many(PRESETS["low"])
    store.stage("graphics.shadow_quality", "ultra")
    assert detect_preset(store.preview_values()) == "custom"


def test_effective_volume_multiplies_master_and_channel(schema):
    values = schema.defaults() | {"audio.master_volume": 50, "audio.music_volume": 40}
    assert effective_volume(values, "music") == pytest.approx(0.2)
    assert effective_volume(values, "music", focused=False) == 0.0
    values["audio.mute_when_unfocused"] = False
    assert effective_volume(values, "music", focused=False) == pytest.approx(0.2)
    with pytest.raises(ValueError):
        effective_volume(values, "master")


@pytest.mark.parametrize("res,scale,expected", [
    ("1920x1080", 1.0, (1920, 1080)),
    ("1920x1080", 0.75, (1440, 810)),
    ("1366x768", 0.55, (752, 422)),
    ("3840x2160", 0.5, (1920, 1080)),
])
def test_render_resolution_is_even(schema, res, scale, expected):
    values = schema.defaults() | {"graphics.resolution": res, "graphics.render_scale": scale}
    assert render_resolution(values) == expected
    assert all(n % 2 == 0 for n in render_resolution(values))


def test_bindings_map(schema):
    store = SettingsStore(schema, {"controls.bind_jump": "F"})
    assert bindings(store.values)["jump"] == "F"
    assert len(set(bindings(store.values).values())) == len(bindings(store.values))
