"""Validated camera tuning parameters.

Every behavior can be switched off independently, which is how the demo lets
you compare them (keys 1-5) and how the tests isolate each stage.
"""
from __future__ import annotations

from dataclasses import dataclass, replace


@dataclass(frozen=True, slots=True)
class CameraConfig:
    # Smooth follow: seconds for the camera to close ~63% of the gap to its
    # goal (an exponential time constant). 0 means hard-lock to the goal.
    follow_enabled: bool = True
    smooth_time: float = 0.18

    # Deadzone: a window (in world units, at zoom 1) around the camera goal
    # inside which the target may move without dragging the camera.
    deadzone_enabled: bool = True
    deadzone_width: float = 160.0
    deadzone_height: float = 110.0

    # Look-ahead: lead the target by velocity * lookahead_time, capped, and
    # eased in so a direction change doesn't whip the view around.
    lookahead_enabled: bool = True
    lookahead_time: float = 0.45
    lookahead_max: float = 170.0
    lookahead_smooth_time: float = 0.35

    # Bounds: keep the visible area inside the world rectangle.
    bounds_enabled: bool = True

    # Trauma shake: trauma in [0, 1] decays linearly; shake = trauma ** power.
    shake_enabled: bool = True
    trauma_decay: float = 0.9          # trauma units per second
    trauma_power: float = 2.0          # 2-3 makes small hits subtle and big hits violent
    shake_max_offset: float = 26.0     # world units at full shake
    shake_max_angle: float = 0.07      # radians at full shake
    shake_frequency: float = 18.0      # noise samples per second (Hz)

    # Zoom range for the demo's mouse wheel.
    min_zoom: float = 0.5
    max_zoom: float = 2.5

    def __post_init__(self) -> None:
        non_negative = {
            "smooth_time": self.smooth_time,
            "deadzone_width": self.deadzone_width,
            "deadzone_height": self.deadzone_height,
            "lookahead_time": self.lookahead_time,
            "lookahead_max": self.lookahead_max,
            "lookahead_smooth_time": self.lookahead_smooth_time,
            "trauma_decay": self.trauma_decay,
            "shake_max_offset": self.shake_max_offset,
            "shake_max_angle": self.shake_max_angle,
            "shake_frequency": self.shake_frequency,
        }
        for name, value in non_negative.items():
            if value < 0:
                raise ValueError(f"{name} must be >= 0, got {value}")
        if self.trauma_power <= 0:
            raise ValueError(f"trauma_power must be > 0, got {self.trauma_power}")
        if not 0 < self.min_zoom <= self.max_zoom:
            raise ValueError(
                f"zoom range must satisfy 0 < min_zoom <= max_zoom, got {self.min_zoom}..{self.max_zoom}"
            )

    def with_changes(self, **changes: object) -> CameraConfig:
        """Return a validated copy with some fields replaced."""
        return replace(self, **changes)

    def toggled(self, feature: str) -> CameraConfig:
        """Flip one of the :data:`FEATURES` switches."""
        if feature not in FEATURES:
            raise KeyError(f"unknown camera feature {feature!r}; expected one of {list(FEATURES)}")
        return replace(self, **{feature: not getattr(self, feature)})


# Switchable stages in display order, with the label the demo shows for each.
FEATURES: dict[str, str] = {
    "follow_enabled": "Smooth follow",
    "deadzone_enabled": "Deadzone",
    "lookahead_enabled": "Look-ahead",
    "bounds_enabled": "Bounds",
    "shake_enabled": "Trauma shake",
}
