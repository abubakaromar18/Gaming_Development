"""Pure camera and world simulation. Imports nothing from pygame."""
from .camera import Camera, CameraDebug, damp, damp_factor, push_into_window
from .config import FEATURES, CameraConfig
from .noise import Noise1D
from .vec import ZERO, Rect, Vec2
from .world import STAR_TILE, Controls, Impact, Shockwave, World, WorldConfig, trauma_for

__all__ = [
    "Camera",
    "CameraConfig",
    "CameraDebug",
    "Controls",
    "FEATURES",
    "Impact",
    "Noise1D",
    "Rect",
    "STAR_TILE",
    "Shockwave",
    "Vec2",
    "World",
    "WorldConfig",
    "ZERO",
    "damp",
    "damp_factor",
    "push_into_window",
    "trauma_for",
]
