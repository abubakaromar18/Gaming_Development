"""A 2D game camera built as a pipeline of small, independently switchable stages.

Each ``update`` runs, in order:

1. **Look-ahead** - offset the point of interest along the target's velocity.
2. **Deadzone** - move the camera's *goal* only when that point leaves a window.
3. **Bounds** - clamp the goal so it can't drift outside the level.
4. **Smooth follow** - ease the camera *position* toward the goal.
5. **Bounds** again - smoothing must never reveal space outside the level.
6. **Trauma decay** - trauma fades; shake is derived from it at render time.

Shake is applied only in the world<->screen transform. It never touches
``position``, so gameplay code that asks "what's on screen?" gets a stable
answer, and the smoothing/deadzone logic never chases its own jitter.
"""
from __future__ import annotations

import math
import random
from dataclasses import dataclass

from .config import CameraConfig
from .noise import Noise1D
from .vec import ZERO, Rect, Vec2


def damp_factor(smooth_time: float, dt: float) -> float:
    """Fraction of the remaining gap to close this frame.

    ``1 - exp(-dt / T)`` is the exact solution of x' = (goal - x) / T, so one
    step of 1/30 s lands exactly where two steps of 1/60 s would: the camera
    feels the same at any frame rate. The common ``lerp(pos, goal, 0.1)``
    does not have this property - it moves twice as fast at 120 FPS.
    """
    if smooth_time <= 0.0:
        return 1.0
    return 1.0 - math.exp(-dt / smooth_time)


def damp(current: Vec2, goal: Vec2, smooth_time: float, dt: float) -> Vec2:
    return current + (goal - current) * damp_factor(smooth_time, dt)


def push_into_window(goal: Vec2, focus: Vec2, half_w: float, half_h: float) -> Vec2:
    """Move ``goal`` the minimum distance so ``focus`` lies within the window around it."""

    def axis(g: float, f: float, half: float) -> float:
        if f > g + half:
            return f - half
        if f < g - half:
            return f + half
        return g

    return Vec2(axis(goal.x, focus.x, half_w), axis(goal.y, focus.y, half_h))


@dataclass(frozen=True, slots=True)
class CameraDebug:
    """What the camera "thought" last frame, for the debug overlay and tests."""

    focus: Vec2
    goal: Vec2
    deadzone: Rect | None


