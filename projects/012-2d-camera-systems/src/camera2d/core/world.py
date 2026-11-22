"""The demo world: a drifting ship in a large, procedurally generated arena.

It exists to give the camera something worth following - acceleration,
sharp turns, wall impacts, and explosions - and it emits *events* rather than
touching the camera directly. The app turns events into trauma with
:func:`trauma_for`, so the world and the camera never know about each other.
"""
from __future__ import annotations

import math
import random
from dataclasses import dataclass, field

from .vec import ZERO, Rect, Vec2


@dataclass(frozen=True, slots=True)
class WorldConfig:
    width: float = 3200.0
    height: float = 2000.0
    rock_count: int = 34
    rock_min_size: float = 50.0
    rock_max_size: float = 190.0
    spawn_clearance: float = 260.0
    ship_size: float = 28.0
    acceleration: float = 1800.0
    drag: float = 2.2               # 1/s; exponential, so frame-rate independent
    max_speed: float = 620.0
    restitution: float = 0.45       # fraction of speed kept when bouncing off a wall
    shockwave_cooldown: float = 0.6
    shockwave_lifetime: float = 0.5
    star_count: int = 220

    def __post_init__(self) -> None:
        if self.width <= 0 or self.height <= 0:
            raise ValueError("world size must be positive")
        if not 0 < self.rock_min_size <= self.rock_max_size:
            raise ValueError("rock sizes must satisfy 0 < min <= max")
        if self.rock_count < 0 or self.star_count < 0:
            raise ValueError("counts must be non-negative")
        if not 0.0 <= self.restitution <= 1.0:
            raise ValueError("restitution must be in [0, 1]")
        if self.ship_size <= 0 or self.max_speed <= 0:
            raise ValueError("ship_size and max_speed must be positive")


@dataclass(frozen=True, slots=True)
class Impact:
    position: Vec2
    speed: float


@dataclass(frozen=True, slots=True)
class Shockwave:
    position: Vec2


Event = Impact | Shockwave


@dataclass(frozen=True, slots=True)
class Controls:
    """One frame of player intent, already decoupled from devices."""

    move: Vec2 = ZERO
    fire: bool = False   # edge-triggered: true only on the frame it was pressed


@dataclass(slots=True)
class Ring:
    position: Vec2
    age: float = 0.0


@dataclass(frozen=True, slots=True)
class Star:
    """A background star in a repeating tile; ``depth`` in (0, 1) is its parallax factor."""

    x: float
    y: float
    depth: float


STAR_TILE = 1024.0

# Impacts slower than this are scrapes, not hits; at the upper speed a hit
# delivers the maximum impact trauma.
_IMPACT_MIN_SPEED = 120.0
_IMPACT_FULL_SPEED = 600.0
_IMPACT_MAX_TRAUMA = 0.6
_SHOCKWAVE_TRAUMA = 0.55


def trauma_for(event: Event) -> float:
    """Map a gameplay event to camera trauma. Designers tune feel here, in one place."""
    if isinstance(event, Shockwave):
        return _SHOCKWAVE_TRAUMA
    t = (event.speed - _IMPACT_MIN_SPEED) / (_IMPACT_FULL_SPEED - _IMPACT_MIN_SPEED)
    return _IMPACT_MAX_TRAUMA * min(1.0, max(0.0, t))


