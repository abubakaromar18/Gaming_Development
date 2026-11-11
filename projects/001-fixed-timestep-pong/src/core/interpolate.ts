/**
 * Render-side interpolation. The renderer never draws raw simulation state;
 * it blends the last two fixed-step snapshots by `alpha` so motion stays
 * smooth even when the simulation runs at 20 Hz on a 144 Hz display.
 */
import type { PongState, Vec2 } from "./pong";

export interface RenderView {
  readonly ball: Vec2;
  readonly leftY: number;
  readonly rightY: number;
}

export function lerp(a: number, b: number, t: number): number {
  return a + (b - a) * t;
}

export function interpolate(prev: PongState, curr: PongState, alpha: number): RenderView {
  // A teleport (serve reset after a point) must not be smeared across the
  // field, so snap whenever the phase changed between snapshots.
  const snap = prev.phase !== curr.phase;
  const t = snap ? 1 : alpha;
  return {
    ball: {
      x: lerp(prev.ball.pos.x, curr.ball.pos.x, t),
      y: lerp(prev.ball.pos.y, curr.ball.pos.y, t),
    },
    leftY: lerp(prev.left.y, curr.left.y, alpha),
    rightY: lerp(prev.right.y, curr.right.y, alpha),
  };
}
