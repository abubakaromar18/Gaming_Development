"""Physical input sources and the action map that binds them to actions.

A *source* names one physical control in a device-neutral way: a keyboard key,
a gamepad button, one direction of a gamepad axis, or one direction of a D-pad
hat. Sources are small frozen values, so they hash, compare, and serialize
cleanly. The platform layer is the only code that knows how to read them.
"""
from __future__ import annotations

from collections.abc import Iterable, Mapping
from dataclasses import dataclass
from enum import Enum
from typing import Union

from .actions import Action, Context


class DeviceClass(Enum):
    KEYBOARD = "keyboard"
    GAMEPAD = "gamepad"


HAT_DIRECTIONS = ("left", "right", "up", "down")


@dataclass(frozen=True)
class Key:
    """A keyboard key, by its lowercase name (``"space"``, ``"left"``, ``"a"``)."""

    name: str


@dataclass(frozen=True)
class PadButton:
    index: int


@dataclass(frozen=True)
class PadAxis:
    """One half of an analog axis: ``direction`` is +1 or -1."""

    index: int
    direction: int

    def __post_init__(self) -> None:
        if self.direction not in (-1, 1):
            raise ValueError(f"axis direction must be +1 or -1, got {self.direction}")


@dataclass(frozen=True)
class PadHat:
    """One direction of a D-pad hat switch."""

    index: int
    direction: str

    def __post_init__(self) -> None:
        if self.direction not in HAT_DIRECTIONS:
            raise ValueError(f"hat direction must be one of {HAT_DIRECTIONS}, got {self.direction!r}")


Source = Union[Key, PadButton, PadAxis, PadHat]


def device_class(source: Source) -> DeviceClass:
    return DeviceClass.KEYBOARD if isinstance(source, Key) else DeviceClass.GAMEPAD


# --- Serialization ---------------------------------------------------------
#
# Sources are stored as short strings ("key:space", "pad:axis:0:-") so a
# bindings file stays readable and hand-editable.


def format_source(source: Source) -> str:
    match source:
        case Key(name):
            return f"key:{name}"
        case PadButton(index):
            return f"pad:button:{index}"
        case PadAxis(index, direction):
            return f"pad:axis:{index}:{'+' if direction > 0 else '-'}"
        case PadHat(index, direction):
            return f"pad:hat:{index}:{direction}"
    raise TypeError(f"not a source: {source!r}")


def parse_source(text: str) -> Source:
    parts = text.strip().split(":")
    try:
        if parts[0] == "key" and len(parts) == 2 and parts[1]:
            return Key(parts[1].lower())
        if parts[0] == "pad" and len(parts) >= 3:
            kind, index = parts[1], int(parts[2])
            if index < 0:
                raise ValueError("negative index")
            if kind == "button" and len(parts) == 3:
                return PadButton(index)
            if kind == "axis" and len(parts) == 4 and parts[3] in ("+", "-"):
                return PadAxis(index, 1 if parts[3] == "+" else -1)
            if kind == "hat" and len(parts) == 4:
                return PadHat(index, parts[3])
    except ValueError as exc:
        raise ValueError(f"invalid input source {text!r}: {exc}") from exc
    raise ValueError(f"invalid input source {text!r}")


def describe_source(source: Source) -> str:
    """Short human label for HUDs and menus."""
    match source:
        case Key(name):
            return name.title() if len(name) > 1 else name.upper()
        case PadButton(index):
            return f"Btn {index}"
        case PadAxis(index, direction):
            return f"Stick {index}{'+' if direction > 0 else '-'}"
        case PadHat(index, direction):
            return f"D-pad {direction}" if index == 0 else f"Hat {index} {direction}"
    raise TypeError(f"not a source: {source!r}")


# --- The action map ---------------------------------------------------------


@dataclass(frozen=True)
class RebindResult:
    """What a rebind changed, so the UI can tell the player."""

    action: Action
    source: Source
    replaced: tuple[Source, ...]
    #: Actions (same context) that lost this source because it moved here.
    stolen_from: tuple[Action, ...]
    #: Actions left with no bindings at all after the move.
    now_unbound: tuple[Action, ...]


