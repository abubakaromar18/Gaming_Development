"""Small builders that keep tests readable."""
from breakout.core.resolver import RawInput


def raw(keys=(), buttons=(), axes=None, hats=()) -> RawInput:
    return RawInput(keys=frozenset(keys), buttons=frozenset(buttons), axes=dict(axes or {}), hats=frozenset(hats))


def keys(*names: str) -> RawInput:
    return raw(keys=names)


def pad(buttons=(), axes=None, hats=()) -> RawInput:
    return raw(buttons=buttons, axes=axes, hats=hats)
