import json
import os

import pytest

from gamesettings.core import MIGRATIONS, SettingsFile


@pytest.fixture
def file(tmp_path, schema):
    return SettingsFile(tmp_path / "cfg" / "settings.json", schema, MIGRATIONS)


def test_missing_file_gives_defaults_silently(file, schema):
    report = file.load()
    assert report.values == schema.defaults() and report.warnings == ()


def test_save_creates_directories_and_round_trips(file, schema):
    values = schema.defaults() | {"audio.subtitles": True}
    file.save(values)
    assert file.load().values == values


def test_save_leaves_no_temp_files(file, schema):
    file.save(schema.defaults())
    file.save(schema.defaults() | {"audio.subtitles": True})
    assert [p.name for p in file.path.parent.iterdir()] == ["settings.json"]


def test_failed_replace_keeps_old_file_and_cleans_up(file, schema, monkeypatch):
    file.save(schema.defaults() | {"audio.subtitles": True})
    before = file.path.read_text()

    def boom(src, dst):
        raise OSError("disk full")

    monkeypatch.setattr(os, "replace", boom)
    with pytest.raises(OSError, match="disk full"):
        file.save(schema.defaults())
    assert file.path.read_text() == before
    assert [p.name for p in file.path.parent.iterdir()] == ["settings.json"]


@pytest.mark.parametrize("content", [b"{truncated", b"\xff\xfe\x00garbage"])
def test_corrupt_file_is_quarantined(file, schema, content):
    file.path.parent.mkdir(parents=True)
    file.path.write_bytes(content)
    report = file.load()
    assert report.values == schema.defaults()
    assert "settings.json.corrupt" in report.warnings[0]
    assert file.backup_path.read_bytes() == content
    assert not file.path.exists()


def test_v1_file_on_disk_upgrades_on_next_save(file, schema):
    file.path.parent.mkdir(parents=True)
    file.path.write_text(json.dumps({"version": 1, "settings": {"audio": {"volume": 0.5}}}))
    report = file.load()
    file.save(report.values)
    assert json.loads(file.path.read_text()) == {"version": 2, "settings": {"audio": {"master_volume": 50}}}


def test_unreadable_path_gives_defaults_with_warning(tmp_path, schema):
    blocker = tmp_path / "not_a_dir"
    blocker.write_text("")
    report = SettingsFile(blocker / "settings.json", schema).load()
    assert report.values == schema.defaults()
    assert "could not read settings" in report.warnings[0]


def test_quarantine_failure_still_boots(file, schema, monkeypatch):
    file.path.parent.mkdir(parents=True)
    file.path.write_text("{oops")

    def refuse(src, dst):
        raise PermissionError("locked")

    monkeypatch.setattr(os, "replace", refuse)
    report = file.load()
    assert report.values == schema.defaults()
    assert "could not move it aside" in report.warnings[0]
