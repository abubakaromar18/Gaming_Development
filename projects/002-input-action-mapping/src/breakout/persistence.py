"""Loading and saving bindings as JSON.

Saving is atomic (write a temp file, then rename over the old one), so a crash
mid-write can't leave a half-written bindings file. Loading never crashes the
game: a missing file means defaults, and a corrupt file means defaults plus a
message the shell can show the player.
"""
from __future__ import annotations

import json
import os
import tempfile
from dataclasses import dataclass
from pathlib import Path

from .core.bindings import ActionMap, default_bindings

FORMAT_VERSION = 1


@dataclass(frozen=True)
class LoadResult:
    action_map: ActionMap
    #: Human-readable reason the file was ignored, or None if it loaded fine.
    warning: str | None = None


def default_bindings_path() -> Path:
    base = os.environ.get("XDG_CONFIG_HOME") or Path.home() / ".config"
    return Path(base) / "breakout-actions" / "bindings.json"


def load_bindings(path: Path) -> LoadResult:
    defaults = default_bindings()
    if not path.exists():
        return LoadResult(defaults)
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
        if not isinstance(data, dict) or data.get("version") != FORMAT_VERSION:
            raise ValueError(f"expected an object with version {FORMAT_VERSION}")
        bindings = data.get("bindings")
        if not isinstance(bindings, dict):
            raise ValueError("missing 'bindings' object")
        return LoadResult(ActionMap.from_dict(bindings, defaults=defaults))
    except (OSError, ValueError) as exc:
        # json.JSONDecodeError is a ValueError subclass.
        return LoadResult(defaults, warning=f"Ignored {path.name}: {exc}")


def save_bindings(path: Path, action_map: ActionMap) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    payload = {"version": FORMAT_VERSION, "bindings": action_map.to_dict()}
    fd, tmp = tempfile.mkstemp(dir=path.parent, prefix=".bindings-", suffix=".json")
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as fh:
            json.dump(payload, fh, indent=2)
            fh.write("\n")
        os.replace(tmp, path)
    except BaseException:
        Path(tmp).unlink(missing_ok=True)
        raise
