import { describe, expect, it } from "vitest";
import { cpuIntent } from "../src/core/ai";
import { interpolate } from "../src/core/interpolate";
import { FixedTimestepLoop } from "../src/core/loop";
import {
  createState,
  DEFAULT_CONFIG as cfg,
  leftPaddleX,
  NO_INPUT,
  rightPaddleX,
  step,
  type PongState,
} from "../src/core/pong";
import { mulberry32 } from "../src/core/rng";

const DT = 1 / 60;
const rng = () => 0.5; // angle 0 → perfectly horizontal serve

function playing(overrides: Partial<PongState> & { ball: PongState["ball"] }): PongState {
  return { ...createState(cfg), phase: "playing", ...overrides };
}

describe("serving", () => {
  it("waits for the serve input", () => {
    const s = step(createState(cfg), NO_INPUT, DT, rng);
    expect(s.phase).toBe("serving");
  });

  it("launches the ball toward the receiver at serve speed", () => {
    const s = step(createState(cfg, "left"), { ...NO_INPUT, serve: true }, DT, rng);
    expect(s.phase).toBe("playing");
    expect(s.ball.vel.x).toBeLessThan(0);
    expect(Math.hypot(s.ball.vel.x, s.ball.vel.y)).toBeCloseTo(cfg.serveSpeed);
  });
});

describe("paddles", () => {
  it("move at paddle speed and stay inside the field", () => {
    let s = createState(cfg);
    s = step(s, { ...NO_INPUT, left: -1 }, DT, rng);
    expect(s.left.y).toBeCloseTo(createState(cfg).left.y - cfg.paddleSpeed * DT);
    for (let i = 0; i < 600; i++) s = step(s, { ...NO_INPUT, left: -1, right: 1 }, DT, rng);
    expect(s.left.y).toBe(0);
    expect(s.right.y).toBe(cfg.height - cfg.paddleHeight);
  });
});

describe("ball physics", () => {
  it("reflects off the top wall", () => {
    const s = step(playing({ ball: { pos: { x: 400, y: 1 }, vel: { x: 0, y: -300 } } }), NO_INPUT, DT, rng);
    expect(s.ball.vel.y).toBeGreaterThan(0);
    expect(s.ball.pos.y).toBeGreaterThanOrEqual(0);
  });

  it("bounces off a paddle, speeds up, and reverses direction", () => {
    const face = leftPaddleX(cfg) + cfg.paddleWidth;
    const paddleY = 200;
    const ballY = paddleY + cfg.paddleHeight / 2 - cfg.ballSize / 2; // dead center
    const before = playing({ left: { y: paddleY }, ball: { pos: { x: face + 2, y: ballY }, vel: { x: -400, y: 0 } } });
    const s = step(before, NO_INPUT, DT, rng);
    expect(s.ball.vel.x).toBeGreaterThan(0);
    expect(s.ball.vel.y).toBeCloseTo(0);
    expect(Math.abs(s.ball.vel.x)).toBeCloseTo(400 * cfg.speedup);
  });

  it("angles the bounce by where the ball hits the paddle", () => {
    const face = rightPaddleX(cfg);
    const paddleY = 200;
    const edgeHit = playing({ right: { y: paddleY }, ball: { pos: { x: face - cfg.ballSize - 2, y: paddleY }, vel: { x: 400, y: 0 } } });
    const s = step(edgeHit, NO_INPUT, DT, rng);
    expect(s.ball.vel.x).toBeLessThan(0);
    expect(s.ball.vel.y).toBeLessThan(0); // hit above center → deflect upward
  });

  it("does not tunnel through a paddle at maximum speed", () => {
    const face = leftPaddleX(cfg) + cfg.paddleWidth;
    // At 900 px/s the ball moves 15 px per step, more than the paddle's 12 px width.
    const before = playing({ left: { y: 200 }, ball: { pos: { x: face + 1, y: 230 }, vel: { x: -cfg.maxBallSpeed, y: 0 } } });
    const s = step(before, NO_INPUT, DT, rng);
    expect(s.ball.vel.x).toBeGreaterThan(0);
  });

  it("caps ball speed", () => {
    const face = leftPaddleX(cfg) + cfg.paddleWidth;
    const before = playing({ left: { y: 200 }, ball: { pos: { x: face + 1, y: 239 }, vel: { x: -cfg.maxBallSpeed, y: 0 } } });
    const s = step(before, NO_INPUT, DT, rng);
    expect(Math.hypot(s.ball.vel.x, s.ball.vel.y)).toBeLessThanOrEqual(cfg.maxBallSpeed + 1e-9);
  });
});

