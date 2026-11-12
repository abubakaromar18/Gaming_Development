import json

from breakout.core.actions import Action
from breakout.core.bindings import Key, default_bindings
from breakout.persistence import FORMAT_VERSION, load_bindings, save_bindings


def test_missing_file_gives_defaults_without_warning(tmp_path):
    result = load_bindings(tmp_path / "nope.json")
    assert result.action_map == default_bindings()
    assert result.warning is None


def test_save_then_load_round_trips(tmp_path):
    path = tmp_path / "sub" / "bindings.json"
    m = default_bindings()
    m.rebind(Action.LAUNCH, Key("z"))
    save_bindings(path, m)
    assert load_bindings(path).action_map == m
    assert json.loads(path.read_text())["version"] == FORMAT_VERSION
    assert [p.name for p in path.parent.iterdir()] == ["bindings.json"], "temp file left behind"


def test_corrupt_file_falls_back_with_warning(tmp_path):
    path = tmp_path / "bindings.json"
    for bad in ["{not json", "[]", json.dumps({"version": 99, "bindings": {}}),
                json.dumps({"version": FORMAT_VERSION, "bindings": {"launch": ["pad:wat"]}})]:
        path.write_text(bad)
        result = load_bindings(path)
        assert result.action_map == default_bindings()
        assert result.warning and "bindings.json" in result.warning


def test_partial_file_keeps_defaults_for_missing_actions(tmp_path):
    path = tmp_path / "bindings.json"
    path.write_text(json.dumps({"version": FORMAT_VERSION, "bindings": {"pause": ["key:tab"]}}))
    m = load_bindings(path).action_map
    assert m.sources_for(Action.PAUSE) == (Key("tab"),)
    assert m.sources_for(Action.LAUNCH) == default_bindings().sources_for(Action.LAUNCH)
