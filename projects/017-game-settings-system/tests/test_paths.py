from pathlib import PurePosixPath, PureWindowsPath

from gamesettings.platform.paths import default_config_path


def test_windows_uses_appdata():
    path = default_config_path("win32", {"APPDATA": r"C:\Users\p\AppData\Roaming"}, PureWindowsPath(r"C:\Users\p"))
    assert path == PureWindowsPath(r"C:\Users\p\AppData\Roaming\GameSettingsDemo\settings.json")


def test_windows_without_appdata_falls_back_to_home():
    path = default_config_path("win32", {}, PureWindowsPath(r"C:\Users\p"))
    assert path == PureWindowsPath(r"C:\Users\p\AppData\Roaming\GameSettingsDemo\settings.json")


def test_macos():
    path = default_config_path("darwin", {}, PurePosixPath("/Users/p"))
    assert str(path) == "/Users/p/Library/Application Support/GameSettingsDemo/settings.json"


def test_linux_honours_absolute_xdg_only():
    home = PurePosixPath("/home/p")
    assert str(default_config_path("linux", {"XDG_CONFIG_HOME": "/cfg"}, home)) == "/cfg/gamesettingsdemo/settings.json"
    assert str(default_config_path("linux", {"XDG_CONFIG_HOME": "rel"}, home)) == \
        "/home/p/.config/gamesettingsdemo/settings.json"
    assert str(default_config_path("linux", {}, home)) == "/home/p/.config/gamesettingsdemo/settings.json"
