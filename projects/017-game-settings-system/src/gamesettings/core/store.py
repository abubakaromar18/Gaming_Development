"""The settings store: live values, staged edits, and the keep-or-revert flow.

Game options menus don't write straight into the live config. The player
nudges sliders (``stage``), sees the result previewed, then hits Apply or
Back. This store models exactly that:

    stage / adjust ──► pending ──apply()──► live ──► listeners notified
                         │                   │
                    discard()        risky change? ──► awaiting confirmation
                                                       │ confirm()  keep
                                                       │ revert()   restore
                                                       └ update(dt) times out → restore

The countdown is driven by ``update(dt)`` rather than a wall clock, so the
caller owns time and tests can fast-forward it.

``durable_values`` is what should be written to disk: while a resolution
change is unconfirmed it still returns the *previous* values, so a crash or
power cut mid-countdown boots into the last known-good display mode.
"""
from __future__ import annotations

from collections.abc import Callable, Iterable, Mapping
from dataclasses import dataclass
from types import MappingProxyType
from typing import Any, NamedTuple

from .errors import RuleViolationError, StateError, ValidationError
from .schema import Schema


class Change(NamedTuple):
    key: str
    old: Any
    new: Any


Listener = Callable[[tuple[Change, ...]], None]


@dataclass(frozen=True, slots=True)
class ApplyResult:
    changes: tuple[Change, ...]
    restart_required: tuple[str, ...]
    awaiting_confirmation: bool


class SettingsStore:
    def __init__(self, schema: Schema, values: Mapping[str, Any] | None = None, *,
                 confirm_timeout: float = 15.0) -> None:
        if confirm_timeout <= 0:
            raise ValueError("confirm_timeout must be positive")
        self.schema = schema
        self.confirm_timeout = confirm_timeout
        live = schema.defaults()
        for key, value in (values or {}).items():
            live[key] = schema.field(key).coerce(value)
        violations = schema.check(live)
        if violations:
            raise RuleViolationError(violations)
        self._live = live
        self._pending: dict[str, Any] = {}
        self._listeners: list[Listener] = []
        self._fallback: dict[str, Any] | None = None
        self._remaining = 0.0

    # -- reading -------------------------------------------------------------

    @property
    def values(self) -> Mapping[str, Any]:
        """Live values the game is running with right now (read-only view)."""
        return MappingProxyType(self._live)

    @property
    def pending(self) -> Mapping[str, Any]:
        return MappingProxyType(self._pending)

    def get(self, key: str) -> Any:
        self.schema.field(key)
        return self._live[key]

    def preview(self, key: str) -> Any:
        """The value the menu should show: staged if edited, else live."""
        self.schema.field(key)
        return self._pending.get(key, self._live[key])

    def preview_values(self) -> dict[str, Any]:
        return {**self._live, **self._pending}

    @property
    def durable_values(self) -> Mapping[str, Any]:
        return MappingProxyType(self._fallback if self._fallback is not None else self._live)

    @property
    def awaiting_confirmation(self) -> bool:
        return self._fallback is not None

    @property
    def seconds_remaining(self) -> float:
        return max(self._remaining, 0.0) if self.awaiting_confirmation else 0.0

    # -- staging -------------------------------------------------------------

    def stage(self, key: str, value: Any) -> Any:
        """Stage a typed value (from code or a JSON source)."""
        return self._stage(key, self.schema.field(key).coerce(value))

    def stage_text(self, key: str, text: str) -> Any:
        """Stage a value the player typed."""
        return self._stage(key, self.schema.field(key).parse(text))

    def adjust(self, key: str, direction: int) -> Any:
        """Menu left/right arrow."""
        field = self.schema.field(key)
        if not field.steppable:
            raise ValidationError(key, "this setting is edited by typing a value")
        return self._stage(key, field.step(self.preview(key), direction))

    def stage_many(self, values: Mapping[str, Any]) -> None:
        coerced = {key: self.schema.field(key).coerce(v) for key, v in values.items()}
        for key, value in coerced.items():  # all-or-nothing: validate first
            self._stage(key, value)

    def stage_defaults(self, keys: Iterable[str] | None = None) -> None:
        for key in self.schema.keys if keys is None else keys:
            self._stage(key, self.schema.default(key))

    def discard(self) -> None:
        self._pending.clear()

    def _stage(self, key: str, value: Any) -> Any:
        self._require_settled()
        if value == self._live[key]:
            self._pending.pop(key, None)  # editing back to live is "no change"
        else:
            self._pending[key] = value
        return value

    # -- committing ----------------------------------------------------------

    def apply(self, *, require_confirmation: bool = True) -> ApplyResult:
        """Commit staged edits. Cross-field rules are checked against the
        merged result; on failure nothing changes and edits stay staged so
        the player can fix them."""
        self._require_settled()
        merged = self.preview_values()
        violations = self.schema.check(merged)
        if violations:
            raise RuleViolationError(violations)
        changes = tuple(Change(k, self._live[k], v) for k, v in self._pending.items())
        risky = any(self.schema.field(c.key).needs_confirmation for c in changes)
        if risky and require_confirmation:
            self._fallback = dict(self._live)
            self._remaining = self.confirm_timeout
        self._live = merged
        self._pending.clear()
        self._notify(changes)
        return ApplyResult(
            changes=changes,
            restart_required=tuple(c.key for c in changes if self.schema.field(c.key).requires_restart),
            awaiting_confirmation=self.awaiting_confirmation,
        )

    def confirm(self) -> None:
        if self._fallback is None:
            raise StateError("nothing is waiting for confirmation")
        self._fallback = None

    def revert(self) -> tuple[Change, ...]:
        if self._fallback is None:
            raise StateError("nothing is waiting for confirmation")
        previous, self._fallback = self._fallback, None
        changes = tuple(Change(k, v, previous[k]) for k, v in self._live.items() if previous[k] != v)
        self._live = previous
        self._notify(changes)
        return changes

    def update(self, dt: float) -> bool:
        """Advance the confirmation countdown. Returns True if it expired and
        the risky change was reverted this call."""
        if dt < 0:
            raise ValueError("dt must be non-negative")
        if self._fallback is None:
            return False
        self._remaining -= dt
        if self._remaining <= 0:
            self.revert()
            return True
        return False

    def _require_settled(self) -> None:
        if self._fallback is not None:
            raise StateError("confirm or revert the pending display change first")

    # -- observing -----------------------------------------------------------

    def subscribe(self, listener: Listener) -> Callable[[], None]:
        """Register for change batches. Returns an unsubscribe function."""
        self._listeners.append(listener)

        def unsubscribe() -> None:
            if listener in self._listeners:
                self._listeners.remove(listener)

        return unsubscribe

    def _notify(self, changes: tuple[Change, ...]) -> None:
        if changes:
            for listener in tuple(self._listeners):  # tolerate unsubscribe mid-loop
                listener(changes)
