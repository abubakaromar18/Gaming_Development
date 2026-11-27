"""Typed setting fields.

A field knows four things about one setting:

* ``coerce``  - turn a JSON-ish Python value into a canonical value or raise
  ``ValidationError``. With ``clamp=True`` numeric fields pull out-of-range
  values back into range instead of rejecting them; the loader uses this so a
  file written when FOV allowed 130 still loads as 120 rather than reverting
  to the default.
* ``parse``   - the same, but from text a player typed or a CLI argument.
* ``step``    - the next value for a menu's left/right arrows (sliders clamp,
  toggles flip, choice lists wrap around).
* ``format``  - how to show the value.

Values are always plain JSON types (bool, int, float, str) so saving them
never needs custom encoders.
"""
from __future__ import annotations

import math
import re
from abc import ABC, abstractmethod
from typing import Any, Generic, TypeVar

from .errors import ValidationError

T = TypeVar("T")


class Field(ABC, Generic[T]):
    #: Whether ``step`` is meaningful (menus hide the arrows otherwise).
    steppable = True

    def __init__(
        self,
        default: T,
        label: str,
        *,
        description: str = "",
        requires_restart: bool = False,
        needs_confirmation: bool = False,
    ) -> None:
        self.label = label
        self.description = description
        # Some engine settings (e.g. texture pool size) only take effect after
        # a restart; the store reports these so the UI can say so.
        self.requires_restart = requires_restart
        # "Risky" settings like resolution can leave the player staring at a
        # black screen, so applying them starts a keep-or-revert countdown.
        self.needs_confirmation = needs_confirmation
        self.key = ""  # assigned by Schema
        self._raw_default = default

    @property
    def default(self) -> T:
        return self._raw_default

    def bind(self, key: str) -> None:
        """Attach the field to its dotted key and validate its own default."""
        self.key = key
        self._raw_default = self.coerce(self._raw_default)

    def _fail(self, message: str) -> ValidationError:
        return ValidationError(self.key, message)

    @abstractmethod
    def coerce(self, value: Any, *, clamp: bool = False) -> T: ...

    def parse(self, text: str) -> T:
        return self.coerce(text.strip())

    def step(self, value: T, direction: int) -> T:
        raise TypeError(f"{self.key} cannot be stepped")

    def format(self, value: T) -> str:
        return str(value)

    @abstractmethod
    def describe_type(self) -> str: ...


class BoolField(Field[bool]):
    _TRUE = {"true", "on", "yes", "1"}
    _FALSE = {"false", "off", "no", "0"}

    def coerce(self, value: Any, *, clamp: bool = False) -> bool:
        if isinstance(value, bool):
            return value
        if isinstance(value, str):
            word = value.strip().lower()
            if word in self._TRUE:
                return True
            if word in self._FALSE:
                return False
        raise self._fail(f"expected on/off, got {value!r}")

    def step(self, value: bool, direction: int) -> bool:
        return not value

    def format(self, value: bool) -> str:
        return "on" if value else "off"

    def describe_type(self) -> str:
        return "on/off"


class _NumberField(Field[T]):
    def __init__(self, default: T, label: str, *, minimum: T, maximum: T, step: T, unit: str = "", **kw: Any) -> None:
        if minimum > maximum:
            raise ValueError(f"{label}: minimum {minimum} > maximum {maximum}")
        if step <= 0:
            raise ValueError(f"{label}: step must be positive")
        self.minimum = minimum
        self.maximum = maximum
        self.step_size = step
        self.unit = unit
        super().__init__(default, label, **kw)

    @abstractmethod
    def _convert(self, value: Any) -> T: ...

    def coerce(self, value: Any, *, clamp: bool = False) -> T:
        if isinstance(value, str):
            text = value.strip().removesuffix(self.unit).strip() if self.unit else value.strip()
            try:
                value = float(text)
            except ValueError:
                raise self._fail(f"expected a number, got {value!r}") from None
        # bool is a subclass of int in Python; "true" is not a volume.
        if isinstance(value, bool) or not isinstance(value, (int, float)):
            raise self._fail(f"expected a number, got {value!r}")
        if not math.isfinite(value):
            raise self._fail(f"expected a finite number, got {value!r}")
        number = self._convert(value)
        if number < self.minimum or number > self.maximum:
            if not clamp:
                span = f"{self.format(self.minimum)}..{self.format(self.maximum)}"
                raise self._fail(f"{self.format(number)} is outside {span}")
            number = min(max(number, self.minimum), self.maximum)
        return number

    def step(self, value: T, direction: int) -> T:
        # Snap to the step grid anchored at the minimum, so repeated presses
        # land on tidy values (0.75, 0.80 ...) instead of accumulating error.
        steps = round((value - self.minimum) / self.step_size) + (1 if direction > 0 else -1)
        return self.coerce(self.minimum + steps * self.step_size, clamp=True)

    def format(self, value: T) -> str:
        return f"{value:g}{self.unit}"

    def describe_type(self) -> str:
        return f"{self.format(self.minimum)}..{self.format(self.maximum)}"


