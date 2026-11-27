"""The schema: every setting the game has, its type, default, and the rules
that span more than one setting.

Keys are dotted ``section.name`` strings (``graphics.vsync``). Sections give
menus their tabs and the save file its nesting; everything else works on the
flat key space, which keeps diffing, migrations and lookups trivial.
"""
from __future__ import annotations

import difflib
from collections.abc import Callable, Iterable, Mapping
from typing import Any

from .errors import UnknownSettingError, Violation
from .fields import Field

Values = Mapping[str, Any]
Rule = Callable[[Values], Iterable[Violation]]


class Schema:
    def __init__(self, sections: Mapping[str, Mapping[str, Field[Any]]], *, version: int,
                 rules: Iterable[Rule] = ()) -> None:
        if version < 1:
            raise ValueError("schema version must be >= 1")
        self.version = version
        self.rules = tuple(rules)
        self._fields: dict[str, Field[Any]] = {}
        self._sections: dict[str, tuple[str, ...]] = {}
        for section, fields in sections.items():
            self._check_name(section)
            keys = []
            for name, field in fields.items():
                self._check_name(name)
                key = f"{section}.{name}"
                field.bind(key)  # also validates the default
                self._fields[key] = field
                keys.append(key)
            self._sections[section] = tuple(keys)
        violations = self.check(self.defaults())
        if violations:
            raise ValueError(f"schema defaults break its own rules: {violations[0]}")

    @staticmethod
    def _check_name(name: str) -> None:
        if not name or "." in name or not name.replace("_", "").isalnum():
            raise ValueError(f"invalid section/setting name {name!r}")

    @property
    def keys(self) -> tuple[str, ...]:
        return tuple(self._fields)

    @property
    def sections(self) -> Mapping[str, tuple[str, ...]]:
        return dict(self._sections)

    def __contains__(self, key: object) -> bool:
        return key in self._fields

    def field(self, key: str) -> Field[Any]:
        try:
            return self._fields[key]
        except KeyError:
            close = tuple(difflib.get_close_matches(key, self._fields, n=3, cutoff=0.6))
            raise UnknownSettingError(key, close) from None

    def default(self, key: str) -> Any:
        return self.field(key).default

    def defaults(self) -> dict[str, Any]:
        return {key: field.default for key, field in self._fields.items()}

    def check(self, values: Values) -> tuple[Violation, ...]:
        """Run every cross-field rule against a complete set of values."""
        return tuple(v for rule in self.rules for v in rule(values))


def unique_values(keys: Iterable[str], what: str) -> Rule:
    """Rule factory: the given settings must all hold different values.
    Used for key bindings, where two actions on one key is a bug report."""
    keys = tuple(keys)

    def rule(values: Values) -> Iterable[Violation]:
        owners: dict[Any, list[str]] = {}
        for key in keys:
            owners.setdefault(values[key], []).append(key)
        for value, clash in owners.items():
            if len(clash) > 1:
                yield Violation(tuple(clash), f"{what} {value} is used by {', '.join(clash)}")

    return rule
