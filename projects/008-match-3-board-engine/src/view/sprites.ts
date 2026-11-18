/**
 * Converts "board + current animation phase" into a flat list of sprites in
 * cell coordinates. The canvas renderer just draws the list, so all the
 * animation math lives here where it can be reasoned about without a canvas.
 */
import { cellAt, samePos, type Board, type Gem, type Pos } from "../core/board";
import { fallDistance, type Phase } from "./playback";

export interface Sprite {
  /** Column, possibly fractional mid-swap. */
  readonly x: number;
  /** Row, possibly fractional or negative mid-fall. */
  readonly y: number;
  readonly gem: Gem;
  readonly scale: number;
  readonly alpha: number;
}

const easeInOut = (t: number): number => (t < 0.5 ? 2 * t * t : 1 - (-2 * t + 2) ** 2 / 2);
const clamp01 = (t: number): number => Math.min(1, Math.max(0, t));

export function boardSprites(board: Board, skip: (p: Pos) => boolean = () => false, alpha = 1): Sprite[] {
  const out: Sprite[] = [];
  for (let y = 0; y < board.height; y++) {
    for (let x = 0; x < board.width; x++) {
      const gem = cellAt(board, { x, y });
      if (gem !== null && !skip({ x, y })) out.push({ x, y, gem, scale: 1, alpha });
    }
  }
  return out;
}

export function phaseSprites(phase: Phase, elapsed: number): Sprite[] {
  const t = clamp01(elapsed / phase.duration);
  switch (phase.kind) {
    case "swap": {
      const { board, a, b } = phase;
      // A bounce runs forward for the first half and back for the second.
      const p = easeInOut(phase.bounce ? 1 - Math.abs(2 * t - 1) : t);
      const sprites = boardSprites(board, (q) => samePos(q, a) || samePos(q, b));
      const moving = (from: Pos, to: Pos): Sprite | null => {
        const gem = cellAt(board, from);
        return gem === null ? null : { x: from.x + (to.x - from.x) * p, y: from.y + (to.y - from.y) * p, gem, scale: 1, alpha: 1 };
      };
      for (const s of [moving(a, b), moving(b, a)]) if (s) sprites.push(s);
      return sprites;
    }
    case "clear": {
      const cleared = phase.step.cleared;
      const isCleared = (q: Pos) => cleared.some((c) => samePos(c, q));
      const sprites = boardSprites(phase.step.before, isCleared);
      for (const p of cleared) {
        const gem = cellAt(phase.step.before, p);
        if (gem !== null) sprites.push({ ...p, gem, scale: 1 + 0.25 * t - 1.25 * t * t, alpha: 1 - t });
      }
      return sprites;
    }
    case "fall": {
      const { after, drops, spawns } = phase.step;
      const startRow = new Map<number, number>();
      for (const m of [...drops, ...spawns]) startRow.set(m.toY * after.width + m.x, m.fromY);
      return boardSprites(after).map((s) => {
        const from = startRow.get(s.y * after.width + s.x);
        if (from === undefined) return s;
        return { ...s, y: from + fallDistance(elapsed, s.y - from) };
      });
    }
    case "shuffle": {
      const fadingOut = t < 0.5;
      const k = fadingOut ? 1 - 2 * t : 2 * t - 1;
      return boardSprites(fadingOut ? phase.from : phase.to).map((s) => ({ ...s, scale: 0.4 + 0.6 * k, alpha: k }));
    }
  }
}
