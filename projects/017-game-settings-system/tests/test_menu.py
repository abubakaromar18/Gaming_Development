import pytest

from gamesettings.platform.menu import SettingsMenu, run


class FakeClock:
    def __init__(self):
        self.now = 0.0

    def __call__(self):
        return self.now


@pytest.fixture
def saved():
    return []


@pytest.fixture
def menu(store, saved):
    return SettingsMenu(store, lambda values: saved.append(dict(values)))


def drive(menu, script, clock=None, gaps=None):
    """Feed commands; ``gaps[i]`` seconds pass before command i arrives."""
    clock = clock or FakeClock()
    lines = iter(script)
    gaps = list(gaps or [0.0] * len(script))
    screens = []

    def read_line():
        clock.now += gaps.pop(0) if gaps else 0.0
        return next(lines)

    run(menu, read_line, screens.append, clock)
    return screens


def test_render_shows_tabs_cursor_and_preset(menu):
    screen = menu.render()
    assert "GRAPHICS" in screen and "Preset: high" in screen
    assert "> " in screen and "Resolution" in screen and "[confirm]" in screen


def test_adjust_marks_pending_then_apply_saves(menu, store, saved):
    for cmd in ["j", "j", "l"]:  # V-Sync off
        menu.handle(cmd)
    assert "*" in menu.render() and saved == []
    menu.handle("a")
    assert store.get("graphics.vsync") is False
    assert saved[-1]["graphics.vsync"] is False


def test_typed_value_and_validation_message(menu, store):
    menu.handle("3")
    for _ in range(6):
        menu.handle("j")  # Jump binding
    menu.handle("= e")
    menu.handle("a")
    assert "is used by" in menu.message
    menu.handle("= hyperdrive")
    assert menu.message.startswith("! controls.bind_jump")


def test_resolution_countdown_reverts_without_saving(menu, store, saved):
    screens = drive(menu, ["l", "a", "j", "q"], gaps=[0, 0, 16, 0])
    assert any("Reverting in 15s" in s for s in screens)
    assert store.get("graphics.resolution") == "1920x1080"
    assert saved == []


def test_late_yes_does_not_keep_reverted_change(menu, store):
    drive(menu, ["l", "a", "y", "q"], gaps=[0, 0, 30, 0])
    assert store.get("graphics.resolution") == "1920x1080"


def test_confirming_in_time_saves(menu, store, saved):
    drive(menu, ["l", "a", "y", "q"], gaps=[0, 0, 5, 0])
    assert store.get("graphics.resolution") == "2560x1440"
    assert saved[-1]["graphics.resolution"] == "2560x1440"


def test_quit_reverts_unconfirmed_and_drops_pending(menu, store):
    menu.handle("l")
    menu.handle("a")
    menu.handle("q")
    assert store.get("graphics.resolution") == "1920x1080" and not menu.running


def test_preset_reset_and_discard(menu, store):
    menu.handle("p low")
    assert "Preset: low" in menu.render()
    menu.handle("a")
    assert "Restart required for: Texture quality" in menu.message
    menu.handle("r")
    menu.handle("a")
    assert store.values == store.schema.defaults()
    menu.handle("p potato")
    assert "unknown preset" in menu.message
    menu.handle("2")
    menu.handle("l")
    menu.handle("d")
    assert not store.pending


def test_save_failure_is_reported_and_retried(store):
    attempts = []

    def flaky(values):
        attempts.append(values)
        if len(attempts) == 1:
            raise OSError("read-only")

    menu = SettingsMenu(store, flaky)
    menu.handle("2")
    menu.handle("l")
    menu.handle("a")
    assert "could not save" in menu.message
    menu.handle("j")  # any later command retries the save
    assert len(attempts) == 2


def test_eof_quits(menu):
    def read_line():
        raise EOFError

    run(menu, read_line, lambda s: None, FakeClock())
    assert not menu.running


def test_unknown_command(menu):
    menu.handle("xyz")
    assert "unknown command" in menu.message