@dataclass(slots=True)
class World:
    config: WorldConfig
    bounds: Rect
    rocks: list[Rect]
    stars: list[Star]
    position: Vec2
    velocity: Vec2 = ZERO
    rings: list[Ring] = field(default_factory=list)
    _cooldown: float = 0.0

    @classmethod
    def generate(cls, config: WorldConfig, rng: random.Random) -> World:
        bounds = Rect(0.0, 0.0, config.width, config.height)
        spawn = bounds.center
        clear_zone = Rect.centered(spawn, config.spawn_clearance * 2, config.spawn_clearance * 2)
        rocks: list[Rect] = []
        # Rejection sampling with a fixed attempt budget: dense configs just
        # get fewer rocks instead of looping forever.
        attempts = config.rock_count * 30
        while len(rocks) < config.rock_count and attempts > 0:
            attempts -= 1
            w = rng.uniform(config.rock_min_size, config.rock_max_size)
            h = rng.uniform(config.rock_min_size, config.rock_max_size)
            r = Rect(rng.uniform(0, config.width - w), rng.uniform(0, config.height - h), w, h)
            padded = Rect(r.left - 40, r.top - 40, r.width + 80, r.height + 80)
            if r.intersects(clear_zone) or any(padded.intersects(o) for o in rocks):
                continue
            rocks.append(r)
        stars = [
            Star(rng.uniform(0, STAR_TILE), rng.uniform(0, STAR_TILE), rng.choice((0.15, 0.3, 0.5)))
            for _ in range(config.star_count)
        ]
        return cls(config=config, bounds=bounds, rocks=rocks, stars=stars, position=spawn)

    def ship_rect(self, at: Vec2 | None = None) -> Rect:
        s = self.config.ship_size
        return Rect.centered(self.position if at is None else at, s, s)

    def update(self, dt: float, controls: Controls) -> list[Event]:
        if dt < 0:
            raise ValueError(f"dt must be >= 0, got {dt}")
        cfg = self.config
        events: list[Event] = []

        accel = controls.move.clamp_length(1.0) * cfg.acceleration
        velocity = (self.velocity + accel * dt) * math.exp(-cfg.drag * dt)
        self.velocity = velocity.clamp_length(cfg.max_speed)

        # Sub-step so a long frame can't tunnel the ship through a thin rock.
        travel = self.velocity.length() * dt
        steps = max(1, math.ceil(travel / (cfg.ship_size / 2)))
        h = dt / steps
        for _ in range(steps):
            events.extend(self._move_axis(h, axis=0))
            events.extend(self._move_axis(h, axis=1))

        self._cooldown = max(0.0, self._cooldown - dt)
        if controls.fire and self._cooldown == 0.0:
            self._cooldown = cfg.shockwave_cooldown
            self.rings.append(Ring(self.position))
            events.append(Shockwave(self.position))

        for ring in self.rings:
            ring.age += dt
        self.rings = [r for r in self.rings if r.age < cfg.shockwave_lifetime]
        return events

    def _move_axis(self, dt: float, axis: int) -> list[Event]:
        """Move along one axis and resolve overlaps (separating axes avoids corner snags)."""
        v = self.velocity.x if axis == 0 else self.velocity.y
        if v == 0.0:
            return []
        delta = Vec2(v * dt, 0.0) if axis == 0 else Vec2(0.0, v * dt)
        candidate = self.position + delta
        half = self.config.ship_size / 2
        box = self.ship_rect(candidate)

        hit: float | None = None   # resolved coordinate on this axis, if blocked
        for rock in self.rocks:
            if box.intersects(rock):
                edge = (rock.left - half if v > 0 else rock.right + half) if axis == 0 else (
                    rock.top - half if v > 0 else rock.bottom + half
                )
                hit = edge if hit is None else (min(hit, edge) if v > 0 else max(hit, edge))
        lo = (self.bounds.left if axis == 0 else self.bounds.top) + half
        hi = (self.bounds.right if axis == 0 else self.bounds.bottom) - half
        coord = candidate.x if axis == 0 else candidate.y
        if coord < lo or coord > hi:
            wall = lo if coord < lo else hi
            hit = wall if hit is None else (min(hit, wall) if v > 0 else max(hit, wall))

        if hit is None:
            self.position = candidate
            return []
        self.position = Vec2(hit, self.position.y) if axis == 0 else Vec2(self.position.x, hit)
        bounced = -v * self.config.restitution
        self.velocity = Vec2(bounced, self.velocity.y) if axis == 0 else Vec2(self.velocity.x, bounced)
        return [Impact(self.position, abs(v))]
