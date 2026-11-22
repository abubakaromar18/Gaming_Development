"""Tiny immutable 2D vector and axis-aligned rectangle types.

The core deliberately avoids pygame.Vector2 / pygame.Rect so it can be
imported (and tested) without the rendering library.
"""
from __future__ import annotations

import math
from dataclasses import dataclass


@dataclass(frozen=True, slots=True)
class Vec2:
    x: float = 0.0
    y: float = 0.0

    def __add__(self, other: Vec2) -> Vec2:
        return Vec2(self.x + other.x, self.y + other.y)

    def __sub__(self, other: Vec2) -> Vec2:
        return Vec2(self.x - other.x, self.y - other.y)

    def __mul__(self, k: float) -> Vec2:
        return Vec2(self.x * k, self.y * k)

    __rmul__ = __mul__

    def __neg__(self) -> Vec2:
        return Vec2(-self.x, -self.y)

    def length(self) -> float:
        return math.hypot(self.x, self.y)

    def clamp_length(self, max_length: float) -> Vec2:
        """Return this vector scaled down (never up) to at most ``max_length``."""
        n = self.length()
        if n <= max_length or n == 0.0:
            return self
        return self * (max_length / n)

    def rotate(self, radians: float) -> Vec2:
        c, s = math.cos(radians), math.sin(radians)
        return Vec2(self.x * c - self.y * s, self.x * s + self.y * c)


ZERO = Vec2(0.0, 0.0)


@dataclass(frozen=True, slots=True)
class Rect:
    """Axis-aligned rectangle stored as left/top/width/height (y grows downward)."""

    left: float
    top: float
    width: float
    height: float

    def __post_init__(self) -> None:
        if self.width < 0 or self.height < 0:
            raise ValueError(f"Rect size must be non-negative, got {self.width}x{self.height}")

    @classmethod
    def centered(cls, center: Vec2, width: float, height: float) -> Rect:
        return cls(center.x - width / 2, center.y - height / 2, width, height)

    @property
    def right(self) -> float:
        return self.left + self.width

    @property
    def bottom(self) -> float:
        return self.top + self.height

    @property
    def center(self) -> Vec2:
        return Vec2(self.left + self.width / 2, self.top + self.height / 2)

    def contains(self, p: Vec2) -> bool:
        return self.left <= p.x <= self.right and self.top <= p.y <= self.bottom

    def intersects(self, other: Rect) -> bool:
        return (
            self.left < other.right
            and other.left < self.right
            and self.top < other.bottom
            and other.top < self.bottom
        )