class IntField(_NumberField[int]):
    def _convert(self, value: Any) -> int:
        if isinstance(value, float) and not value.is_integer():
            raise self._fail(f"expected a whole number, got {value!r}")
        return int(value)


class FloatField(_NumberField[float]):
    def _convert(self, value: Any) -> float:
        # Round away binary noise (0.1 + 0.2) so saved files stay readable
        # and equality against defaults behaves.
        return round(float(value), 6)


class ChoiceField(Field[str]):
    def __init__(self, default: str, label: str, *, choices: tuple[str, ...], **kw: Any) -> None:
        if not choices:
            raise ValueError(f"{label}: choices must not be empty")
        if len({c.lower() for c in choices}) != len(choices):
            raise ValueError(f"{label}: choices must be unique ignoring case")
        self.choices = tuple(choices)
        super().__init__(default, label, **kw)

    def coerce(self, value: Any, *, clamp: bool = False) -> str:
        if isinstance(value, str):
            for choice in self.choices:
                if choice.lower() == value.strip().lower():
                    return choice
        raise self._fail(f"expected one of {', '.join(self.choices)}, got {value!r}")

    def step(self, value: str, direction: int) -> str:
        i = self.choices.index(value)
        return self.choices[(i + (1 if direction > 0 else -1)) % len(self.choices)]

    def describe_type(self) -> str:
        return " | ".join(self.choices)


class ResolutionField(Field[str]):
    """A ``WIDTHxHEIGHT`` string. Any size in range is allowed (players with
    odd monitors exist); stepping walks a list of common modes."""

    COMMON = ("1280x720", "1366x768", "1600x900", "1920x1080", "2560x1440", "3440x1440", "3840x2160")
    _PATTERN = re.compile(r"^\s*(\d{2,5})\s*[xX×]\s*(\d{2,5})\s*$")

    def __init__(self, default: str, label: str, *, minimum: tuple[int, int] = (640, 360),
                 maximum: tuple[int, int] = (7680, 4320), **kw: Any) -> None:
        self.minimum = minimum
        self.maximum = maximum
        super().__init__(default, label, **kw)

    @staticmethod
    def size(value: str) -> tuple[int, int]:
        width, height = value.split("x")
        return int(width), int(height)

    def coerce(self, value: Any, *, clamp: bool = False) -> str:
        match = self._PATTERN.match(value) if isinstance(value, str) else None
        if not match:
            raise self._fail(f"expected WIDTHxHEIGHT, got {value!r}")
        width, height = int(match[1]), int(match[2])
        (min_w, min_h), (max_w, max_h) = self.minimum, self.maximum
        if not (min_w <= width <= max_w and min_h <= height <= max_h):
            raise self._fail(f"{width}x{height} is outside {min_w}x{min_h}..{max_w}x{max_h}")
        return f"{width}x{height}"

    def step(self, value: str, direction: int) -> str:
        pixels = math.prod(self.size(value))
        modes = sorted(self.COMMON, key=lambda m: math.prod(self.size(m)))
        if direction > 0:
            bigger = [m for m in modes if math.prod(self.size(m)) > pixels]
            return bigger[0] if bigger else value
        smaller = [m for m in modes if math.prod(self.size(m)) < pixels]
        return smaller[-1] if smaller else value

    def describe_type(self) -> str:
        return "WIDTHxHEIGHT"


# Key names an input layer would map to scancodes. Kept engine-agnostic.
KEY_NAMES: frozenset[str] = frozenset(
    [chr(c) for c in range(ord("A"), ord("Z") + 1)]
    + [str(d) for d in range(10)]
    + [f"F{n}" for n in range(1, 13)]
    + ["SPACE", "ESCAPE", "ENTER", "TAB", "BACKSPACE", "LSHIFT", "RSHIFT", "LCTRL", "RCTRL",
       "LALT", "RALT", "UP", "DOWN", "LEFT", "RIGHT", "MOUSE1", "MOUSE2", "MOUSE3", "MOUSE4", "MOUSE5"]
)
KEY_ALIASES = {"ESC": "ESCAPE", "RETURN": "ENTER", "SHIFT": "LSHIFT", "CTRL": "LCTRL", "ALT": "LALT",
               "LMB": "MOUSE1", "RMB": "MOUSE2", "MMB": "MOUSE3"}


class KeyField(Field[str]):
    """A key binding. Normalized to upper case; common aliases accepted."""

    steppable = False

    def coerce(self, value: Any, *, clamp: bool = False) -> str:
        if isinstance(value, str):
            name = value.strip().upper()
            name = KEY_ALIASES.get(name, name)
            if name in KEY_NAMES:
                return name
        raise self._fail(f"unknown key {value!r}")

    def describe_type(self) -> str:
        return "key name"
