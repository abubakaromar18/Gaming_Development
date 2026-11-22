import math

import pytest

from camera2d.core import Rect, Vec2
from helpers import close


def test_arithmetic():
    assert Vec2(1, 2) + Vec2(3, 4) == Vec2(4, 6)
    assert Vec2(3, 4) - Vec2(1, 1) == Vec2(2, 3)
    assert 2 * Vec2(1, -2) == Vec2(2, -4)
    assert -Vec2(1, -2) == Vec2(-1, 2)
    assert Vec2(3, 4).length() == 5


def test_clamp_length_only_shrinks():
    assert Vec2(30, 40).clamp_length(10) == Vec2(6, 8)
    assert Vec2(3, 4).clamp_length(10) == Vec2(3, 4)
    assert Vec2(0, 0).clamp_length(0) == Vec2(0, 0)


def test_rotate_quarter_turn():
    assert close(Vec2(1, 0).rotate(math.pi / 2), Vec2(0, 1))


def test_rect_geometry():
    r = Rect.centered(Vec2(10, 20), 4, 6)
    assert (r.left, r.top, r.right, r.bottom) == (8, 17, 12, 23)
    assert r.center == Vec2(10, 20)
    assert r.contains(Vec2(12, 23)) and not r.contains(Vec2(12.1, 20))
    assert r.intersects(Rect(11, 22, 5, 5))
    assert not r.intersects(Rect(12, 17, 5, 5))  # touching edges is not overlap


def test_rect_rejects_negative_size():
    with pytest.raises(ValueError):
        Rect(0, 0, -1, 5)
