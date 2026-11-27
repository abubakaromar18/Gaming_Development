"""Where each OS expects per-user config files to live.

Pure function of (platform, environment, home) so it can be tested for all
three OSes from any one of them.
"""
from __future__ import annotations

import os
import sys
from collections.abc import Mapping
from pathlib import Path, PurePath, PurePosixPath, PureWindowsPath

APP_DIR = "GameSettingsDemo"
FILE_NAME = "settings.json"


def default_config_path(platform: str, env: Mapping[str, str], home: PurePath) -> PurePath:
    if platform.startswith("win"):
        base = PureWindowsPath(env["APPDATA"]) if env.get("APPDATA") else PureWindowsPath(home, "AppData", "Roaming")
        return base / APP_DIR / FILE_NAME
    if platform == "darwin":
        return PurePosixPath(home, "Library", "Application Support", APP_DIR, FILE_NAME)
    # Linux / BSD: XDG Base Directory spec. Relative XDG paths must be ignored.
    xdg = env.get("XDG_CONFIG_HOME", "")
    base = PurePosixPath(xdg) if xdg.startswith("/") else PurePosixPath(home, ".config")
    return base / APP_DIR.lower() / FILE_NAME


def current_config_path() -> Path:
    return Path(default_config_path(sys.platform, os.environ, Path.home()))
