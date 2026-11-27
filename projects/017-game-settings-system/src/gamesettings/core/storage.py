"""Settings persistence on a real filesystem.

Two failure modes matter for a settings file:

1. **Torn writes.** If the game crashes (or the console is switched off)
   halfway through writing, a naive ``open(path, "w")`` leaves a truncated
   file. We write to a temporary file in the same directory, ``fsync`` it,
   then ``os.replace`` it over the old one. ``os.replace`` is atomic on POSIX
   and Windows, so readers see either the old file or the new one, never half.
2. **Corrupt files.** If the file is unreadable anyway (disk error, a bad
   hand edit), the game must still boot. We move it aside to
   ``<name>.corrupt`` (so support can inspect it and the player can recover
   it) and continue with defaults.
"""
from __future__ import annotations

import os
import tempfile
from collections.abc import Mapping
from pathlib import Path
from typing import Any

from .codec import LoadReport, Migration, decode, encode
from .errors import CorruptSettingsError
from .schema import Schema


class SettingsFile:
    def __init__(self, path: Path | str, schema: Schema, migrations: Mapping[int, Migration] | None = None) -> None:
        self.path = Path(path)
        self.schema = schema
        self.migrations = dict(migrations or {})

    @property
    def backup_path(self) -> Path:
        return self.path.with_name(self.path.name + ".corrupt")

    def load(self) -> LoadReport:
        """Load settings. Never raises: a game must boot even when its
        settings file is missing, corrupt, or unreadable."""
        try:
            text = self.path.read_text(encoding="utf-8")
        except FileNotFoundError:
            return LoadReport(self.schema.defaults())
        except UnicodeDecodeError:
            return self._quarantine("file is not UTF-8 text")
        except OSError as exc:  # permissions, a directory in the way, I/O error
            return LoadReport(self.schema.defaults(),
                              (f"could not read settings ({exc.strerror or exc}); using defaults",))
        try:
            return decode(text, self.schema, self.migrations)
        except CorruptSettingsError as exc:
            return self._quarantine(str(exc))

    def save(self, values: Mapping[str, Any]) -> None:
        """Atomically replace the settings file. Raises ``OSError`` if the
        disk refuses; the previous file is left intact in that case."""
        text = encode(values, self.schema)
        self.path.parent.mkdir(parents=True, exist_ok=True)
        fd, tmp_name = tempfile.mkstemp(dir=self.path.parent, prefix=f".{self.path.name}.", suffix=".tmp")
        try:
            with os.fdopen(fd, "w", encoding="utf-8", newline="\n") as tmp:
                tmp.write(text)
                tmp.flush()
                os.fsync(tmp.fileno())
            os.replace(tmp_name, self.path)
        except BaseException:
            Path(tmp_name).unlink(missing_ok=True)
            raise

    def _quarantine(self, reason: str) -> LoadReport:
        try:
            os.replace(self.path, self.backup_path)
            where = f"moved to {self.backup_path.name}"
        except OSError:
            where = "could not move it aside"  # the next save overwrites it
        warning = f"settings file was unreadable ({reason}); {where}, using defaults"
        return LoadReport(self.schema.defaults(), (warning,))
