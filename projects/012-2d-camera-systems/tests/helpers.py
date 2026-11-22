import math
import random

from camera2d.core import Camera, CameraConfig, Rect, Vec2

# Every stage off: a hard-locked camera. Tests switch on only what they test.
BARE = CameraConfig(
    follow_enabled=False,
    deadzone_enabled=False,
    lookahead_enabled=False,
    bounds_enabled=False,
    shake_enabled=False,
)


def make_camera(config=BARE, bounds=None, size=(800, 600), seed=1):
    return Camera(size[0], size[1], config, random.Random(seed), bounds)


def close(a: Vec2, b: Vec2, tol: float = 1e-6) -> bool:
    return math.isclose(a.x, b.x, abs_tol=tol) and math.isclose(a.y, b.y, abs_tol=tol)


BIG_WORLD = Rect(0, 0, 4000, 3000)
