"""Pure (no I/O) conversion between settings values and the on-disk JSON.

Format::

    {
      "version": 2,
      "settings": {
        "graphics": {"vsync": false},
        "audio": {"music_volume": 40}
      }
    }

Only values that differ from the default are written. That keeps files tiny
and human-diffable, and means a patch that improves a default reaches every
player who never touched that option.

Loading is *forgiving*: a settings file is user-editable and survives across
game versions, so one bad value must never cost the player the rest of their
setup, let alone crash the game at boot. Each problem becomes a warning in
the ``LoadReport`` and the affected key falls back to its default.
"""
from __future__ import annotations

import json
from collections.abc import Callable, Mapping
from dataclasses import dataclass
from typing import Any

from .errors import CorruptSettingsError, ValidationError
from .schema import Schema

FlatValues = dict[str, Any]
# A migration upgrades a flat ``{dotted.key: raw_value}`` dict by one version.
Migration = Callable[[FlatValues], FlatValues]


@dataclass(frozen=True, slots=True)
class LoadReport:
    values: dict[str, Any]
    warnings: tuple[str, ...] = ()
    migrated_from: int | None = None


def encode(values: Mapping[str, Any], schema: Schema) -> str:
    nested: dict[str, dict[str, Any]] = {}
    for section, keys in schema.sections.items():
        diff = {key.split(".", 1)[1]: values[key] for key in keys if values[key] != schema.default(key)}
        if diff:
            nested[section] = diff
    return json.dumps({"version": schema.version, "settings": nested}, indent=2, ensure_ascii=False) + "\n"


def decode(text: str, schema: Schema, migrations: Mapping[int, Migration] | None = None) -> LoadReport:
    """Parse a settings document. Raises ``CorruptSettingsError`` only when
    the document is unusable as a whole (not JSON, wrong top-level shape);
    everything finer-grained is repaired and reported."""
    try:
        document = json.loads(text)
    except json.JSONDecodeError as exc:
        raise CorruptSettingsError(f"not valid JSON: {exc.msg} at line {exc.lineno}") from None
    if not isinstance(document, dict):
        raise CorruptSettingsError("top level must be a JSON object")

    warnings: list[str] = []
    version = document.get("version", 1)  # files from before versioning existed
    if isinstance(version, bool) or not isinstance(version, int) or version < 1:
        raise CorruptSettingsError(f"invalid version {version!r}")

    raw = _flatten(document.get("settings", {}), warnings)
    migrated_from = None
    if version < schema.version:
        raw, ok = _migrate(raw, version, schema.version, migrations or {}, warnings)
        if not ok:
            return LoadReport(schema.defaults(), tuple(warnings))
        migrated_from = version
    elif version > schema.version:
        warnings.append(f"file is from a newer version ({version} > {schema.version}); "
                        "unknown settings will be dropped on save")

    values = schema.defaults()
    for key, value in raw.items():
        if key not in schema:
            warnings.append(f"ignored unknown setting {key}")
            continue
        field = schema.field(key)
        try:
            values[key] = field.coerce(value)
        except ValidationError as exc:
            try:
                values[key] = field.coerce(value, clamp=True)
                warnings.append(f"{exc}; clamped to {field.format(values[key])}")
            except ValidationError:
                warnings.append(f"{exc}; using default {field.format(field.default)}")

    _repair_rules(values, schema, warnings)
    return LoadReport(values, tuple(warnings), migrated_from)


def _flatten(settings: Any, warnings: list[str]) -> FlatValues:
    if not isinstance(settings, dict):
        warnings.append("'settings' is not an object; using defaults")
        return {}
    flat: FlatValues = {}
    for section, entries in settings.items():
        if not isinstance(entries, dict):
            warnings.append(f"section {section!r} is not an object; ignored")
            continue
        for name, value in entries.items():
            flat[f"{section}.{name}"] = value
    return flat


def _migrate(raw: FlatValues, start: int, target: int, migrations: Mapping[int, Migration],
             warnings: list[str]) -> tuple[FlatValues, bool]:
    """Run migrations start→start+1→…→target. Each step is a small function
    that only knows about its own version bump, so they compose."""
    for version in range(start, target):
        migration = migrations.get(version)
        if migration is None:
            warnings.append(f"no migration from version {version}; settings reset to defaults")
            return {}, False
        raw = migration(dict(raw))
    return raw, True


def _repair_rules(values: dict[str, Any], schema: Schema, warnings: list[str]) -> None:
    """Reset the player-chosen side of any cross-field conflict to defaults.

    Each pass resets at least one non-default value, so the loop terminates.
    A rule that names only default-valued keys must depend on something it
    didn't name; the only safe answer then is the full (conflict-free, as the
    schema guarantees) default set.
    """
    while violations := schema.check(values):
        culprits = {k for v in violations for k in v.keys if values[k] != schema.default(k)}
        for violation in violations:
            warnings.append(f"{violation}; reset to defaults")
        if not culprits:
            values.update(schema.defaults())
            return
        for key in culprits:
            values[key] = schema.default(key)
