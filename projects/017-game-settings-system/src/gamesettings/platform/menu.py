"""A text-mode options menu over the settings store.

It plays the role a game's pause-menu UI would: tabs per section, a cursor,
left/right to adjust, Apply/Discard, presets, and the "Keep these display
settings?" countdown. Rendering returns a string and input arrives as
command lines, so the whole menu is driven by tests with scripted input and
a fake clock.
"""
from __future__ import annotations

from collections.abc import Callable, Mapping
from typing import Any

from ..core import PRESETS, SettingsError, SettingsStore, detect_preset, render_resolution

HELP = ("tabs: 1-3 · move: j/k · adjust: h/l or -/+ · type a value: = VALUE · "
        "a apply · d discard · r reset tab · p PRESET · q quit")

Persist = Callable[[Mapping[str, Any]], None]


class SettingsMenu:
    def __init__(self, store: SettingsStore, persist: Persist) -> None:
        self.store = store
        self.persist = persist
        self.sections = tuple(store.schema.sections)
        self.section = 0
        self.cursor = 0
        self.message = ""
        self.running = True
        self._saved = dict(store.durable_values)

    @property
    def keys(self) -> tuple[str, ...]:
        return self.store.schema.sections[self.sections[self.section]]

    @property
    def selected(self) -> str:
        return self.keys[self.cursor]

    # -- time ----------------------------------------------------------------

    def tick(self, dt: float) -> bool:
        """Advance time; True if the confirmation countdown just expired."""
        if self.store.update(dt):
            self.message = "Display change not confirmed in time; reverted."
            self._persist_if_changed()
            return True
        return False

    # -- drawing -------------------------------------------------------------

    def render(self) -> str:
        schema = self.store.schema
        preview = self.store.preview_values()
        tabs = "  ".join(f"[{i + 1}] {s.capitalize()}" if i != self.section else f"[{i + 1}] {s.upper()}"
                         for i, s in enumerate(self.sections))
        width, height = render_resolution(preview)
        lines = [f"SETTINGS   {tabs}", f"Preset: {detect_preset(preview)} · renders at {width}x{height}", ""]
        for i, key in enumerate(self.keys):
            field = schema.field(key)
            pointer = ">" if i == self.cursor else " "
            staged = "*" if key in self.store.pending else " "
            arrows = "◄ ►" if i == self.cursor and field.steppable else ""
            tags = " ".join(t for t, on in (("[restart]", field.requires_restart),
                                              ("[confirm]", field.needs_confirmation)) if on)
            row = f"{pointer}{staged} {field.label:<20} {field.format(preview[key]):<12} {arrows:<4}{tags}"
            lines.append(row.rstrip())
        selected = schema.field(self.selected)
        lines += ["", f"  {selected.description or selected.describe_type()}"]
        if self.store.pending:
            lines.append(f"  {len(self.store.pending)} unapplied change(s) marked *")
        if self.store.awaiting_confirmation:
            lines.append(f"  Keep these display settings? Reverting in {self.store.seconds_remaining:.0f}s  "
                         "[y] keep  [n] revert")
        if self.message:
            lines.append(f"  {self.message}")
        lines.append(f"  {HELP}")
        return "\n".join(lines)

    # -- input ---------------------------------------------------------------

    def handle(self, command: str) -> None:
        self.message = ""
        command = command.strip()
        try:
            self._dispatch(command)
        except SettingsError as exc:
            self.message = f"! {exc}"
        self._persist_if_changed()

    def _dispatch(self, command: str) -> None:
        store = self.store
        if store.awaiting_confirmation and command in ("y", "n"):
            if command == "y":
                store.confirm()
                self.message = "Display settings kept."
            else:
                store.revert()
                self.message = "Display settings reverted."
            return
        if command == "q":
            self._quit()
        elif command.isdigit() and 1 <= int(command) <= len(self.sections):
            self.section, self.cursor = int(command) - 1, 0
        elif command in ("j", "k"):
            self.cursor = (self.cursor + (1 if command == "j" else -1)) % len(self.keys)
        elif command in ("h", "l", "-", "+"):
            store.adjust(self.selected, 1 if command in ("l", "+") else -1)
        elif command.startswith("="):
            store.stage_text(self.selected, command[1:])
        elif command == "a":
            self._apply()
        elif command == "d":
            store.discard()
            self.message = "Changes discarded."
        elif command == "r":
            store.stage_defaults(self.keys)
            self.message = f"{self.sections[self.section].capitalize()} reset to defaults (apply to keep)."
        elif command.startswith("p "):
            name = command[2:].strip().lower()
            if name not in PRESETS:
                self.message = f"! unknown preset {name!r} (choose {', '.join(PRESETS)})"
            else:
                store.stage_many(PRESETS[name])
                self.message = f"Preset {name} staged (apply to keep)."
        elif command:
            self.message = f"! unknown command {command!r}"

    def _apply(self) -> None:
        result = self.store.apply()
        if not result.changes:
            self.message = "Nothing to apply."
            return
        notes = [f"Applied {len(result.changes)} change(s)."]
        if result.restart_required:
            notes.append("Restart required for: " + ", ".join(
                self.store.schema.field(k).label for k in result.restart_required) + ".")
        self.message = " ".join(notes)

    def _quit(self) -> None:
        # Leaving the menu behaves like "Back": unapplied edits are dropped and
        # an unconfirmed display change is rolled back to the safe values.
        if self.store.awaiting_confirmation:
            self.store.revert()
        self.store.discard()
        self.running = False

    def _persist_if_changed(self) -> None:
        durable = dict(self.store.durable_values)
        if durable == self._saved:
            return
        try:
            self.persist(durable)
        except OSError as exc:
            self.message = f"! could not save settings: {exc}"
            return
        self._saved = durable


def run(menu: SettingsMenu, read_line: Callable[[], str], write: Callable[[str], None],
        clock: Callable[[], float]) -> None:
    """Drive the menu. ``clock`` is a monotonic seconds source; the elapsed
    time between inputs is fed to the store as ``dt``."""
    last = clock()
    while menu.running:
        write(menu.render())
        try:
            line = read_line()
        except EOFError:
            line = "q"
        now = clock()
        expired = menu.tick(now - last)
        last = now
        if expired and line.strip() in ("y", "n"):
            continue  # the answer arrived too late; the revert already happened
        menu.handle(line)
