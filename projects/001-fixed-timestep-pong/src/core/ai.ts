/**
 * Simple CPU opponent: track the ball's center with a dead zone so the
 * paddle doesn't jitter, and drift back to center when the ball moves away.
 */
import type { PongConfig, PongState } from "./pong";

export function cpuIntent(state: PongState, cfg: PongConfig, deadZone = 10): -1 | 0 | 1 {
  const paddleCenter = state.right.y + cfg.paddleHeight / 2;
  const approaching = state.phase === "playing" && state.ball.vel.x > 0;
  const target = approaching ? state.ball.pos.y + cfg.ballSize / 2 : cfg.height / 2;
  const diff = target - paddleCenter;
  if (Math.abs(diff) <= deadZone) return 0;
  return diff > 0 ? 1 : -1;
}
