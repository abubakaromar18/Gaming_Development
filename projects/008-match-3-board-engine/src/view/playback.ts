/**
 * Turns an engine TurnResult into a queue of timed animation phases and
 * plays them back with an injected dt. The engine resolves a whole turn
 * instantly; this layer only decides how long each recorded step is shown.
 * No DOM access, so it is unit-testable like the core.
 */
import type { Board, Pos } from "../core/board";
import type { CascadeStep } from "../core/cascade";
import type { TurnResult } from "../core/game";

export const TIMING = {
  /** Seconds for two gems to trade places. */
  swap: 0.16,
  /** Seconds for an illegal swap to go there and come back. */
  bounce: 0.3,
  /** Seconds for matched gems to shrink away. */
  clear: 0.22,
  /** Fall acceleration in cells/s². Constant gravity reads better than linear tweens. */
  gravity: 60,
  shuffle: 0.6,
} as const;

export type Phase =
  | { readonly kind: "swap"; readonly board: Board; readonly a: Pos; readonly b: Pos; readonly bounce: boolean; readonly duration: number }
  | { readonly kind: "clear"; readonly step: CascadeStep; readonly duration: number }
  | { readonly kind: "fall"; readonly step: CascadeStep; readonly duration: number }
  | { readonly kind: "shuffle"; readonly from: Board; readonly to: Board; readonly duration: number };

/** Time for a gem to fall `cells` rows from rest: d = ½·g·t². */
export function fallTime(cells: number): number {
  return Math.sqrt((2 * cells) / TIMING.gravity);
}

/** Offset in rows after falling for `seconds`, capped at the landing row. */
export function fallDistance(seconds: number, maxCells: number): number {
  return Math.min(maxCells, 0.5 * TIMING.gravity * seconds * seconds);
}

export function phasesForTurn(board: Board, a: Pos, b: Pos, result: TurnResult): Phase[] {
  if (!result.ok) {
    return result.reason === "no-match" ? [{ kind: "swap", board, a, b, bounce: true, duration: TIMING.bounce }] : [];
  }
  const phases: Phase[] = [{ kind: "swap", board, a, b, bounce: false, duration: TIMING.swap }];
  for (const step of result.steps) {
    phases.push({ kind: "clear", step, duration: TIMING.clear });
    const longest = Math.max(0, ...step.drops.map((d) => d.toY - d.fromY), ...step.spawns.map((s) => s.toY - s.fromY));
    phases.push({ kind: "fall", step, duration: fallTime(longest) });
  }
  const settled = result.steps.at(-1)?.after ?? result.swapped;
  if (result.reshuffled) phases.push({ kind: "shuffle", from: settled, to: result.reshuffled, duration: TIMING.shuffle });
  return phases;
}

export class Playback {
  private readonly queue: Phase[] = [];
  private elapsed = 0;

  enqueue(phases: readonly Phase[]): void {
    this.queue.push(...phases);
  }

  /** Advances by dt, carrying leftover time into the next phase. */
  update(dt: number): void {
    if (!Number.isFinite(dt) || dt < 0) throw new RangeError("dt must be a non-negative number");
    this.elapsed += dt;
    while (this.queue[0] && this.elapsed >= this.queue[0].duration) {
      this.elapsed -= this.queue[0].duration;
      this.queue.shift();
    }
    if (this.queue.length === 0) this.elapsed = 0;
  }

  get busy(): boolean {
    return this.queue.length > 0;
  }

  /** The playing phase and seconds into it, or null when idle. */
  get current(): { readonly phase: Phase; readonly elapsed: number } | null {
    const phase = this.queue[0];
    return phase ? { phase, elapsed: this.elapsed } : null;
  }

  clear(): void {
    this.queue.length = 0;
    this.elapsed = 0;
  }
}
