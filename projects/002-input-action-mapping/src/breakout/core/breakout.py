"""Breakout rules as a framework-free simulation.

``Breakout.update(commands, dt)`` advances the game. Time arrives as ``dt`` and
randomness through an injected ``random.Random``, so a test (or a replay) that
feeds the same commands with the same seed gets exactly the same game.

Collision uses axis-aligned boxes with the ball treated as a small square. The
ball moves in sub-steps no longer than its half-size, so it can't tunnel
through a brick or the paddle even after a long frame.
"""
from __future__ import annotations

import math
import random
from dataclasses import dataclass
from enum import Enum

from .commands import Commands


@dataclass(frozen=True)
class GameConfig:
    width: float = 640.0
    height: float = 480.0
    paddle_width: float = 96.0
    paddle_height: float = 12.0
    #: Distance from the bottom of the playfield to the paddle's top edge.
    paddle_offset: float = 40.0
    paddle_speed: float = 520.0
    ball_half: float = 5.0
    serve_speed: float = 320.0
    #: Speed multiplier for every brick broken.
    speedup: float = 1.015
    max_ball_speed: float = 700.0
    #: Launch angle range either side of straight up, in radians.
    max_serve_angle: float = math.radians(30)
    #: Bounce angle off the paddle's very edge, in radians.
    max_bounce_angle: float = math.radians(60)
    brick_rows: int = 6
    brick_cols: int = 10
    brick_height: float = 18.0
    brick_gap: float = 4.0
    brick_top: float = 60.0
    side_margin: float = 16.0
    lives: int = 3

    def __post_init__(self) -> None:
        if self.brick_rows < 1 or self.brick_cols < 1:
            raise ValueError("need at least one row and one column of bricks")
        if self.lives < 1:
            raise ValueError("lives must be at least 1")
        if not 0 < self.paddle_width < self.width:
            raise ValueError("paddle must be narrower than the playfield")
        if self.serve_speed <= 0 or self.max_ball_speed < self.serve_speed:
            raise ValueError("need 0 < serve_speed <= max_ball_speed")


class Phase(Enum):
    SERVING = "serving"
    PLAYING = "playing"
    WON = "won"
    LOST = "lost"


class Event(Enum):
    """Things that happened during an update, for sound, juice, and tests."""

    LAUNCH = "launch"
    WALL = "wall"
    PADDLE = "paddle"
    BRICK = "brick"
    LIFE_LOST = "life_lost"
    WON = "won"
    LOST = "lost"


@dataclass(frozen=True)
class Brick:
    x: float
    y: float
    w: float
    h: float
    row: int


@dataclass
class Ball:
    x: float
    y: float
    vx: float = 0.0
    vy: float = 0.0

    @property
    def speed(self) -> float:
        return math.hypot(self.vx, self.vy)


def build_bricks(cfg: GameConfig) -> list[Brick]:
    usable = cfg.width - 2 * cfg.side_margin - (cfg.brick_cols - 1) * cfg.brick_gap
    w = usable / cfg.brick_cols
    return [
        Brick(
            x=cfg.side_margin + c * (w + cfg.brick_gap),
            y=cfg.brick_top + r * (cfg.brick_height + cfg.brick_gap),
            w=w,
            h=cfg.brick_height,
            row=r,
        )
        for r in range(cfg.brick_rows)
        for c in range(cfg.brick_cols)
    ]


def brick_points(cfg: GameConfig, brick: Brick) -> int:
    """Higher rows are worth more, as in the arcade original."""
    return (cfg.brick_rows - brick.row) * 10


