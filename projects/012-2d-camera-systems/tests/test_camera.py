import math
import random

import pytest

from camera2d.core import Camera, CameraConfig, Rect, Vec2, damp, damp_factor, push_into_window
from helpers import BARE, BIG_WORLD, close, make_camera


# ------------------------------------------------------------ smooth follow

def test_damp_factor_edges():
    assert damp_factor(0.0, 1 / 60) == 1.0
    assert damp_factor(0.5, 0.0) == 0.0
    assert math.isclose(damp_factor(0.2, 0.2), 1 - math.exp(-1))


def test_damping_is_frame_rate_independent():
    goal = Vec2(500, -300)
    a = b = Vec2(0, 0)
    for _ in range(120):
        a = damp(a, goal, 0.25, 1 / 120)
    for _ in range(30):
        b = damp(b, goal, 0.25, 1 / 30)
    assert close(a, b, 1e-9)


def test_follow_eases_toward_static_target_without_overshoot():
    cam = make_camera(BARE.with_changes(follow_enabled=True, smooth_time=0.2))
    target = Vec2(400, 0)
    xs = []
    for _ in range(90):
        cam.update(1 / 60, target)
        xs.append(cam.position.x)
    assert xs == sorted(xs)  # monotonic
    assert all(x <= 400 for x in xs)
    assert 399 < xs[-1] <= 400
    # After one time constant, ~63% of the gap is closed.
    cam2 = make_camera(BARE.with_changes(follow_enabled=True, smooth_time=0.2))
    cam2.update(0.2, target)
    assert math.isclose(cam2.position.x, 400 * (1 - math.exp(-1)))


def test_follow_disabled_locks_to_target():
    cam = make_camera()
    cam.update(1 / 60, Vec2(123, 456))
    assert cam.position == Vec2(123, 456)


def test_zero_dt_changes_nothing_and_negative_dt_rejected():
    cam = make_camera(CameraConfig(), BIG_WORLD)
    cam.snap_to(Vec2(1000, 1000))
    cam.add_trauma(0.5)
    before = (cam.position, cam.trauma)
    cam.update(0.0, Vec2(1500, 1500), Vec2(300, 0))
    assert (cam.position, cam.trauma) == before
    with pytest.raises(ValueError):
        cam.update(-0.01, Vec2(0, 0))


# ---------------------------------------------------------------- deadzone

def test_push_into_window():
    assert push_into_window(Vec2(0, 0), Vec2(5, -5), 10, 10) == Vec2(0, 0)
    assert push_into_window(Vec2(0, 0), Vec2(25, 0), 10, 10) == Vec2(15, 0)
    assert push_into_window(Vec2(0, 0), Vec2(-12, 30), 10, 10) == Vec2(-2, 20)


def test_target_inside_deadzone_does_not_move_camera():
    cam = make_camera(BARE.with_changes(deadzone_enabled=True, deadzone_width=200, deadzone_height=100))
    cam.snap_to(Vec2(0, 0))
    for x in range(-90, 91, 10):
        cam.update(1 / 60, Vec2(x, 40))
    assert cam.position == Vec2(0, 0)


def test_leaving_deadzone_drags_camera_by_the_excess():
    cam = make_camera(BARE.with_changes(deadzone_enabled=True, deadzone_width=200, deadzone_height=100))
    cam.snap_to(Vec2(0, 0))
    cam.update(1 / 60, Vec2(130, -80))
    assert cam.position == Vec2(30, -30)
    assert cam.debug.deadzone == Rect.centered(Vec2(30, -30), 200, 100)
    # Walking back inside leaves the camera where it is.
    cam.update(1 / 60, Vec2(60, -40))
    assert cam.position == Vec2(30, -30)


def test_deadzone_keeps_screen_size_when_zoomed():
    cam = make_camera(BARE.with_changes(deadzone_enabled=True, deadzone_width=200, deadzone_height=100))
    cam.set_zoom(2.0)
    cam.update(1 / 60, Vec2(0, 0))
    assert cam.debug.deadzone.width == 100 and cam.debug.deadzone.height == 50


# -------------------------------------------------------------- look-ahead

def lookahead_camera(**changes):
    return make_camera(
        BARE.with_changes(lookahead_enabled=True, lookahead_time=0.5, lookahead_max=100, **changes)
    )


def test_lookahead_leads_in_direction_of_motion():
    cam = lookahead_camera(lookahead_smooth_time=0.0)
    cam.update(1 / 60, Vec2(0, 0), Vec2(100, 0))
    assert cam.position == Vec2(50, 0)
    cam.update(1 / 60, Vec2(0, 0), Vec2(0, -60))
    assert close(cam.position, Vec2(0, -30))


def test_lookahead_is_capped():
    cam = lookahead_camera(lookahead_smooth_time=0.0)
    cam.update(1 / 60, Vec2(0, 0), Vec2(3000, 4000))
    assert close(cam.position, Vec2(60, 80))


