"""Turn a raw device snapshot into per-action values and edges.

Each frame the platform layer hands over a ``RawInput``: plain sets and numbers
with no library types. The resolver looks every action's sources up in that
snapshot and produces an ``ActionState`` with:

* ``value``: an analog strength in [0, 1] (keys and buttons are 0 or 1; stick
  halves are deadzoned and rescaled), the max over all bound sources;
* ``held``: a digital reading of that value, with hysteresis;
* ``just_pressed`` / ``just_released``: edges relative to the previous frame.

Edges are what menus and one-shot commands (launch, pause) need, and computing
them here means no game code has to remember last frame's input.
"""
from __future__ import annotations

from collections.abc import Mapping
from dataclasses import dataclass, field

from .actions import Action
from .bindings import ActionMap, Key, PadAxis, PadButton, PadHat, Source


@dataclass(frozen=True)
class RawInput:
    """Everything the devices report this frame, as plain data."""

    keys: frozenset[str] = frozenset()
    buttons: frozenset[int] = frozenset()
    axes: Mapping[int, float] = field(default_factory=dict)
    #: Active hat directions as (hat index, "left"|"right"|"up"|"down").
    hats: frozenset[tuple[int, str]] = frozenset()


NO_INPUT = RawInput()


@dataclass(frozen=True)
class ResolverConfig:
    #: Stick travel below this counts as zero (stops drift on worn sticks).
    deadzone: float = 0.2
    #: Value at which an analog source starts counting as "held".
    press_threshold: float = 0.5
    #: Value below which a held source is released. Lower than the press
    #: threshold so a stick hovering near the edge doesn't chatter.
    release_threshold: float = 0.35

    def __post_init__(self) -> None:
        if not 0.0 <= self.deadzone < 1.0:
            raise ValueError("deadzone must be in [0, 1)")
        if not 0.0 < self.release_threshold <= self.press_threshold <= 1.0:
            raise ValueError("need 0 < release_threshold <= press_threshold <= 1")


@dataclass(frozen=True)
class ActionState:
    values: Mapping[Action, float]
    held_set: frozenset[Action]
    pressed_set: frozenset[Action]
    released_set: frozenset[Action]

    def value(self, action: Action) -> float:
        return self.values.get(action, 0.0)

    def held(self, action: Action) -> bool:
        return action in self.held_set

    def just_pressed(self, action: Action) -> bool:
        return action in self.pressed_set

    def just_released(self, action: Action) -> bool:
        return action in self.released_set


def axis_value(raw: float, direction: int, deadzone: float) -> float:
    """Value of one half of an axis after a rescaled deadzone.

    Rescaling (rather than just clipping) keeps the response continuous: just
    past the deadzone the value starts at 0, not at ``deadzone``.
    """
    v = raw * direction
    if v <= deadzone:
        return 0.0
    return min(1.0, (v - deadzone) / (1.0 - deadzone))


def source_value(source: Source, raw: RawInput, deadzone: float) -> float:
    match source:
        case Key(name):
            return 1.0 if name in raw.keys else 0.0
        case PadButton(index):
            return 1.0 if index in raw.buttons else 0.0
        case PadAxis(index, direction):
            return axis_value(raw.axes.get(index, 0.0), direction, deadzone)
        case PadHat(index, direction):
            return 1.0 if (index, direction) in raw.hats else 0.0
    raise TypeError(f"not a source: {source!r}")


class InputResolver:
    """Stateful only in what it must be: last frame's held set, for edges."""

    def __init__(self, action_map: ActionMap, config: ResolverConfig | None = None) -> None:
        self.action_map = action_map
        self.config = config or ResolverConfig()
        self._held: frozenset[Action] = frozenset()

    def resolve(self, raw: RawInput) -> ActionState:
        cfg = self.config
        values: dict[Action, float] = {}
        held: set[Action] = set()
        for action in Action:
            v = max((source_value(s, raw, cfg.deadzone) for s in self.action_map.sources_for(action)), default=0.0)
            values[action] = v
            threshold = cfg.release_threshold if action in self._held else cfg.press_threshold
            if v >= threshold:
                held.add(action)
        frozen = frozenset(held)
        state = ActionState(
            values=values,
            held_set=frozen,
            pressed_set=frozen - self._held,
            released_set=self._held - frozen,
        )
        self._held = frozen
        return state
