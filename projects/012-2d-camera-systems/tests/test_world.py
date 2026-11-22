import math
import random

import pytest

from camera2d.core import Controls, Impact, Rect, Shockwave, Vec2, World, WorldConfig, trauma_for


def make_world(seed=5, **changes):
    return World.generate(WorldConfig(**changes), random.Random(seed))


def empty_world(**changes):
    return make_world(rock_count=0, **changes)


def test_generation_is_deterministic_and_keeps_spawn_clear():
    a, b = make_world(11), make_world(11)
    assert a.rocks == b.rocks and a.stars == b.stars
    assert make_world(12).rocks != a.rocks
    spawn_zone = Rect.centered(a.position, 520, 520)
    assert a.rocks and not any(r.intersects(spawn_zone) for r in a.rocks)
    assert all(r.left >= 0 and r.top >= 0 and r.right <= 3200 and r.bottom <= 2000 for r in a.rocks)
    for i, r in enumerate(a.rocks):
        assert not any(r.intersects(o) for o in a.rocks[i + 1:])


def test_generation_gives_up_gracefully_when_too_dense():
    w = make_world(width=400, height=400, rock_count=500, spawn_clearance=10)
    assert 0 < len(w.rocks) < 500


def test_config_validation():
    with pytest.raises(ValueError):
        WorldConfig(width=0)
    with pytest.raises(ValueError):
        WorldConfig(restitution=1.5)
    with pytest.raises(ValueError):
        WorldConfig(rock_min_size=100, rock_max_size=50)


def test_ship_accelerates_and_caps_speed():
    w = empty_world(width=100_000, height=100_000)
    for _ in range(600):
        w.update(1 / 60, Controls(Vec2(1, 0)))
    assert w.velocity.x > 0 and w.velocity.length() <= w.config.max_speed + 1e-9
    assert w.position.x > 50_000


def test_drag_brings_ship_to_rest():
    w = empty_world(width=100_000, height=100_000)
    for _ in range(60):
        w.update(1 / 60, Controls(Vec2(0, 1)))
    for _ in range(600):
        w.update(1 / 60, Controls())
    assert w.velocity.length() < 1.0


def test_diagonal_input_is_not_faster():
    a, b = empty_world(width=100_000, height=100_000), empty_world(width=100_000, height=100_000)
    for _ in range(30):
        a.update(1 / 60, Controls(Vec2(1, 0)))
        b.update(1 / 60, Controls(Vec2(1, 1)))
    assert math.isclose(a.velocity.length(), b.velocity.length())


def test_hitting_wall_bounces_and_reports_impact():
    w = empty_world()
    w.position = Vec2(3100, 1000)
    w.velocity = Vec2(500, 0)
    events = []
    for _ in range(30):
        events += w.update(1 / 60, Controls())
    impacts = [e for e in events if isinstance(e, Impact)]
    assert impacts and impacts[0].speed > 250  # drag bled some speed on the way
    assert w.velocity.x < 0
    assert w.position.x <= 3200 - w.config.ship_size / 2


def test_rock_blocks_even_on_a_huge_frame():
    w = empty_world()
    w.rocks = [Rect(1700, 900, 10, 200)]
    w.position = Vec2(1600, 1000)
    w.velocity = Vec2(600, 0)
    events = w.update(0.5, Controls())  # 300 units in one frame: would tunnel without sub-steps
    assert any(isinstance(e, Impact) for e in events)
    assert w.ship_rect().right <= 1700 + 1e-9


def test_ship_never_overlaps_rocks_or_leaves_world_soak():
    w = make_world(21)
    rng = random.Random(21)
    for _ in range(20_000):
        move = Vec2(rng.uniform(-1, 1), rng.uniform(-1, 1))
        w.update(rng.uniform(0.0, 0.05), Controls(move, rng.random() < 0.02))
        ship = w.ship_rect()
        assert not any(ship.intersects(r) for r in w.rocks)
        assert w.bounds.contains(Vec2(ship.left, ship.top)) and w.bounds.contains(Vec2(ship.right, ship.bottom))


def test_shockwave_respects_cooldown_and_rings_expire():
    w = empty_world()
    first = w.update(1 / 60, Controls(fire=True))
    assert [type(e) for e in first] == [Shockwave]
    assert w.update(1 / 60, Controls(fire=True)) == []  # cooling down
    for _ in range(40):
        w.update(1 / 60, Controls())
    assert w.rings == []
    assert any(isinstance(e, Shockwave) for e in w.update(1 / 60, Controls(fire=True)))


def test_trauma_mapping():
    assert trauma_for(Shockwave(Vec2())) > 0.5
    assert trauma_for(Impact(Vec2(), 50)) == 0.0          # scrape
    assert 0 < trauma_for(Impact(Vec2(), 300)) < trauma_for(Impact(Vec2(), 500))
    assert trauma_for(Impact(Vec2(), 10_000)) == trauma_for(Impact(Vec2(), 600))


def test_simulation_is_deterministic():
    def play(seed):
        w = make_world(seed)
        rng = random.Random(seed)
        for _ in range(3000):
            w.update(1 / 60, Controls(Vec2(rng.uniform(-1, 1), rng.uniform(-1, 1)), rng.random() < 0.01))
        return w.position, w.velocity

    assert play(3) == play(3)


def test_negative_dt_rejected():
    with pytest.raises(ValueError):
        empty_world().update(-1, Controls())
