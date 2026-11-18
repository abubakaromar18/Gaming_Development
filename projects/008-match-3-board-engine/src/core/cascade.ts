/**
 * Cascade resolution: clear → gravity → refill, repeated until the board is
 * stable. Each pass is recorded as a CascadeStep so the renderer can animate
 * exactly what the simulation did, without re-deriving any rules.
 */
import { cellAt, indexOf, type Board, type Cell, type Gem, type Pos } from "./board";
import type { Match3Config } from "./config";
import { findRuns, uniqueCells, type Run } from "./match";
import { randomInt, type Rng } from "./rng";

/** An existing gem that fell from `fromY` to `toY` in column `x`. */
export interface Drop {
  readonly x: number;
  readonly fromY: number;
  readonly toY: number;
  readonly gem: Gem;
}

/** A new gem that entered from above. `fromY` is negative (off the top edge). */
export interface Spawn {
  readonly x: number;
  readonly fromY: number;
  readonly toY: number;
  readonly gem: Gem;
}

export interface CascadeStep {
  /** 1 for the match the player made, 2 for the first cascade, and so on. */
  readonly chain: number;
  readonly before: Board;
  readonly runs: readonly Run[];
  readonly cleared: readonly Pos[];
  readonly drops: readonly Drop[];
  readonly spawns: readonly Spawn[];
  readonly after: Board;
  readonly points: number;
}

export interface Resolution {
  readonly board: Board;
  readonly steps: readonly CascadeStep[];
  readonly points: number;
}

export function clearCells(board: Board, cells: readonly Pos[]): Board {
  const next = board.cells.slice();
  for (const p of cells) next[indexOf(board, p)] = null;
  return { ...board, cells: next };
}

/**
 * Compacts every column toward the bottom, keeping gem order. Walking each
 * column bottom-up with a write cursor makes this a single O(height) pass.
 */
export function applyGravity(board: Board): { board: Board; drops: Drop[] } {
  const cells: Cell[] = board.cells.slice();
  const drops: Drop[] = [];
  for (let x = 0; x < board.width; x++) {
    let write = board.height - 1;
    for (let y = board.height - 1; y >= 0; y--) {
      const gem = cellAt(board, { x, y });
      if (gem === null) continue;
      if (write !== y) {
        cells[write * board.width + x] = gem;
        cells[y * board.width + x] = null;
        drops.push({ x, fromY: y, toY: write, gem });
      }
      write--;
    }
  }
  return { board: { ...board, cells }, drops };
}

/**
 * Fills holes with random gems. Assumes gravity already ran, so holes are
 * at the top of each column. New gems are stacked above the board in the
 * same order so they appear to fall in as one column.
 */
export function refill(board: Board, rng: Rng, colors: number): { board: Board; spawns: Spawn[] } {
  const cells: Cell[] = board.cells.slice();
  const spawns: Spawn[] = [];
  for (let x = 0; x < board.width; x++) {
    let holes = 0;
    while (holes < board.height && cellAt(board, { x, y: holes }) === null) holes++;
    for (let y = 0; y < holes; y++) {
      const gem = randomInt(rng, colors);
      cells[y * board.width + x] = gem;
      spawns.push({ x, fromY: y - holes, toY: y, gem });
    }
  }
  return { board: { ...board, cells }, spawns };
}

/** Points for one clearing pass, multiplied by how deep the chain is. */
export function scoreRuns(runs: readonly Run[], chain: number, cfg: Match3Config): number {
  let base = 0;
  for (const run of runs) {
    base += run.cells.length * cfg.pointsPerGem + Math.max(0, run.cells.length - 3) * cfg.longRunBonus;
  }
  return base * chain;
}

/** Repeats clear → gravity → refill until no runs remain. */
export function resolveCascades(board: Board, rng: Rng, cfg: Match3Config): Resolution {
  const steps: CascadeStep[] = [];
  let current = board;
  let points = 0;
  for (let chain = 1; ; chain++) {
    const runs = findRuns(current);
    if (runs.length === 0) return { board: current, steps, points };
    if (chain > cfg.maxCascades) {
      throw new Error(`cascade did not settle after ${cfg.maxCascades} passes`);
    }
    const cleared = uniqueCells(current, runs);
    const fallen = applyGravity(clearCells(current, cleared));
    const filled = refill(fallen.board, rng, cfg.colors);
    const stepPoints = scoreRuns(runs, chain, cfg);
    steps.push({ chain, before: current, runs, cleared, drops: fallen.drops, spawns: filled.spawns, after: filled.board, points: stepPoints });
    points += stepPoints;
    current = filled.board;
  }
}
