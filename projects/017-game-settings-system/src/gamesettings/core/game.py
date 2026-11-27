"""This game's concrete settings: the schema, its save-format migrations,
graphics quality presets, and the derived values an engine actually reads.

Everything above this module is reusable for any game; this is the one file
a designer edits to add an option.
"""
from __future__ import annotations

import math
from collections.abc import Iterable, Mapping
from typing import Any

from .codec import FlatValues, Migration
from .errors import Violation
from .fields import BoolField, ChoiceField, FloatField, IntField, KeyField, ResolutionField
from .schema import Schema, unique_values

QUALITY = ("low", "medium", "high", "ultra")
ACTIONS = ("move_up", "move_down", "move_left", "move_right", "jump", "interact", "inventory", "pause")
DEFAULT_BINDINGS = ("W", "S", "A", "D", "SPACE", "E", "TAB", "ESCAPE")
VOLUME_CHANNELS = ("music", "sfx", "voice")


def _percent(default: int, label: str, **kw: Any) -> IntField:
    return IntField(default, label, minimum=0, maximum=100, step=5, unit="%", **kw)


def _fps_limit_rule(values: Mapping[str, Any]) -> Iterable[Violation]:
    # 0 means "uncapped"; caps below 30 FPS make the game feel broken, so the
    # int range alone (0..360) isn't enough to express the valid set.
    fps = values["graphics.fps_limit"]
    if 0 < fps < 30:
        yield Violation(("graphics.fps_limit",), f"FPS limit {fps} is too low (use 0 for uncapped, or 30+)")


def build_schema() -> Schema:
    return Schema(
        {
            "graphics": {
                "resolution": ResolutionField("1920x1080", "Resolution", needs_confirmation=True),
                "window_mode": ChoiceField("borderless", "Window mode",
                                           choices=("windowed", "borderless", "fullscreen"),
                                           needs_confirmation=True),
                "vsync": BoolField(True, "V-Sync"),
                "fps_limit": IntField(0, "FPS limit", minimum=0, maximum=360, step=30,
                                      description="0 = uncapped"),
                "render_scale": FloatField(1.0, "Render scale", minimum=0.5, maximum=2.0, step=0.05,
                                           description="Internal resolution multiplier"),
                "texture_quality": ChoiceField("high", "Texture quality", choices=QUALITY, requires_restart=True),
                "shadow_quality": ChoiceField("high", "Shadow quality", choices=QUALITY),
                "anti_aliasing": ChoiceField("taa", "Anti-aliasing", choices=("off", "fxaa", "taa", "msaa4x")),
                "effects_quality": ChoiceField("high", "Effects quality", choices=QUALITY),
                "field_of_view": IntField(90, "Field of view", minimum=60, maximum=120, step=5, unit="°"),
                "brightness": FloatField(1.0, "Brightness", minimum=0.5, maximum=1.5, step=0.05),
            },
            "audio": {
                "master_volume": _percent(80, "Master volume"),
                "music_volume": _percent(70, "Music volume"),
                "sfx_volume": _percent(100, "Effects volume"),
                "voice_volume": _percent(100, "Voice volume"),
                "mute_when_unfocused": BoolField(True, "Mute in background"),
                "subtitles": BoolField(False, "Subtitles"),
            },
            "controls": {
                "mouse_sensitivity": FloatField(1.0, "Mouse sensitivity", minimum=0.1, maximum=10.0, step=0.1),
                "invert_y": BoolField(False, "Invert Y axis"),
                **{f"bind_{action}": KeyField(key, action.replace("_", " ").capitalize())
                   for action, key in zip(ACTIONS, DEFAULT_BINDINGS, strict=True)},
            },
        },
        version=2,
        rules=[
            _fps_limit_rule,
            unique_values((f"controls.bind_{a}" for a in ACTIONS), "key"),
        ],
    )


# -- migrations --------------------------------------------------------------

def _v1_to_v2(raw: FlatValues) -> FlatValues:
    """Version 1 had a ``fullscreen`` checkbox and a 0.0-1.0 master volume.
    Version 2 has a three-way window mode and percentage volumes.

    Migrations run on raw, unvalidated data: they only translate shapes they
    recognize and carry anything odd across unchanged, so field validation
    reports it under the new key.
    """
    if "graphics.fullscreen" in raw:
        fullscreen = raw.pop("graphics.fullscreen")
        raw["graphics.window_mode"] = (
            ("fullscreen" if fullscreen else "windowed") if isinstance(fullscreen, bool) else fullscreen)
    if "audio.volume" in raw:
        volume = raw.pop("audio.volume")
        is_number = isinstance(volume, (int, float)) and not isinstance(volume, bool) and math.isfinite(volume)
        raw["audio.master_volume"] = round(volume * 100) if is_number else volume
    return raw


MIGRATIONS: dict[int, Migration] = {1: _v1_to_v2}


# -- quality presets ---------------------------------------------------------

PRESETS: dict[str, dict[str, Any]] = {
    "low": {"graphics.texture_quality": "low", "graphics.shadow_quality": "low",
            "graphics.effects_quality": "low", "graphics.anti_aliasing": "off", "graphics.render_scale": 0.75},
    "medium": {"graphics.texture_quality": "medium", "graphics.shadow_quality": "medium",
               "graphics.effects_quality": "medium", "graphics.anti_aliasing": "fxaa", "graphics.render_scale": 1.0},
    "high": {"graphics.texture_quality": "high", "graphics.shadow_quality": "high",
             "graphics.effects_quality": "high", "graphics.anti_aliasing": "taa", "graphics.render_scale": 1.0},
    "ultra": {"graphics.texture_quality": "ultra", "graphics.shadow_quality": "ultra",
              "graphics.effects_quality": "ultra", "graphics.anti_aliasing": "msaa4x", "graphics.render_scale": 1.25},
}


def detect_preset(values: Mapping[str, Any]) -> str:
    """Name of the preset the values match exactly, else ``"custom"``.

    The preset is *derived*, not stored: storing it would let it disagree with
    the individual options it claims to describe.
    """
    for name, preset in PRESETS.items():
        if all(values[k] == v for k, v in preset.items()):
            return name
    return "custom"


# -- derived values the engine consumes --------------------------------------

def effective_volume(values: Mapping[str, Any], channel: str, *, focused: bool = True) -> float:
    """Linear gain 0..1 for a mixer bus: master × channel, muted in the
    background if the player asked for that."""
    if channel not in VOLUME_CHANNELS:
        raise ValueError(f"unknown audio channel {channel!r}")
    if not focused and values["audio.mute_when_unfocused"]:
        return 0.0
    return values["audio.master_volume"] / 100 * values[f"audio.{channel}_volume"] / 100


def render_resolution(values: Mapping[str, Any]) -> tuple[int, int]:
    """Internal render target size: output resolution × render scale,
    rounded to even numbers (many upscalers and video encoders need that)."""
    width, height = ResolutionField.size(values["graphics.resolution"])
    scale = values["graphics.render_scale"]
    return max(2, round(width * scale / 2) * 2), max(2, round(height * scale / 2) * 2)


def bindings(values: Mapping[str, Any]) -> dict[str, str]:
    """Action → key map for the input layer."""
    return {action: values[f"controls.bind_{action}"] for action in ACTIONS}
