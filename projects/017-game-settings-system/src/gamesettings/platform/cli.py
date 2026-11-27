"""Command-line shell: load the file, run one command, save, report.

    gamesettings show [SECTION]
    gamesettings get KEY
    gamesettings set KEY=VALUE [KEY=VALUE ...]
    gamesettings reset [KEY ...]
    gamesettings preset NAME
    gamesettings path
    gamesettings menu

Exit codes: 0 success, 2 invalid input, 1 the file could not be written.
"""
from __future__ import annotations

import argparse
import sys
import time
from collections.abc import Sequence
from pathlib import Path
from typing import TextIO

from ..core import (
    MIGRATIONS,
    PRESETS,
    SettingsError,
    SettingsFile,
    SettingsStore,
    build_schema,
    detect_preset,
    effective_volume,
    render_resolution,
)
from ..core.game import VOLUME_CHANNELS
from .menu import SettingsMenu, run
from .paths import current_config_path


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="gamesettings", description="Inspect and edit game settings.")
    parser.add_argument("--config", type=Path, help="settings file (default: per-user config dir)")
    commands = parser.add_subparsers(dest="command")
    show = commands.add_parser("show", help="list settings (default command)")
    show.add_argument("section", nargs="?")
    commands.add_parser("get", help="print one value").add_argument("key")
    commands.add_parser("set", help="change values").add_argument("assignments", nargs="+", metavar="KEY=VALUE")
    commands.add_parser("reset", help="restore defaults (all, or the given keys)").add_argument("keys", nargs="*")
    commands.add_parser("preset", help="apply a graphics preset").add_argument("name", choices=tuple(PRESETS))
    commands.add_parser("path", help="print the settings file location")
    commands.add_parser("menu", help="interactive options menu")
    return parser


def main(argv: Sequence[str] | None = None, *, stdin: TextIO | None = None,
         stdout: TextIO | None = None, stderr: TextIO | None = None) -> int:
    stdin, stdout, stderr = stdin or sys.stdin, stdout or sys.stdout, stderr or sys.stderr
    args = _parser().parse_args(argv)
    file = SettingsFile(args.config or current_config_path(), build_schema(), MIGRATIONS)

    if args.command == "path":
        print(file.path, file=stdout)
        return 0

    report = file.load()
    for warning in report.warnings:
        print(f"warning: {warning}", file=stderr)
    if report.migrated_from is not None:
        print(f"note: upgraded settings from version {report.migrated_from}", file=stderr)
    store = SettingsStore(file.schema, report.values)

    try:
        if args.command in (None, "show"):
            _show(store, args.section if args.command else None, stdout)
            return 0
        if args.command == "get":
            print(store.schema.field(args.key).format(store.get(args.key)), file=stdout)
            return 0
        if args.command == "menu":
            menu = SettingsMenu(store, file.save)
            run(menu, lambda: stdin.readline() if stdin.readable() else "q",
                lambda text: print(text, file=stdout), time.monotonic)
            return 0
        _stage(store, args)
    except SettingsError as exc:
        print(f"error: {exc}", file=stderr)
        return 2

    try:
        result = store.apply(require_confirmation=False)  # no screen to lose in a terminal
    except SettingsError as exc:
        print(f"error: {exc}", file=stderr)
        return 2
    try:
        file.save(store.durable_values)
    except OSError as exc:
        print(f"error: could not save {file.path}: {exc}", file=stderr)
        return 1
    for change in result.changes:
        field = store.schema.field(change.key)
        print(f"{change.key}: {field.format(change.old)} -> {field.format(change.new)}", file=stdout)
    if not result.changes:
        print("no changes", file=stdout)
    if result.restart_required:
        print(f"restart required for: {', '.join(result.restart_required)}", file=stdout)
    return 0


def _stage(store: SettingsStore, args: argparse.Namespace) -> None:
    if args.command == "set":
        for assignment in args.assignments:
            key, sep, value = assignment.partition("=")
            if not sep:
                raise SettingsError(f"expected KEY=VALUE, got {assignment!r}")
            store.stage_text(key.strip(), value)
    elif args.command == "reset":
        store.stage_defaults(args.keys or None)
    elif args.command == "preset":
        store.stage_many(PRESETS[args.name])


def _show(store: SettingsStore, section: str | None, out: TextIO) -> None:
    schema = store.schema
    sections = schema.sections
    if section is not None and section not in sections:
        raise SettingsError(f"unknown section {section!r} (choose {', '.join(sections)})")
    for name, keys in sections.items():
        if section not in (None, name):
            continue
        print(f"[{name}]", file=out)
        for key in keys:
            field = schema.field(key)
            value = store.get(key)
            marker = " " if value == field.default else "*"
            print(f" {marker} {key:<32} {field.format(value)}", file=out)
    values = store.values
    width, height = render_resolution(values)
    mix = ", ".join(f"{c} {effective_volume(values, c):.2f}" for c in VOLUME_CHANNELS)
    print(f"\npreset: {detect_preset(values)} · render target {width}x{height} · gain {mix}", file=out)
    print("(* = changed from default)", file=out)
