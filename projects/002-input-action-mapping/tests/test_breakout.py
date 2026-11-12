import math
import random

import pytest

from breakout.core.breakout import Ball, Breakout, Brick, Event, GameConfig, Phase, brick_points, build_bricks
from breakout.core.commands import IDLE, Commands

LAUNCH = Commands(launch=True)
DT = 1 / 120


def new_game(seed=1, **cfg):
    return Breakout(random.Random(seed), GameConfig(**cfg))


def place_ball(game, x, y, vx, vy):
    game.phase = Phase.PLAYING
    game.ball = Ball(x, y, vx, vy)


def test_layout_fits_the_playfield():
    cfg = GameConfig()
    bricks = build_bricks(cfg)
    assert len(bricks) == cfg.brick_rows * cfg.brick_cols
    assert min(b.x for b in bricks) == pytest.approx(cfg.side_margin)
    assert max(b.x + b.w for b in bricks) == pytest.approx(cfg.width - cfg.side_margin)


def test_top_rows_score_more():
    cfg = GameConfig()
    top = Brick(0, 0, 1, 1, row=0)
    bottom = Brick(0, 0, 1, 1, row=cfg.brick_rows - 1)
    assert brick_points(cfg, top) > brick_points(cfg, bottom)


def test_ball_rides_the_paddle_while_serving():
    g = new_game()
    g.update(Commands(move=1.0), 0.1)
    assert g.phase is Phase.SERVING
    assert g.ball.x == pytest.approx(g.paddle_x + g.cfg.paddle_width / 2)


def test_launch_goes_upward_within_serve_cone():
    for seed in range(20):
        g = new_game(seed)
        assert Event.LAUNCH in g.update(LAUNCH, DT)
        angle = math.atan2(g.ball.vx, -g.ball.vy)
        assert g.ball.vy < 0
        assert abs(angle) <= g.cfg.max_serve_angle + 1e-9
        assert g.ball.speed == pytest.approx(g.cfg.serve_speed)


def test_paddle_is_clamped_and_analog_move_is_proportional():
    g = new_game()
    start = g.paddle_x
    g.update(Commands(move=0.5), 0.1)
    assert g.paddle_x - start == pytest.approx(0.5 * g.cfg.paddle_speed * 0.1)
    g.update(Commands(move=-1.0), 10.0)
    assert g.paddle_x == 0.0
    g.update(Commands(move=1.0), 10.0)
    assert g.paddle_x == g.cfg.width - g.cfg.paddle_width


def test_side_and_top_walls_reflect():
    g = new_game()
    g.bricks.clear()
    g.bricks.append(Brick(0, 0, 1, 1, row=0))  # keep the game from being "won"
    place_ball(g, 8, 200, -300, 0)
    events = g.update(IDLE, 0.05)
    assert Event.WALL in events and g.ball.vx > 0
    place_ball(g, 300, 8, 0, -300)
    g.update(IDLE, 0.05)
    assert g.ball.vy > 0


def test_paddle_hit_angle_depends_on_contact_point():
    def bounce(offset):
        g = new_game()
        centre = g.paddle_x + g.cfg.paddle_width / 2
        place_ball(g, centre + offset, g.paddle_top - 10, 0, 300)
        assert Event.PADDLE in g.update(IDLE, 0.05)
        return g.ball

    middle, right, left = bounce(0), bounce(40), bounce(-40)
    assert middle.vx == pytest.approx(0) and middle.vy < 0
    assert right.vx > 0 and left.vx < 0
    assert right.speed == pytest.approx(300)


def test_breaking_a_brick_scores_and_reflects():
    g = new_game()
    target = g.bricks[-1]  # bottom row, right-most
    place_ball(g, target.x + target.w / 2, target.y + target.h + 20, 0, -300)
    events = g.update(IDLE, 0.1)
    assert Event.BRICK in events
    assert target not in g.bricks
    assert g.score == brick_points(g.cfg, target)
    assert g.ball.vy > 0
    assert g.ball.speed == pytest.approx(300 * g.cfg.speedup)


def test_fast_ball_cannot_tunnel_through_a_brick():
    g = new_game(brick_rows=1, brick_cols=1)
    brick = g.bricks[0]
    # One frame moves the ball far further than the brick is tall.
    place_ball(g, brick.x + brick.w / 2, brick.y + brick.h + 30, 0, -g.cfg.max_ball_speed)
    events = g.update(IDLE, 0.2)
    assert Event.BRICK in events


def test_losing_all_lives_ends_the_game_and_launch_restarts():
    g = new_game(lives=2)
    for expected_lives in (1, 0):
        place_ball(g, 5, g.cfg.height - 1, 0, 400)
        g.paddle_x = g.cfg.width - g.cfg.paddle_width
        events = g.update(IDLE, 0.1)
        assert Event.LIFE_LOST in events
        assert g.lives == expected_lives
    assert g.phase is Phase.LOST and Event.LOST in events
    g.update(LAUNCH, DT)
    assert g.phase is Phase.SERVING and g.lives == 2 and g.score == 0


def test_clearing_the_last_brick_wins():
    g = new_game(brick_rows=1, brick_cols=1)
    b = g.bricks[0]
    place_ball(g, b.x + b.w / 2, b.y + b.h + 10, 0, -300)
    events = g.update(IDLE, 0.1)
    assert g.phase is Phase.WON and Event.WON in events


def test_pause_freezes_everything():
    g = new_game()
    g.update(LAUNCH, DT)
    g.update(Commands(toggle_pause=True), DT)
    before = (g.ball.x, g.ball.y, g.paddle_x)
    g.update(Commands(move=1.0), 1.0)
    assert (g.ball.x, g.ball.y, g.paddle_x) == before
    g.update(Commands(toggle_pause=True), DT)
    assert not g.paused


def test_negative_dt_is_rejected():
    with pytest.raises(ValueError):
        new_game().update(IDLE, -0.1)


@pytest.mark.parametrize("kwargs", [{"brick_rows": 0}, {"lives": 0}, {"paddle_width": 1000}, {"serve_speed": 0}])
def test_invalid_config_is_rejected(kwargs):
    with pytest.raises(ValueError):
        GameConfig(**kwargs)


def scripted_play(seed, frame_dts):
    """A crude autopilot that tracks the ball, fed a sequence of frame times."""
    g = new_game(seed)
    for i, dt in enumerate(frame_dts):
        centre = g.paddle_x + g.cfg.paddle_width / 2
        move = max(-1.0, min(1.0, (g.ball.x - centre) / 20))
        g.update(Commands(move=move, launch=i % 60 == 0), dt)
    return g


def test_same_seed_and_inputs_reproduce_the_same_game():
    dts = [1 / 60] * 3000
    a, b = scripted_play(7, dts), scripted_play(7, dts)
    assert (a.score, a.lives, a.ball, a.paddle_x, len(a.bricks)) == (b.score, b.lives, b.ball, b.paddle_x, len(b.bricks))
    assert a.score > 0


def test_long_soak_keeps_the_ball_in_bounds():
    g = new_game(3)
    rng = random.Random(99)
    for i in range(20000):
        g.update(Commands(move=rng.uniform(-1, 1), launch=i % 30 == 0), rng.uniform(0.001, 0.05))
        r = g.cfg.ball_half
        assert -1e-9 <= g.ball.x - r and g.ball.x + r <= g.cfg.width + 1e-9
        assert g.ball.y - r >= -1e-9
        assert g.ball.speed <= g.cfg.max_ball_speed + 1e-6