def test_lookahead_eases_in_after_direction_change():
    cam = lookahead_camera(lookahead_smooth_time=0.3)
    cam.update(1 / 60, Vec2(0, 0), Vec2(200, 0))
    first = cam.debug.focus.x
    assert 0 < first < 10  # starts small instead of snapping 100 units ahead
    for _ in range(180):
        cam.update(1 / 60, Vec2(0, 0), Vec2(200, 0))
    assert math.isclose(cam.debug.focus.x, 100, abs_tol=0.01)


def test_lookahead_disabled_resets_offset():
    cam = lookahead_camera(lookahead_smooth_time=0.0)
    cam.update(1 / 60, Vec2(0, 0), Vec2(200, 0))
    cam.config = cam.config.toggled("lookahead_enabled")
    cam.update(1 / 60, Vec2(0, 0), Vec2(200, 0))
    assert cam.position == Vec2(0, 0)


# ------------------------------------------------------------------ bounds

def test_view_never_leaves_world():
    cfg = CameraConfig(shake_enabled=False)
    cam = make_camera(cfg, BIG_WORLD)
    rng = random.Random(9)
    target, vel = Vec2(2000, 1500), Vec2(0, 0)
    for _ in range(5000):
        vel = Vec2(rng.uniform(-900, 900), rng.uniform(-900, 900))
        target = Vec2(min(4200, max(-200, target.x + vel.x / 60)), min(3200, max(-200, target.y + vel.y / 60)))
        cam.update(rng.uniform(0.0, 0.05), target, vel)
        v = cam.view_rect()
        assert v.left >= -1e-9 and v.top >= -1e-9
        assert v.right <= BIG_WORLD.right + 1e-9 and v.bottom <= BIG_WORLD.bottom + 1e-9


def test_world_smaller_than_view_is_centered():
    cam = make_camera(BARE.with_changes(bounds_enabled=True), Rect(100, 100, 300, 2000))
    cam.update(1 / 60, Vec2(-5000, -5000))
    assert cam.position == Vec2(250, 400)  # x centered (narrow), y clamped to top


def test_camera_leaves_edge_immediately_when_player_turns_back():
    # The goal is clamped too, so it can't drift past the edge and stall the camera.
    cfg = BARE.with_changes(bounds_enabled=True, deadzone_enabled=True, deadzone_width=200, deadzone_height=200)
    cam = make_camera(cfg, Rect(0, 0, 2000, 2000))
    cam.snap_to(Vec2(1000, 1000))
    cam.update(1 / 60, Vec2(1900, 1000))  # far right: camera pinned at x = 1600
    assert cam.position.x == 1600
    cam.update(1 / 60, Vec2(1400, 1000))  # 200 back is just past the deadzone's left edge
    assert cam.position.x == 1500


def test_bounds_follow_zoom():
    cam = make_camera(BARE.with_changes(bounds_enabled=True), Rect(0, 0, 2000, 2000))
    cam.update(1 / 60, Vec2(0, 0))
    assert cam.position == Vec2(400, 300)
    cam.set_zoom(0.5)  # view doubles to 1600x1200
    assert cam.position == Vec2(800, 600)


def test_bounds_toggle_off():
    cam = make_camera(BARE, Rect(0, 0, 100, 100))
    cam.update(1 / 60, Vec2(-500, 900))
    assert cam.position == Vec2(-500, 900)


# ------------------------------------------------------------------- shake

def shaky(seed=1, **changes):
    return make_camera(BARE.with_changes(shake_enabled=True, **changes), seed=seed)


def test_no_trauma_no_shake():
    cam = shaky()
    for _ in range(30):
        cam.update(1 / 60, Vec2(0, 0))
    assert cam.shake == 0.0 and cam.shake_offset == Vec2(0, 0) and cam.shake_angle == 0.0


def test_trauma_accumulates_clamps_and_rejects_negative():
    cam = shaky()
    cam.add_trauma(0.4)
    cam.add_trauma(0.4)
    assert math.isclose(cam.trauma, 0.8)
    cam.add_trauma(5)
    assert cam.trauma == 1.0
    with pytest.raises(ValueError):
        cam.add_trauma(-0.1)


def test_trauma_decays_linearly_to_zero():
    cam = shaky(trauma_decay=0.5)
    cam.add_trauma(1.0)
    cam.update(0.5, Vec2(0, 0))
    assert math.isclose(cam.trauma, 0.75)
    cam.update(10.0, Vec2(0, 0))
    assert cam.trauma == 0.0


def test_shake_is_trauma_squared_and_bounded():
    cam = shaky(trauma_power=2.0, shake_max_offset=20, shake_max_angle=0.1, trauma_decay=0.0)
    cam.add_trauma(0.5)
    assert cam.shake == 0.25
    for _ in range(600):
        cam.update(1 / 60, Vec2(0, 0))
        o = cam.shake_offset
        assert abs(o.x) <= 5 and abs(o.y) <= 5
        assert abs(cam.shake_angle) <= 0.025