describe("scoring", () => {
  it("awards a point when the ball passes a paddle and serves to the loser", () => {
    const missed = playing({ left: { y: 0 }, ball: { pos: { x: -cfg.ballSize + 1, y: 400 }, vel: { x: -400, y: 0 } } });
    const s = step(missed, NO_INPUT, DT, rng);
    expect(s.score).toEqual({ left: 0, right: 1 });
    expect(s.phase).toBe("serving");
    expect(s.serveToward).toBe("left");
  });

  it("ends the game at pointsToWin and restarts on serve", () => {
    const nearWin = playing({
      score: { left: cfg.pointsToWin - 1, right: 0 },
      right: { y: 0 },
      ball: { pos: { x: cfg.width - 1, y: 400 }, vel: { x: 400, y: 0 } },
    });
    const over = step(nearWin, NO_INPUT, DT, rng);
    expect(over.phase).toBe("gameOver");
    expect(over.winner).toBe("left");
    const restarted = step(over, { ...NO_INPUT, serve: true }, DT, rng);
    expect(restarted.score).toEqual({ left: 0, right: 0 });
    expect(restarted.phase).toBe("serving");
  });
});

describe("determinism", () => {
  /** Run exactly `totalSteps` fixed steps, fed by frames of `frameSeconds`. */
  function simulate(frameSeconds: number, totalSteps: number): PongState {
    const r = mulberry32(42);
    let s = createState(cfg);
    let steps = 0;
    const loop = new FixedTimestepLoop({ hz: 120 }, (dt) => {
      if (steps >= totalSteps) return;
      s = step(s, { left: -1, right: cpuIntent(s, cfg), serve: s.phase !== "playing" }, dt, r);
      steps++;
    });
    while (steps < totalSteps) loop.advance(frameSeconds);
    return s;
  }

  it("reaches the same state regardless of display frame rate", () => {
    // Left paddle parks at the top so points get scored. 20 simulated seconds, delivered as 240 Hz frames vs jittery 30 Hz frames.
    const a = simulate(1 / 240, 2400);
    const b = simulate(1 / 30 + 0.0007, 2400);
    expect(a.score.left + a.score.right).toBeGreaterThan(0); // the match actually progressed
    expect(a).toEqual(b);
  });
});

describe("interpolation", () => {
  it("blends positions by alpha", () => {
    const a = playing({ ball: { pos: { x: 100, y: 100 }, vel: { x: 1, y: 1 } } });
    const b = { ...a, ball: { ...a.ball, pos: { x: 200, y: 300 } } };
    expect(interpolate(a, b, 0.25).ball).toEqual({ x: 125, y: 150 });
  });

  it("snaps instead of smearing across a point reset", () => {
    const a = playing({ ball: { pos: { x: 790, y: 100 }, vel: { x: 1, y: 0 } } });
    const b = { ...createState(cfg), phase: "serving" as const };
    expect(interpolate(a, b, 0.1).ball).toEqual(b.ball.pos);
  });
});

describe("cpu", () => {
  it("tracks an approaching ball and recenters otherwise", () => {
    const high = playing({ right: { y: 300 }, ball: { pos: { x: 400, y: 50 }, vel: { x: 300, y: 0 } } });
    expect(cpuIntent(high, cfg)).toBe(-1);
    const leaving = { ...high, ball: { ...high.ball, vel: { x: -300, y: 0 } } };
    expect(cpuIntent(leaving, cfg)).toBe(-1); // paddle center 345 > field center 250
    const centered = { ...leaving, right: { y: (cfg.height - cfg.paddleHeight) / 2 } };
    expect(cpuIntent(centered, cfg)).toBe(0);
  });
});