class Breakout:
    def __init__(self, rng: random.Random, config: GameConfig | None = None) -> None:
        self.cfg = config or GameConfig()
        self.rng = rng
        self.new_game()

    # --- Lifecycle ----------------------------------------------------------

    def new_game(self) -> None:
        cfg = self.cfg
        self.bricks: list[Brick] = build_bricks(cfg)
        self.lives = cfg.lives
        self.score = 0
        self.paused = False
        self.paddle_x = (cfg.width - cfg.paddle_width) / 2
        self.ball = Ball(0.0, 0.0)
        self._serve()

    def _serve(self) -> None:
        self.phase = Phase.SERVING
        self.ball.vx = self.ball.vy = 0.0
        self._stick_ball_to_paddle()

    @property
    def paddle_top(self) -> float:
        return self.cfg.height - self.cfg.paddle_offset

    def _stick_ball_to_paddle(self) -> None:
        self.ball.x = self.paddle_x + self.cfg.paddle_width / 2
        self.ball.y = self.paddle_top - self.cfg.ball_half

    # --- Update -------------------------------------------------------------

    def update(self, cmd: Commands, dt: float) -> list[Event]:
        if dt < 0:
            raise ValueError("dt must be non-negative")
        events: list[Event] = []

        if self.phase in (Phase.WON, Phase.LOST):
            if cmd.launch:
                self.new_game()
            return events

        if cmd.toggle_pause:
            self.paused = not self.paused
        if self.paused:
            return events

        self._move_paddle(cmd.move, dt)

        if self.phase is Phase.SERVING:
            self._stick_ball_to_paddle()
            if cmd.launch:
                self._launch()
                events.append(Event.LAUNCH)
            return events

        self._move_ball(dt, events)
        return events

    def _move_paddle(self, move: float, dt: float) -> None:
        move = max(-1.0, min(1.0, move))
        x = self.paddle_x + move * self.cfg.paddle_speed * dt
        self.paddle_x = max(0.0, min(self.cfg.width - self.cfg.paddle_width, x))

    def _launch(self) -> None:
        angle = self.rng.uniform(-self.cfg.max_serve_angle, self.cfg.max_serve_angle)
        self.ball.vx = self.cfg.serve_speed * math.sin(angle)
        self.ball.vy = -self.cfg.serve_speed * math.cos(angle)
        self.phase = Phase.PLAYING

    def _move_ball(self, dt: float, events: list[Event]) -> None:
        distance = self.ball.speed * dt
        steps = max(1, math.ceil(distance / self.cfg.ball_half))
        sub_dt = dt / steps
        for _ in range(steps):
            self._substep(sub_dt, events)
            if self.phase is not Phase.PLAYING:
                return

    def _substep(self, dt: float, events: list[Event]) -> None:
        cfg, ball, r = self.cfg, self.ball, self.cfg.ball_half
        ball.x += ball.vx * dt
        ball.y += ball.vy * dt

        # Walls: clamp back inside and force the velocity away from the wall,
        # which is robust even if the ball somehow ends up deep in a wall.
        if ball.x - r < 0:
            ball.x, ball.vx = r, abs(ball.vx)
            events.append(Event.WALL)
        elif ball.x + r > cfg.width:
            ball.x, ball.vx = cfg.width - r, -abs(ball.vx)
            events.append(Event.WALL)
        if ball.y - r < 0:
            ball.y, ball.vy = r, abs(ball.vy)
            events.append(Event.WALL)

        if ball.y - r > cfg.height:
            self._lose_life(events)
            return

        if ball.vy > 0 and self._overlaps(self.paddle_x, self.paddle_top, cfg.paddle_width, cfg.paddle_height):
            self._bounce_off_paddle()
            events.append(Event.PADDLE)
            return

        self._collide_bricks(events)

    def _overlaps(self, x: float, y: float, w: float, h: float) -> bool:
        b, r = self.ball, self.cfg.ball_half
        return b.x + r > x and b.x - r < x + w and b.y + r > y and b.y - r < y + h

    def _bounce_off_paddle(self) -> None:
        """Bounce angle depends on where the ball hits, giving the player aim.

        Hitting dead centre sends the ball straight up; the edges send it off
        at ``max_bounce_angle``. Incoming angle is ignored on purpose: that is
        what makes Breakout controllable.
        """
        cfg, ball = self.cfg, self.ball
        centre = self.paddle_x + cfg.paddle_width / 2
        reach = cfg.paddle_width / 2 + cfg.ball_half
        offset = max(-1.0, min(1.0, (ball.x - centre) / reach))
        angle = offset * cfg.max_bounce_angle
        speed = ball.speed
        ball.vx = speed * math.sin(angle)
        ball.vy = -speed * math.cos(angle)
        ball.y = self.paddle_top - cfg.ball_half

    def _collide_bricks(self, events: list[Event]) -> None:
        ball, r = self.ball, self.cfg.ball_half
        hits = [b for b in self.bricks if self._overlaps(b.x, b.y, b.w, b.h)]
        if not hits:
            return
        # One brick per sub-step: the one whose centre is nearest the ball.
        brick = min(hits, key=lambda b: (b.x + b.w / 2 - ball.x) ** 2 + (b.y + b.h / 2 - ball.y) ** 2)
        self.bricks.remove(brick)
        self.score += brick_points(self.cfg, brick)
        events.append(Event.BRICK)

        # Reflect on the axis of least penetration and push the ball out.
        pen_left = (ball.x + r) - brick.x
        pen_right = (brick.x + brick.w) - (ball.x - r)
        pen_top = (ball.y + r) - brick.y
        pen_bottom = (brick.y + brick.h) - (ball.y - r)
        if min(pen_left, pen_right) < min(pen_top, pen_bottom):
            if pen_left < pen_right:
                ball.x, ball.vx = brick.x - r, -abs(ball.vx)
            else:
                ball.x, ball.vx = brick.x + brick.w + r, abs(ball.vx)
        else:
            if pen_top < pen_bottom:
                ball.y, ball.vy = brick.y - r, -abs(ball.vy)
            else:
                ball.y, ball.vy = brick.y + brick.h + r, abs(ball.vy)

        speed = ball.speed
        new_speed = min(self.cfg.max_ball_speed, speed * self.cfg.speedup)
        ball.vx *= new_speed / speed
        ball.vy *= new_speed / speed

        if not self.bricks:
            self.phase = Phase.WON
            events.append(Event.WON)

    def _lose_life(self, events: list[Event]) -> None:
        self.lives -= 1
        events.append(Event.LIFE_LOST)
        if self.lives <= 0:
            self.phase = Phase.LOST
            events.append(Event.LOST)
        else:
            self._serve()
