"""Capturing the next physical input for a "Press a key for Launch..." prompt.

The tricky part of a rebind prompt is ignoring input that was *already* active
when the prompt opened. The player confirmed the menu item with Enter or the A
button, and that press is usually still held on the first capture frame. A
naive capture would bind CONFIRM's own key. So every source active at the start
is blocked until it is released, and only a fresh activation is captured.

Sticks need a higher threshold than gameplay: a capture should be a deliberate
push, not resting drift.
"""
from __future__ import annotations

from collections.abc import Iterable, Iterator
from dataclasses import dataclass
from enum import Enum

from .actions import Action
from .bindings import Key, PadAxis, PadButton, PadHat, Source
from .resolver import RawInput, source_value


class CaptureStatus(Enum):
    WAITING = "waiting"
    CAPTURED = "captured"
    CANCELLED = "cancelled"
    TIMED_OUT = "timed_out"


@dataclass(frozen=True)
class CaptureConfig:
    axis_threshold: float = 0.6
    timeout_seconds: float = 5.0


def active_sources(raw: RawInput, axis_threshold: float) -> Iterator[Source]:
    """All sources active in ``raw``, in a stable order (keys, buttons, hats, axes)."""
    for name in sorted(raw.keys):
        yield Key(name)
    for index in sorted(raw.buttons):
        yield PadButton(index)
    for index, direction in sorted(raw.hats):
        yield PadHat(index, direction)
    for index in sorted(raw.axes):
        for direction in (-1, 1):
            axis = PadAxis(index, direction)
            if source_value(axis, raw, 0.0) >= axis_threshold:
                yield axis


class RebindCapture:
    """A small state machine fed one ``RawInput`` (and ``dt``) per frame."""

    def __init__(
        self,
        action: Action,
        start: RawInput,
        cancel_sources: Iterable[Source],
        config: CaptureConfig | None = None,
    ) -> None:
        self.action = action
        self.config = config or CaptureConfig()
        self._cancel = frozenset(cancel_sources)
        self._blocked = set(active_sources(start, self.config.axis_threshold))
        self._remaining = self.config.timeout_seconds
        self.status = CaptureStatus.WAITING
        self.captured: Source | None = None

    @property
    def seconds_left(self) -> float:
        return max(0.0, self._remaining)

    def feed(self, raw: RawInput, dt: float) -> CaptureStatus:
        if self.status is not CaptureStatus.WAITING:
            return self.status
        active = list(active_sources(raw, self.config.axis_threshold))
        # Released sources become capturable again.
        self._blocked.intersection_update(active)
        fresh = [s for s in active if s not in self._blocked]
        if fresh:
            source = fresh[0]
            if source in self._cancel:
                self.status = CaptureStatus.CANCELLED
            else:
                self.status = CaptureStatus.CAPTURED
                self.captured = source
            return self.status
        self._remaining -= dt
        if self._remaining <= 0.0:
            self.status = CaptureStatus.TIMED_OUT
        return self.status