def test_shake_is_reproducible_from_seed():
    def trace(seed):
        cam = shaky(seed=seed)
        cam.add_trauma(1.0)
        out = []
        for _ in range(60):
            cam.update(1 / 60, Vec2(0, 0))
            out.append((cam.shake_offset, cam.shake_angle))
        return out

    assert trace(3) == trace(3)
    assert trace(3) != trace(4)


def test_shake_does_not_move_logical_position():
    cam = make_camera(CameraConfig(trauma_decay=0.0), BIG_WORLD)
    cam.snap_to(Vec2(2000, 1500))
    cam.add_trauma(1.0)
    for _ in range(20):
        cam.update(1 / 60, Vec2(2000, 1500))
    assert cam.position == Vec2(2000, 1500)
    assert cam.shake_offset != Vec2(0, 0)
    center = Vec2(400, 300)
    assert cam.screen_to_world(center, shaken=False) == Vec2(2000, 1500)
    assert cam.screen_to_world(center) != Vec2(2000, 1500)


def test_shake_disabled_suppresses_offsets_but_keeps_trauma():
    cam = make_camera(BARE)
    cam.add_trauma(1.0)
    cam.update(1 / 60, Vec2(0, 0))
    assert cam.trauma > 0 and cam.shake == 0.0 and cam.shake_offset == Vec2(0, 0)


# -------------------------------------------------------------- transforms

def test_world_screen_round_trip_with_zoom_and_shake():
    cam = make_camera(CameraConfig(trauma_decay=0.0), BIG_WORLD)
    cam.snap_to(Vec2(1700, 900))
    cam.set_zoom(1.7)
    cam.add_trauma(1.0)
    for _ in range(13):
        cam.update(1 / 60, Vec2(1700, 900))
    assert cam.shake_angle != 0.0
    rng = random.Random(2)
    for _ in range(200):
        p = Vec2(rng.uniform(0, 4000), rng.uniform(0, 3000))
        assert close(cam.screen_to_world(cam.world_to_screen(p)), p, 1e-6)


def test_camera_center_maps_to_screen_center_and_zoom_scales():
    cam = make_camera()
    cam.snap_to(Vec2(100, 100))
    assert cam.world_to_screen(Vec2(100, 100)) == Vec2(400, 300)
    cam.set_zoom(2.0)
    assert cam.world_to_screen(Vec2(110, 100)) == Vec2(420, 300)
    assert cam.view_size == Vec2(400, 300)


def test_zoom_is_clamped():
    cam = make_camera(BARE.with_changes(min_zoom=0.5, max_zoom=2.0))
    cam.set_zoom(10)
    assert cam.zoom == 2.0
    cam.set_zoom(0.01)
    assert cam.zoom == 0.5


def test_zoom_at_keeps_point_under_cursor():
    cam = make_camera()
    cam.snap_to(Vec2(0, 0))
    cursor = Vec2(700, 100)
    anchor = cam.screen_to_world(cursor)
    cam.zoom_at(cursor, 2.0)
    assert cam.zoom == 2.0
    assert close(cam.screen_to_world(cursor), anchor)
    cam.zoom_at(cursor, 0.7)
    assert close(cam.screen_to_world(cursor), anchor)


def test_parallax_scroll():
    cam = make_camera()
    cam.snap_to(Vec2(1000, -400))
    assert cam.parallax_scroll(0.0) == Vec2(0, 0)
    assert cam.parallax_scroll(1.0) == Vec2(1000, -400)
    assert cam.parallax_scroll(0.25) == Vec2(250, -100)
    with pytest.raises(ValueError):
        cam.parallax_scroll(-1)


def test_rejects_bad_viewport():
    with pytest.raises(ValueError):
        Camera(0, 600, CameraConfig(), random.Random(0))


# ---------------------------------------------------------------- pipeline

def test_full_pipeline_frame_rate_independent_for_steady_motion():
    def run(steps, dt):
        cam = make_camera(CameraConfig(shake_enabled=False), BIG_WORLD)
        cam.snap_to(Vec2(1000, 1000))
        vel = Vec2(300, 120)
        for i in range(1, steps + 1):
            cam.update(dt, Vec2(1000, 1000) + vel * (i * dt), vel)
        return cam.position

    a, b = run(240, 1 / 120), run(60, 1 / 30)
    assert (a - b).length() < 6.0  # same feel at 30 and 120 FPS (deadzone adds small step error)


def test_shake_channels_never_all_freeze_together():
    cam = shaky(trauma_decay=0.0)
    cam.add_trauma(1.0)
    for _ in range(600):
        cam.update(1 / 60, Vec2(0, 0))
        o = cam.shake_offset
        assert (o.x, o.y, cam.shake_angle) != (0.0, 0.0, 0.0)
