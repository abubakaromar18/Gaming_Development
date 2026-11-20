/**
 * Pure animation: (turn record, final grid, seconds since the turn) → sprites.
 *
 * The reducer has already decided everything; this module only interpolates.
 * Phase 1 slides every pre-move tile from its old cell to its new one. Phase 2
 * swaps in the final grid, pops merged tiles, and grows spawned ones. No DOM
 * access, so it is tested like the core.
 */
import { tiles, type Grid } from "../core/grid";
import type { TurnInfo } from "../core/reducer";

export const TIMING = {
  /** Seconds for tiles to slide to their destination. */
  slide: 0.1,
  /** Seconds for merge pops and spawns after the slide. */
  settle: 0.14,
  /** Peak scale of a merge pop. */
  popScale: 1.18,
} as const;

export const TURN_DURATION = TIMING.slide + TIMING.settle;

export interface Sprite {
  readonly id: number;
  readonly value: number;
  /** Position in cells; fractional while sliding. */
  readonly x: number;
  readonly y: number;
  readonly scale: number;
}

const clamp01 = (t: number) => Math.min(1, Math.max(0, t));
const easeOutCubic = (t: number) => 1 - (1 - t) ** 3;
const lerp = (a: number, b: number, t: number) => a + (b - a) * t;

/** Scale curve for a merge: 1 → popScale → 1 over the settle phase. */
export function popScale(t: number): number {
  return 1 + (TIMING.popScale - 1) * Math.sin(Math.PI * clamp01(t));
}

export function spritesAt(turn: TurnInfo, grid: Grid, elapsed: number): Sprite[] {
  const sliding = turn.slides.length > 0 && elapsed < TIMING.slide;
  if (sliding) {
    const t = easeOutCubic(clamp01(elapsed / TIMING.slide));
    return turn.slides.map((s) => ({
      id: s.id,
      value: s.value,
      x: lerp(s.from.x, s.to.x, t),
      y: lerp(s.from.y, s.to.y, t),
      scale: 1,
    }));
  }

  // Without slides (a new game) spawns animate from t = 0.
  const settleStart = turn.slides.length > 0 ? TIMING.slide : 0;
  const t = clamp01((elapsed - settleStart) / TIMING.settle);
  const merged = new Set(turn.merges.map((m) => m.id));
  const spawned = new Set(turn.spawns.map((s) => s.id));
  return tiles(grid).map(({ tile, at }) => ({
    id: tile.id,
    value: tile.value,
    x: at.x,
    y: at.y,
    scale: merged.has(tile.id) ? popScale(t) : spawned.has(tile.id) ? easeOutCubic(t) : 1,
  }));
}
