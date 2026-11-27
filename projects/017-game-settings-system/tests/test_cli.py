import io
import json

import pytest

from gamesettings.platform.cli import main


@pytest.fixture
def cfg(tmp_path):
    return tmp_path / "settings.json"


def cli(cfg, *args, stdin=""):
    out, err = io.StringIO(), io.StringIO()
    code = main(["--config", str(cfg), *args], stdin=io.StringIO(stdin), stdout=out, stderr=err)
    return code, out.getvalue(), err.getvalue()


def test_show_defaults(cfg):
    code, out, _ = cli(cfg)
    assert code == 0 and "graphics.vsync" in out and "preset: high" in out
    assert not cfg.exists()


def test_set_get_and_persist(cfg):
    code, out, _ = cli(cfg, "set", "graphics.vsync=off", "audio.music_volume=40%")
    assert code == 0 and "graphics.vsync: on -> off" in out
    assert json.loads(cfg.read_text())["settings"] == {"graphics": {"vsync": False}, "audio": {"music_volume": 40}}
    assert cli(cfg, "get", "audio.music_volume")[1].strip() == "40%"


def test_invalid_value_exits_2_and_writes_nothing(cfg):
    code, _, err = cli(cfg, "set", "graphics.field_of_view=500")
    assert code == 2 and "outside" in err and not cfg.exists()


def test_conflict_and_typo_messages(cfg):
    assert "is used by" in cli(cfg, "set", "controls.bind_jump=E")[2]
    assert "did you mean graphics.vsync" in cli(cfg, "get", "graphics.vsnyc")[2]
    assert cli(cfg, "set", "novalue")[0] == 2
    assert cli(cfg, "show", "network")[0] == 2


def test_preset_reset_and_restart_notice(cfg):
    code, out, _ = cli(cfg, "preset", "low")
    assert code == 0 and "restart required for: graphics.texture_quality" in out
    assert "preset: low" in cli(cfg, "show", "graphics")[1]
    cli(cfg, "reset")
    assert json.loads(cfg.read_text())["settings"] == {}
    assert "no changes" in cli(cfg, "reset")[1]


def test_warnings_go_to_stderr(cfg):
    cfg.write_text(json.dumps({"version": 1, "settings": {"graphics": {"fullscreen": True, "bogus": 1}}}))
    code, out, err = cli(cfg, "get", "graphics.window_mode")
    assert out.strip() == "fullscreen"
    assert "ignored unknown setting graphics.bogus" in err and "upgraded settings from version 1" in err


def test_unwritable_location_exits_1(tmp_path):
    blocker = tmp_path / "file"
    blocker.write_text("")
    code, _, err = cli(blocker / "settings.json", "set", "graphics.vsync=off")
    assert code == 1 and "could not save" in err


def test_menu_session_over_stdin(cfg):
    code, out, _ = cli(cfg, "menu", stdin="2\nh\na\nq\n")
    assert code == 0 and "Applied 1 change(s)." in out
    assert json.loads(cfg.read_text())["settings"] == {"audio": {"master_volume": 75}}


def test_path_command(cfg):
    assert cli(cfg, "path")[1].strip() == str(cfg)
