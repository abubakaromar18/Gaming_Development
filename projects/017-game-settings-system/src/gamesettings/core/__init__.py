"""Pure settings logic: fields, schema, store, codec, and persistence.
No terminal, rendering, or audio code lives here."""
from .codec import LoadReport, decode, encode
from .errors import (
    CorruptSettingsError,
    RuleViolationError,
    SettingsError,
    StateError,
    UnknownSettingError,
    ValidationError,
    Violation,
)
from .fields import BoolField, ChoiceField, Field, FloatField, IntField, KeyField, ResolutionField
from .game import MIGRATIONS, PRESETS, bindings, build_schema, detect_preset, effective_volume, render_resolution
from .schema import Schema, unique_values
from .storage import SettingsFile
from .store import ApplyResult, Change, SettingsStore

__all__ = [
    "ApplyResult", "BoolField", "Change", "ChoiceField", "CorruptSettingsError", "Field", "FloatField",
    "IntField", "KeyField", "LoadReport", "MIGRATIONS", "PRESETS", "ResolutionField", "RuleViolationError",
    "Schema", "SettingsError", "SettingsFile", "SettingsStore", "StateError", "UnknownSettingError",
    "ValidationError", "Violation", "bindings", "build_schema", "decode", "detect_preset",
    "effective_volume", "encode", "render_resolution", "unique_values",
]
