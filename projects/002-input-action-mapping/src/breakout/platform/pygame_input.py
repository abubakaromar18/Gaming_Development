"""Translate pygame events into a device-neutral ``RawInput`` snapshot.

This is the only module that knows pygame key codes or joystick objects. It
tracks state from events (rather than polling ``key.get_pressed()``) so keys
are named the same way bindings files name them, gamepads can be hot-plugged,
and everything is cleared when the window loses focus. Without that last step a
key released while another window has focus stays "stuck" down.
"""
from __future__ import annotations

from dataclasses import dataclass, field

import pygame

from ..core.resolver import RawInput

_HAT_X = {-1: "left", 1: "right"}
_HAT_Y = {1: "up", -1: "down"}


@dataclass
class _PadState:
    joystick: pygame.joystick.JoystickType
    buttons: set[int] = field(default_factory=set)
    axes: dict[int, float] = field(default_factory=dict)
    hats: set[tuple[int, str]] = field(default_factory=set)


class PygameInputTracker:
    def __init__(self) -> None:
        self._keys: set[str] = set()
        self._pads: dict[int, _PadState] = {}

    @property
    def pad_names(self) -> list[str]:
        return [p.joystick.get_name() for p in self._pads.values()]

    def handle(self, event: pygame.event.Event) -> None:
        t = event.type
        if t == pygame.KEYDOWN:
            self._keys.add(pygame.key.name(event.key).lower())
        elif t == pygame.KEYUP:
            self._keys.discard(pygame.key.name(event.key).lower())
        elif t == pygame.JOYDEVICEADDED:
            joystick = pygame.joystick.Joystick(event.device_index)
            self._pads[joystick.get_instance_id()] = _PadState(joystick)
        elif t == pygame.JOYDEVICEREMOVED:
            self._pads.pop(event.instance_id, None)
        elif t in (pygame.JOYBUTTONDOWN, pygame.JOYBUTTONUP):
            pad = self._pads.get(event.instance_id)
            if pad is not None:
                (pad.buttons.add if t == pygame.JOYBUTTONDOWN else pad.buttons.discard)(event.button)
        elif t == pygame.JOYAXISMOTION:
            pad = self._pads.get(event.instance_id)
            if pad is not None:
                pad.axes[event.axis] = float(event.value)
        elif t == pygame.JOYHATMOTION:
            pad = self._pads.get(event.instance_id)
            if pad is not None:
                x, y = event.value
                pad.hats = {h for h in pad.hats if h[0] != event.hat}
                if x in _HAT_X:
                    pad.hats.add((event.hat, _HAT_X[x]))
                if y in _HAT_Y:
                    pad.hats.add((event.hat, _HAT_Y[y]))
        elif t == pygame.WINDOWFOCUSLOST:
            self.clear()

    def clear(self) -> None:
        self._keys.clear()
        for pad in self._pads.values():
            pad.buttons.clear()
            pad.axes.clear()
            pad.hats.clear()

    def snapshot(self) -> RawInput:
        """Merge all pads into one virtual pad (single-player game)."""
        buttons: set[int] = set()
        hats: set[tuple[int, str]] = set()
        axes: dict[int, float] = {}
        for pad in self._pads.values():
            buttons |= pad.buttons
            hats |= pad.hats
            for index, value in pad.axes.items():
                if abs(value) > abs(axes.get(index, 0.0)):
                    axes[index] = value
        return RawInput(
            keys=frozenset(self._keys),
            buttons=frozenset(buttons),
            axes=axes,
            hats=frozenset(hats),
        )
