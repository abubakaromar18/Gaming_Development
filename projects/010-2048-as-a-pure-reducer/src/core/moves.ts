/**
 * Applies a move to a whole grid by cutting it into lines.
 *
 * Instead of rotating the grid four ways, `lines(size, dir)` lists the cell
 * positions of each row or column ordered from the edge tiles slide toward.
 * Every direction then reuses the same `slideLine`, and no grid is copied just
 * to be rotated back.
 */
import { cellAt, emptyCells, type Cell, type Grid, type Pos } from "./grid";
import { slideLine } from "./slide";

export type Direction = "up" | "down" | "left" | "right";
export const DIRECTIONS: readonly Direction[] = ["up", "down", "left", "right"];

/** Where a pre-move tile went. Unmoved tiles appear with from === to. */
export interface SlideEvent {
  readonly id: number;
  readonly value: number;
  readonly from: Pos;
  readonly to: Pos;
}

export interface MergeEvent {
  /** Id of the new, doubled tile. */
  readonly id: number;
  readonly value: number;
  readonly at: Pos;
  readonly from: readonly [number, number];
}

export interface MoveResult {
  readonly grid: Grid;
  readonly moved: boolean;
  readonly gained: number;
  readonly nextId: number;
  readonly slides: readonly SlideEvent[];
  readonly merges: readonly MergeEvent[];
}

export function lines(size: number, dir: Direction): Pos[][] {
  const idx = Array.from({ length: size }, (_, i) => i);
  const rev = [...idx].reverse();
  switch (dir) {
    case "left":
      return idx.map((y) => idx.map((x) => ({ x, y })));
    case "right":
      return idx.map((y) => rev.map((x) => ({ x, y })));
    case "up":
      return idx.map((x) => idx.map((y) => ({ x, y })));
    case "down":
      return idx.map((x) => rev.map((y) => ({ x, y })));
  }
}

/**
 * Slides every line. New merged tiles get ids starting at `nextId`, so the
 * caller (the reducer) owns id allocation and the function stays pure.
 */
export function applyMove(grid: Grid, dir: Direction, nextId: number): MoveResult {
  const size = grid.length;
  const out: Cell[][] = Array.from({ length: size }, () => Array<Cell>(size).fill(null));
  const slides: SlideEvent[] = [];
  const merges: MergeEvent[] = [];
  let moved = false;
  let gained = 0;
  let id = nextId;

  for (const line of lines(size, dir)) {
    const cells = line.map((p) => cellAt(grid, p));
    const { slots, changed } = slideLine(cells);
    moved ||= changed;
    slots.forEach((slot, i) => {
      const to = line[i]!;
      for (const src of slot.sources) {
        slides.push({ id: src.id, value: src.value, from: line[cells.indexOf(src)]!, to });
      }
      if (slot.sources.length === 2) {
        const value = slot.sources[0].value * 2;
        const merged = { id: id++, value };
        merges.push({ id: merged.id, value, at: to, from: [slot.sources[0].id, slot.sources[1].id] });
        gained += value;
        out[to.y]![to.x] = merged;
      } else {
        out[to.y]![to.x] = slot.sources[0];
      }
    });
  }

  if (!moved) return { grid, moved, gained: 0, nextId, slides: [], merges: [] };
  return { grid: out, moved, gained, nextId: id, slides, merges };
}

/** Cheap test: would this direction change anything? No grid is built. */
export function canMove(grid: Grid, dir: Direction): boolean {
  return lines(grid.length, dir).some((line) => slideLine(line.map((p) => cellAt(grid, p))).changed);
}

/** The game is over when the board is full and no two orthogonal neighbors match. */
export function hasAnyMove(grid: Grid): boolean {
  if (emptyCells(grid).length > 0) return true;
  return grid.some((row, y) =>
    row.some((c, x) => c !== null && (c.value === row[x + 1]?.value || c.value === grid[y + 1]?.[x]?.value)),
  );
}
