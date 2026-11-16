"""Loading level collections from disk or from the package's bundled data."""
from __future__ import annotations

from importlib import resources
from pathlib import Path

from .core.level import Level, LevelError, parse_collection


def bundled_levels() -> list[Level]:
    text = resources.files("sokoban.data").joinpath("levels.xsb").read_text(encoding="utf-8")
    return parse_collection(text)


def load_levels(path: Path) -> list[Level]:
    """Load an XSB collection, turning I/O failures into ``LevelError``."""
    try:
        text = path.read_text(encoding="utf-8")
    except OSError as exc:
        raise LevelError(f"cannot read {path}: {exc.strerror or exc}") from exc
    return parse_collection(text)
