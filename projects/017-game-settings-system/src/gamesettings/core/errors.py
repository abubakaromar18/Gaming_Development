"""Exception hierarchy for the settings system.

Callers can catch ``SettingsError`` for "anything settings-related went wrong"
or a subclass when they want to react to one specific failure (for example a
menu that shows a ``ValidationError`` inline next to the field).
"""
from __future__ import annotations

from dataclasses import dataclass


class SettingsError(Exception):
    """Base class for every error raised by this package."""


class ValidationError(SettingsError, ValueError):
    """A single value was rejected by its field."""

    def __init__(self, key: str, message: str) -> None:
        super().__init__(f"{key}: {message}" if key else message)
        self.key = key
        self.message = message


class UnknownSettingError(SettingsError, KeyError):
    """A key does not exist in the schema."""

    def __init__(self, key: str, suggestions: tuple[str, ...] = ()) -> None:
        hint = f" (did you mean {', '.join(suggestions)}?)" if suggestions else ""
        super().__init__(f"unknown setting {key!r}{hint}")
        self.key = key
        self.suggestions = suggestions

    def __str__(self) -> str:  # KeyError would otherwise repr() the message
        return str(self.args[0])


@dataclass(frozen=True, slots=True)
class Violation:
    """A cross-field rule failure: which keys conflict and why."""

    keys: tuple[str, ...]
    message: str

    def __str__(self) -> str:
        return self.message


class RuleViolationError(SettingsError):
    """Applying the staged changes would break one or more cross-field rules."""

    def __init__(self, violations: tuple[Violation, ...]) -> None:
        super().__init__("; ".join(str(v) for v in violations))
        self.violations = violations


class StateError(SettingsError):
    """The operation is not allowed in the store's current state."""


class CorruptSettingsError(SettingsError):
    """A settings document could not be parsed at all."""
