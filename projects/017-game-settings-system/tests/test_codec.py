import json
import random

import pytest

from gamesettings.core import MIGRATIONS, CorruptSettingsError, decode, encode


def doc(settings, version=2):
    return json.dumps({"version": version, "settings": settings})


def test_defaults_encode_to_empty_settings(schema):
    assert json.loads(encode(schema.defaults(), schema)) == {"version": 2, "settings": {}}


def test_only_non_defaults_are_written(schema):
    values = schema.defaults() | {"graphics.vsync": False, "audio.music_volume": 40}
    assert json.loads(encode(values, schema))["settings"] == {
        "graphics": {"vsync": False}, "audio": {"music_volume": 40}}


def test_round_trip(schema):
    values = schema.defaults() | {"graphics.resolution": "2560x1440", "controls.mouse_sensitivity": 2.5,
                                  "controls.bind_jump": "F"}
    report = decode(encode(values, schema), schema)
    assert report.values == values and report.warnings == ()


@pytest.mark.parametrize("text", ["{not json", "[1, 2]", '"hi"', '{"version": "two"}', '{"version": 0}',
                                  '{"version": true}'])
def test_corrupt_documents_raise(schema, text):
    with pytest.raises(CorruptSettingsError):
        decode(text, schema)


def test_bad_values_fall_back_individually(schema):
    report = decode(doc({"graphics": {"vsync": "sometimes", "field_of_view": 75},
                         "audio": {"music_volume": "loud"}}), schema)
    assert report.values["graphics.vsync"] is True
    assert report.values["audio.music_volume"] == 70
    assert report.values["graphics.field_of_view"] == 75  # the good value survived
    assert len(report.warnings) == 2


def test_out_of_range_numbers_are_clamped_not_reset(schema):
    report = decode(doc({"graphics": {"field_of_view": 130}}), schema)
    assert report.values["graphics.field_of_view"] == 120
    assert "clamped" in report.warnings[0]


def test_unknown_keys_and_bad_sections_are_reported(schema):
    report = decode(doc({"graphics": {"ray_tracing": True}, "network": 5}), schema)
    assert report.values == schema.defaults()
    assert any("ray_tracing" in w for w in report.warnings)
    assert any("network" in w for w in report.warnings)


def test_conflicting_bindings_reset_only_the_custom_side(schema):
    report = decode(doc({"controls": {"bind_jump": "E", "bind_pause": "P"}}), schema)
    assert report.values["controls.bind_jump"] == "SPACE"
    assert report.values["controls.bind_interact"] == "E"
    assert report.values["controls.bind_pause"] == "P"
    assert "reset to defaults" in report.warnings[0]


def test_v1_file_is_migrated(schema):
    v1 = json.dumps({"version": 1, "settings": {"graphics": {"fullscreen": True, "vsync": False},
                                                "audio": {"volume": 0.35}}})
    report = decode(v1, schema, MIGRATIONS)
    assert report.migrated_from == 1
    assert report.values["graphics.window_mode"] == "fullscreen"
    assert report.values["audio.master_volume"] == 35
    assert report.values["graphics.vsync"] is False
    assert report.warnings == ()


def test_unversioned_file_counts_as_v1(schema):
    report = decode(json.dumps({"settings": {"graphics": {"fullscreen": False}}}), schema, MIGRATIONS)
    assert report.values["graphics.window_mode"] == "windowed"


def test_migration_leaves_odd_shapes_for_validation(schema):
    report = decode(doc({"audio": {"volume": "max"}}, version=1), schema, MIGRATIONS)
    assert report.values["audio.master_volume"] == 80
    assert any("audio.master_volume" in w and "max" in w for w in report.warnings)


@pytest.mark.parametrize("raw", ["NaN", "Infinity", "1e999"])
def test_non_finite_numbers_are_rejected_not_crashing(schema, raw):
    text = f'{{"version": 1, "settings": {{"audio": {{"volume": {raw}}}, "graphics": {{"brightness": {raw}}}}}}}'
    report = decode(text, schema, MIGRATIONS)
    assert report.values == schema.defaults()
    assert len(report.warnings) == 2


def test_missing_migration_resets_with_warning(schema):
    report = decode(doc({"graphics": {"vsync": False}}, version=1), schema, migrations={})
    assert report.values == schema.defaults()
    assert "no migration" in report.warnings[0]


def test_newer_file_loads_best_effort(schema):
    report = decode(doc({"graphics": {"vsync": False, "hdr": True}}, version=3), schema)
    assert report.values["graphics.vsync"] is False
    assert "newer version" in report.warnings[0]


def _junk(rng, depth=0):
    kind = rng.randrange(8 if depth < 3 else 5)
    return [lambda: None, lambda: rng.random() > 0.5, lambda: rng.randint(-1000, 1000),
            lambda: rng.uniform(-1e6, 1e6), lambda: rng.choice(["", "on", "W", "1920x1080", "ultra", "ß"]),
            lambda: [_junk(rng, depth + 1) for _ in range(rng.randrange(3))],
            lambda: {rng.choice(["vsync", "bind_jump", "field_of_view", "x"]): _junk(rng, depth + 1)},
            lambda: {s: _junk(rng, depth + 1) for s in rng.sample(["graphics", "audio", "controls"], 2)}][kind]()


def test_fuzz_never_crashes_and_always_yields_valid_values(schema):
    """Seeded fuzz: any JSON object decodes to a complete, rule-abiding set."""
    rng = random.Random(1234)
    for _ in range(2000):
        settings = {s: {k.split(".")[1]: _junk(rng) for k in rng.sample(schema.sections[s], 3)}
                    for s in schema.sections}
        if rng.random() < 0.2:
            settings["controls"] = {f"bind_{a}": rng.choice("WASDEQ") for a in ("jump", "interact", "pause")}
        report = decode(doc(settings, version=rng.choice([1, 2, 3])), schema, MIGRATIONS)
        assert set(report.values) == set(schema.keys)
        assert schema.check(report.values) == ()
        for key, value in report.values.items():
            assert schema.field(key).coerce(value) == value
