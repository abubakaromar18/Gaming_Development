"""The controls menu as a pure state machine.

The menu is driven by MENU-context actions (so it works the same on keyboard
and gamepad) and, while a rebind prompt is open, by raw input fed to a
``RebindCapture``. The renderer only reads ``rows()``, ``cursor``,
``capture`` and ``message``.
"""
from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from enum import Enum

from .actions import REBINDABLE, Action
from .bindings import ActionMap, describe_source
from .capture import CaptureConfig, CaptureStatus, RebindCapture
from .resolver import ActionState, RawInput

RESET_ITEM = "Reset to defaults"
DONE_ITEM = "Done"


class MenuOutcome(Enum):
    STAY = "stay"
    CLOSE = "close"


@dataclass(frozen=True)
class MenuRow:
    label: str
    bindings: str
    selected: bool


class BindingsMenu:
    def __init__(
        self,
        action_map: ActionMap,
        defaults: Callable[[], ActionMap],
        capture_config: CaptureConfig | None = None,
    ) -> None:
        self.action_map = action_map
        self._defaults = defaults
        self._capture_config = capture_config or CaptureConfig()
        self._items: tuple[Action | str, ...] = (*REBINDABLE, RESET_ITEM, DONE_ITEM)
        self.cursor = 0
        self.capture: RebindCapture | None = None
        self.message = "Select an action to rebind."
        #: Set whenever bindings change, so the shell knows to persist them.
        self.dirty = False

    def rows(self) -> list[MenuRow]:
        rows = []
        for i, item in enumerate(self._items):
            if isinstance(item, Action):
                sources = self.action_map.sources_for(item)
                text = ", ".join(describe_source(s) for s in sources) or "(unbound)"
                rows.append(MenuRow(item.label, text, i == self.cursor))
            else:
                rows.append(MenuRow(item, "", i == self.cursor))
        return rows

    def update(self, actions: ActionState, raw: RawInput, dt: float) -> MenuOutcome:
        if self.capture is not None:
            self._update_capture(raw, dt)
            return MenuOutcome.STAY

        if actions.just_pressed(Action.BACK):
            return MenuOutcome.CLOSE
        if actions.just_pressed(Action.MENU_UP):
            self.cursor = (self.cursor - 1) % len(self._items)
        if actions.just_pressed(Action.MENU_DOWN):
            self.cursor = (self.cursor + 1) % len(self._items)
        if actions.just_pressed(Action.CONFIRM):
            return self._activate(raw)
        return MenuOutcome.STAY

    def _activate(self, raw: RawInput) -> MenuOutcome:
        item = self._items[self.cursor]
        if item == DONE_ITEM:
            return MenuOutcome.CLOSE
        if item == RESET_ITEM:
            self._replace_map(self._defaults())
            self.message = "Controls reset to defaults."
            return MenuOutcome.STAY
        assert isinstance(item, Action)
        self.capture = RebindCapture(
            item,
            start=raw,
            cancel_sources=self.action_map.sources_for(Action.BACK),
            config=self._capture_config,
        )
        self.message = f"Press a key, button, or stick for {item.label}. Back cancels."
        return MenuOutcome.STAY

    def _update_capture(self, raw: RawInput, dt: float) -> None:
        capture = self.capture
        assert capture is not None
        status = capture.feed(raw, dt)
        if status is CaptureStatus.WAITING:
            return
        self.capture = None
        if status is CaptureStatus.CANCELLED:
            self.message = "Rebind cancelled."
        elif status is CaptureStatus.TIMED_OUT:
            self.message = "No input received; binding unchanged."
        else:
            assert capture.captured is not None
            result = self.action_map.rebind(capture.action, capture.captured)
            self.dirty = True
            parts = [f"{capture.action.label} -> {describe_source(result.source)}."]
            if result.stolen_from:
                parts.append("Moved from " + ", ".join(a.label for a in result.stolen_from) + ".")
            if result.now_unbound:
                parts.append("Warning: " + ", ".join(a.label for a in result.now_unbound) + " is now unbound!")
            self.message = " ".join(parts)

    def _replace_map(self, new_map: ActionMap) -> None:
        # Update in place: the resolver holds a reference to the same map.
        self.action_map.assign(new_map)
        self.dirty = True
