/**
 * Pong rules as a framework-free simulation.
 *
 * State is a plain immutable object and `step()` is a pure function of
 * (state, input, dt, rng). That makes the game trivially testable, easy to
 * snapshot for interpolation, and safe to replay.
 */
import type { Rng } from "./rng";

export interface Vec2 {
  readonly x: number;
  readonly y: number;
}

export interface Paddle {
  /** Top edge y-coordinate. */
  readonly y: number;
}

export type Phase = "serving" | "playing" | "gameOver";
export type Side = "left" | "right";

export interface PongState {
  readonly phase: Phase;
  readonly ball: { readonly pos: Vec2; readonly vel: Vec2 };
  readonly left: Paddle;
  readonly right: Paddle;
  readonly score: { readonly left: number; readonly right: number };
  /** Who receives the next serve (the ball travels toward them). */
  readonly serveToward: Side;
  readonly winner: Side | null;
}

/** Per-step intent for each paddle: -1 up, 0 still, +1 down. */
export interface PongInput {
  readonly left: -1 | 0 | 1;
  readonly right: -1 | 0 | 1;
  readonly serve: boolean;
}

export const NO_INPUT: PongInput = { left: 0, right: 0, serve: false };

export interface PongConfig {
  readonly width: number;
  readonly height: number;
  readonly paddleWidth: number;
  readonly paddleHeight: number;
  /** Distance from the wall to the paddle's outer face. */
  readonly paddleInset: number;
  readonly paddleSpeed: number;
  readonly ballSize: number;
  readonly serveSpeed: number;
  /** Speed multiplier applied on every paddle hit. */
  readonly speedup: number;
  readonly maxBallSpeed: number;
  /** Max bounce angle off a paddle edge, in radians. */
  readonly maxBounceAngle: number;
  readonly pointsToWin: number;
}

export const DEFAULT_CONFIG: PongConfig = {
  width: 800,
  height: 500,
  paddleWidth: 12,
  paddleHeight: 90,
  paddleInset: 24,
  paddleSpeed: 420,
  ballSize: 12,
  serveSpeed: 360,
  speedup: 1.06,
  maxBallSpeed: 900,
  maxBounceAngle: (5 * Math.PI) / 12, // 75°
  pointsToWin: 7,
};

export function createState(cfg: PongConfig = DEFAULT_CONFIG, serveToward: Side = "right"): PongState {
  const paddleY = (cfg.height - cfg.paddleHeight) / 2;
  return {
    phase: "serving",
    ball: { pos: centeredBall(cfg), vel: { x: 0, y: 0 } },
    left: { y: paddleY },
    right: { y: paddleY },
    score: { left: 0, right: 0 },
    serveToward,
    winner: null,
  };
}

export function leftPaddleX(cfg: PongConfig): number {
  return cfg.paddleInset;
}

export function rightPaddleX(cfg: PongConfig): number {
  return cfg.width - cfg.paddleInset - cfg.paddleWidth;
}

/** Advance the simulation by exactly `dt` seconds. */
export function step(state: PongState, input: PongInput, dt: number, rng: Rng, cfg: PongConfig = DEFAULT_CONFIG): PongState {
  const left = movePaddle(state.left, input.left, dt, cfg);
  const right = movePaddle(state.right, input.right, dt, cfg);

  switch (state.phase) {
    case "gameOver":
      return input.serve ? createState(cfg, state.winner === "left" ? "right" : "left") : { ...state, left, right };
    case "serving":
      if (!input.serve) return { ...state, left, right };
      return { ...state, left, right, phase: "playing", ball: serve(state.serveToward, rng, cfg) };
    case "playing":
      return stepBall({ ...state, left, right }, dt, cfg);
  }
}

function movePaddle(p: Paddle, dir: -1 | 0 | 1, dt: number, cfg: PongConfig): Paddle {
  if (dir === 0) return p;
  const y = clamp(p.y + dir * cfg.paddleSpeed * dt, 0, cfg.height - cfg.paddleHeight);
  return { y };
}

