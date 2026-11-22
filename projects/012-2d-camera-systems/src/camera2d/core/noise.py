"""Seeded 1D gradient (Perlin) noise.

Screen shake driven by ``random.uniform`` every frame looks like static:
the offset jumps to an unrelated value each frame, so the amount of motion
depends on the frame rate. Sampling smooth noise at ``time * frequency``
gives a coherent wobble whose speed is set in Hz, independent of frame rate,
and which is fully reproducible from a seed.
"""
from __future__ import annotations

import math
import random

_TABLE_SIZE = 256


class Noise1D:
    """Continuous noise in [-1, 1] with value 0 at every integer lattice point."""

    def __init__(self, rng: random.Random) -> None:
        perm = list(range(_TABLE_SIZE))
        rng.shuffle(perm)
        self._perm = perm
        self._grad = [rng.uniform(-1.0, 1.0) for _ in range(_TABLE_SIZE)]

    def _gradient(self, i: int) -> float:
        return self._grad[self._perm[i % _TABLE_SIZE]]

    @staticmethod
    def _fade(t: float) -> float:
        # Quintic smoothstep: continuous first and second derivatives, so the
        # shake has no visible "kinks" at lattice points.
        return t * t * t * (t * (t * 6.0 - 15.0) + 10.0)

    def sample(self, x: float) -> float:
        i = math.floor(x)
        f = x - i
        d0 = self._gradient(i) * f
        d1 = self._gradient(i + 1) * (f - 1.0)
        value = d0 + (d1 - d0) * self._fade(f)
        # 1D Perlin with gradients in [-1, 1] spans [-0.5, 0.5]; normalize.
        return max(-1.0, min(1.0, value * 2.0))