class ActionMap:
    """Many-to-many mapping between actions and sources.

    Invariant: within one context, a source drives at most one action. Across
    contexts sharing is fine, because only one context is read at a time.
    """

    def __init__(self, bindings: Mapping[Action, Iterable[Source]]) -> None:
        self._bindings: dict[Action, list[Source]] = {a: [] for a in Action}
        for action, sources in bindings.items():
            for source in sources:
                self._add(action, source)

    def _add(self, action: Action, source: Source) -> None:
        owner = self.action_for(source, action.context)
        if owner is not None and owner is not action:
            raise ValueError(
                f"{format_source(source)} is bound to both {owner.value} and {action.value} "
                f"in the {action.context.value} context"
            )
        if source not in self._bindings[action]:
            self._bindings[action].append(source)

    def sources_for(self, action: Action) -> tuple[Source, ...]:
        return tuple(self._bindings[action])

    def action_for(self, source: Source, context: Context) -> Action | None:
        for action, sources in self._bindings.items():
            if action.context is context and source in sources:
                return action
        return None

    def rebind(self, action: Action, source: Source) -> RebindResult:
        """Make ``source`` the binding for ``action`` on its device class.

        The action's other bindings of the same device class are replaced, so
        rebinding Move Left on the keyboard leaves its gamepad bindings alone.
        If another action in the same context already used ``source``, the
        source moves (rather than the rebind being refused): this is what
        players expect from most PC games' controls menus.
        """
        cls = device_class(source)
        stolen: list[Action] = []
        for other in Action:
            if other is not action and other.context is action.context and source in self._bindings[other]:
                self._bindings[other].remove(source)
                stolen.append(other)
        replaced = tuple(s for s in self._bindings[action] if device_class(s) is cls and s != source)
        kept = [s for s in self._bindings[action] if device_class(s) is not cls]
        self._bindings[action] = kept + [source]
        return RebindResult(
            action=action,
            source=source,
            replaced=replaced,
            stolen_from=tuple(stolen),
            now_unbound=tuple(a for a in stolen if not self._bindings[a]),
        )

    def assign(self, other: ActionMap) -> None:
        """Replace every binding with ``other``'s, keeping this object's identity."""
        self._bindings = {a: list(other.sources_for(a)) for a in Action}

    def __eq__(self, other: object) -> bool:
        return isinstance(other, ActionMap) and self.to_dict() == other.to_dict()

    def to_dict(self) -> dict[str, list[str]]:
        return {a.value: [format_source(s) for s in self._bindings[a]] for a in Action}

    @classmethod
    def from_dict(cls, data: Mapping[str, object], defaults: ActionMap | None = None) -> ActionMap:
        """Build a map from saved data.

        Actions missing from ``data`` fall back to ``defaults``, so a bindings
        file written by an older build keeps working after new actions ship.
        Unknown action names or malformed sources raise ``ValueError``.
        """
        by_name = {a.value: a for a in Action}
        merged: dict[Action, list[Source]] = {}
        if defaults is not None:
            merged = {a: list(defaults.sources_for(a)) for a in Action}
        for name, raw_sources in data.items():
            if name not in by_name:
                raise ValueError(f"unknown action {name!r}")
            if not isinstance(raw_sources, list) or not all(isinstance(s, str) for s in raw_sources):
                raise ValueError(f"bindings for {name!r} must be a list of strings")
            merged[by_name[name]] = [parse_source(s) for s in raw_sources]
        return cls(merged)


def default_bindings() -> ActionMap:
    """Keyboard + SDL-style gamepad defaults (button 0 = A/Cross on most pads)."""
    return ActionMap(
        {
            Action.MOVE_LEFT: [Key("left"), Key("a"), PadAxis(0, -1), PadHat(0, "left")],
            Action.MOVE_RIGHT: [Key("right"), Key("d"), PadAxis(0, 1), PadHat(0, "right")],
            Action.LAUNCH: [Key("space"), PadButton(0)],
            Action.PAUSE: [Key("p"), PadButton(7)],
            Action.OPEN_BINDINGS: [Key("f1"), PadButton(6)],
            Action.QUIT: [Key("escape")],
            Action.MENU_UP: [Key("up"), Key("w"), PadAxis(1, -1), PadHat(0, "up")],
            Action.MENU_DOWN: [Key("down"), Key("s"), PadAxis(1, 1), PadHat(0, "down")],
            Action.CONFIRM: [Key("return"), Key("space"), PadButton(0)],
            Action.BACK: [Key("escape"), PadButton(1)],
        }
    )