class Camera:
    def __init__(
        self,
        viewport_width: int,
        viewport_height: int,
        config: CameraConfig,
        rng: random.Random,
        world_bounds: Rect | None = None,
    ) -> None:
        if viewport_width <= 0 or viewport_height <= 0:
            raise ValueError(f"viewport must be positive, got {viewport_width}x{viewport_height}")
        self.viewport = Vec2(float(viewport_width), float(viewport_height))
        self.config = config
        self.world_bounds = world_bounds
        self.zoom = 1.0
        self.position = ZERO
        self.trauma = 0.0
        self._goal = ZERO
        self._lookahead = ZERO
        self._time = 0.0
        # One independent noise channel per shaken degree of freedom; sharing
        # one would make x, y, and roll move in lockstep along a diagonal.
        self._noise_x = Noise1D(rng)
        self._noise_y = Noise1D(rng)
        self._noise_angle = Noise1D(rng)
        # Gradient noise is exactly 0 at integer inputs. Distinct phases keep
        # the channels from all crossing zero on the same frame, which would
        # read as the shake briefly freezing.
        self._phases = (rng.random(), rng.random(), rng.random())
        self.debug = CameraDebug(ZERO, ZERO, None)

    # ----------------------------------------------------------------- setup

    def snap_to(self, target: Vec2) -> None:
        """Teleport with no easing (level start, respawn, cutscene cut)."""
        self._lookahead = ZERO
        self._goal = self._clamp(target)
        self.position = self._goal
        self.debug = CameraDebug(target, self._goal, self._deadzone_rect())

    def set_zoom(self, zoom: float) -> None:
        self.zoom = min(self.config.max_zoom, max(self.config.min_zoom, zoom))
        self.position = self._clamp(self.position)
        self._goal = self._clamp(self._goal)

    def zoom_at(self, screen_point: Vec2, zoom: float) -> None:
        """Zoom while keeping the world point under ``screen_point`` fixed (mouse-wheel zoom)."""
        anchor = self.screen_to_world(screen_point, shaken=False)
        old_zoom = self.zoom
        self.set_zoom(zoom)
        if self.zoom == old_zoom:
            return
        # After zooming, the same screen point maps to a different world
        # point; shift the camera by the difference to pin the anchor.
        drift = anchor - self.screen_to_world(screen_point, shaken=False)
        self.position = self._clamp(self.position + drift)
        self._goal = self._clamp(self._goal + drift)

    def add_trauma(self, amount: float) -> None:
        """Accumulate trauma (clamped to [0, 1]). Hits stack; shake follows."""
        if amount < 0:
            raise ValueError(f"trauma amount must be >= 0, got {amount}")
        self.trauma = min(1.0, self.trauma + amount)

    # ---------------------------------------------------------------- update

    def update(self, dt: float, target: Vec2, target_velocity: Vec2 = ZERO) -> None:
        if dt < 0:
            raise ValueError(f"dt must be >= 0, got {dt}")
        cfg = self.config

        if cfg.lookahead_enabled:
            desired = (target_velocity * cfg.lookahead_time).clamp_length(cfg.lookahead_max)
            self._lookahead = damp(self._lookahead, desired, cfg.lookahead_smooth_time, dt)
        else:
            self._lookahead = ZERO
        focus = target + self._lookahead

        deadzone = self._deadzone_rect()
        if deadzone is not None:
            goal = push_into_window(self._goal, focus, deadzone.width / 2, deadzone.height / 2)
        else:
            goal = focus
        # Clamping the goal (not just the position) matters: otherwise the
        # goal wanders past the level edge and, when the player turns back,
        # the camera sits still until the goal returns into range.
        self._goal = self._clamp(goal)

        if cfg.follow_enabled:
            self.position = damp(self.position, self._goal, cfg.smooth_time, dt)
        else:
            self.position = self._goal
        self.position = self._clamp(self.position)

        self.trauma = max(0.0, self.trauma - cfg.trauma_decay * dt)
        self._time += dt
        self.debug = CameraDebug(focus, self._goal, self._deadzone_rect())

    # ----------------------------------------------------------------- shake

    @property
    def shake(self) -> float:
        """Shake intensity in [0, 1]. Squaring trauma keeps small hits subtle."""
        if not self.config.shake_enabled:
            return 0.0
        return self.trauma ** self.config.trauma_power

    @property
    def shake_offset(self) -> Vec2:
        s = self.shake
        if s == 0.0:
            return ZERO
        t = self._time * self.config.shake_frequency
        m = self.config.shake_max_offset * s
        px, py, _ = self._phases
        return Vec2(m * self._noise_x.sample(t + px), m * self._noise_y.sample(t + py))

    @property
    def shake_angle(self) -> float:
        s = self.shake
        if s == 0.0:
            return 0.0
        t = self._time * self.config.shake_frequency + self._phases[2]
        return self.config.shake_max_angle * s * self._noise_angle.sample(t)

    # ------------------------------------------------------------ transforms

    @property
    def view_size(self) -> Vec2:
        """Visible world area. Zooming in (zoom > 1) shows less of the world."""
        return self.viewport * (1.0 / self.zoom)

    def view_rect(self) -> Rect:
        """The logical (unshaken) visible world rectangle, for culling and gameplay."""
        size = self.view_size
        return Rect.centered(self.position, size.x, size.y)

    def world_to_screen(self, p: Vec2, shaken: bool = True) -> Vec2:
        center, angle = self._render_pose(shaken)
        local = (p - center).rotate(-angle) * self.zoom
        return local + self.viewport * 0.5

    def screen_to_world(self, p: Vec2, shaken: bool = True) -> Vec2:
        center, angle = self._render_pose(shaken)
        local = (p - self.viewport * 0.5) * (1.0 / self.zoom)
        return local.rotate(angle) + center

    def parallax_scroll(self, depth: float) -> Vec2:
        """Scroll amount for a background layer: depth 0 is pinned to the screen, 1 moves with the world.

        Far layers move less than the camera, which is what sells depth in a
        2D scene. Shake is included so the whole frame shakes together.
        """
        if depth < 0:
            raise ValueError(f"depth must be >= 0, got {depth}")
        return (self.position + self.shake_offset) * depth

    def _render_pose(self, shaken: bool) -> tuple[Vec2, float]:
        if not shaken:
            return self.position, 0.0
        return self.position + self.shake_offset, self.shake_angle

    # --------------------------------------------------------------- helpers

    def _deadzone_rect(self) -> Rect | None:
        cfg = self.config
        if not cfg.deadzone_enabled:
            return None
        # Deadzone is tuned in screen terms, so it shrinks in world units as
        # you zoom in and keeps the same on-screen size.
        return Rect.centered(self._goal, cfg.deadzone_width / self.zoom, cfg.deadzone_height / self.zoom)

    def _clamp(self, center: Vec2) -> Vec2:
        """Keep the view inside the world; center it on axes where the world is smaller."""
        bounds = self.world_bounds
        if bounds is None or not self.config.bounds_enabled:
            return center
        size = self.view_size

        def axis(c: float, lo: float, extent: float, view: float) -> float:
            if extent <= view:
                return lo + extent / 2
            half = view / 2
            return min(lo + extent - half, max(lo + half, c))

        return Vec2(
            axis(center.x, bounds.left, bounds.width, size.x),
            axis(center.y, bounds.top, bounds.height, size.y),
        )
