/**
 * The heart of 2048: sliding one line toward its leading edge.
 *
 * Rules, applied leading-edge first:
 *  - gaps close up;
 *  - two equal neighbors (after gaps close) merge into one tile of double value;
 *  - a tile produced by a merge cannot merge again in the same move,
 *    so [2, 2, 4] → [4, 4], not [8], and [2, 2, 2, 2] → [4, 4].
 *
 * The function is generic over identity: it reports which source tiles end up
 * in which slot and leaves id allocation to the caller.
 */
import type { Cell, Tile } from "./grid";

export interface Slot {
  /** One tile that just moved (or stayed), or the two tiles that merged here. */
  readonly sources: readonly [Tile] | readonly [Tile, Tile];
}

export interface LineResult {
  /** Filled slots from the leading edge; the rest of the line is empty. */
  readonly slots: readonly Slot[];
  /** False when the line is already packed with no possible merge. */
  readonly changed: boolean;
}

export function slideLine(line: readonly Cell[]): LineResult {
  const packed = line.filter((c): c is Tile => c !== null);
  const slots: Slot[] = [];
  for (let i = 0; i < packed.length; i++) {
    const a = packed[i]!;
    const b = packed[i + 1];
    if (b && b.value === a.value) {
      slots.push({ sources: [a, b] });
      i++; // b is consumed, and the merged tile is never revisited
    } else {
      slots.push({ sources: [a] });
    }
  }
  // Changed iff any slot holds a merge or a tile ends up in a different index.
  const changed = slots.some((s, i) => s.sources.length === 2 || line[i] !== s.sources[0]);
  return { slots, changed };
}