function serve(toward: Side, rng: Rng, cfg: PongConfig) {
  // Random angle within ±30° so serves are never perfectly flat or steep.
  const angle = (rng() * 2 - 1) * (Math.PI / 6);
  const dirX = toward === "right" ? 1 : -1;
  return {
    pos: centeredBall(cfg),
    vel: { x: dirX * Math.cos(angle) * cfg.serveSpeed, y: Math.sin(angle) * cfg.serveSpeed },
  };
}

function stepBall(state: PongState, dt: number, cfg: PongConfig): PongState {
  let { x, y } = state.ball.pos;
  let { x: vx, y: vy } = state.ball.vel;
  const prevX = x;

  x += vx * dt;
  y += vy * dt;

  // Top/bottom walls: reflect and mirror any penetration back into the field.
  if (y < 0) {
    y = -y;
    vy = Math.abs(vy);
  } else if (y + cfg.ballSize > cfg.height) {
    y = 2 * (cfg.height - cfg.ballSize) - y;
    vy = -Math.abs(vy);
  }

  // Paddles. We test whether the ball's leading edge *crossed* the paddle
  // face during this step (a cheap swept test), so a fast ball can't tunnel
  // through a thin paddle.
  const lFace = leftPaddleX(cfg) + cfg.paddleWidth;
  if (vx < 0 && prevX >= lFace && x < lFace && overlapsPaddle(y, state.left, cfg)) {
    x = lFace;
    ({ vx, vy } = bounce(y, state.left, 1, Math.hypot(vx, vy), cfg));
  }
  const rFace = rightPaddleX(cfg);
  if (vx > 0 && prevX + cfg.ballSize <= rFace && x + cfg.ballSize > rFace && overlapsPaddle(y, state.right, cfg)) {
    x = rFace - cfg.ballSize;
    ({ vx, vy } = bounce(y, state.right, -1, Math.hypot(vx, vy), cfg));
  }

  // Scoring: the ball left the field entirely.
  if (x + cfg.ballSize < 0) return scorePoint(state, "right", cfg);
  if (x > cfg.width) return scorePoint(state, "left", cfg);

  return { ...state, ball: { pos: { x, y }, vel: { x: vx, y: vy } } };
}

function overlapsPaddle(ballY: number, p: Paddle, cfg: PongConfig): boolean {
  return ballY + cfg.ballSize >= p.y && ballY <= p.y + cfg.paddleHeight;
}

/**
 * Classic Pong control: where the ball meets the paddle decides the bounce
 * angle. Center hits go flat, edge hits go steep.
 */
function bounce(ballY: number, p: Paddle, dirX: 1 | -1, speed: number, cfg: PongConfig) {
  const ballCenter = ballY + cfg.ballSize / 2;
  const paddleCenter = p.y + cfg.paddleHeight / 2;
  const half = (cfg.paddleHeight + cfg.ballSize) / 2;
  const offset = clamp((ballCenter - paddleCenter) / half, -1, 1);
  const angle = offset * cfg.maxBounceAngle;
  const newSpeed = Math.min(speed * cfg.speedup, cfg.maxBallSpeed);
  return { vx: dirX * Math.cos(angle) * newSpeed, vy: Math.sin(angle) * newSpeed };
}

function scorePoint(state: PongState, scorer: Side, cfg: PongConfig): PongState {
  const score = { ...state.score, [scorer]: state.score[scorer] + 1 };
  const won = score[scorer] >= cfg.pointsToWin;
  return {
    ...state,
    score,
    phase: won ? "gameOver" : "serving",
    winner: won ? scorer : null,
    // The player who was scored on receives the next serve.
    serveToward: scorer === "left" ? "right" : "left",
    ball: { pos: centeredBall(cfg), vel: { x: 0, y: 0 } },
  };
}

function centeredBall(cfg: PongConfig): Vec2 {
  return { x: (cfg.width - cfg.ballSize) / 2, y: (cfg.height - cfg.ballSize) / 2 };
}

function clamp(v: number, lo: number, hi: number): number {
  return v < lo ? lo : v > hi ? hi : v;
}
