/**
 * Immutable grid of tiles. Each tile has a stable `id` so the view can follow
 * a tile across moves (slide it, pop it when it merges) without diffing values.
 */
export interface Tile {
  readonly id: number;
  readonly value: number;
}

export type Cell = Tile | null;
/** Row-major: grid[y][x]. */
export type Grid = readonly (readonly Cell[])[];

export interface Pos {
  readonly x: number;
  readonly y: number;
}

export function emptyGrid(size: number): Grid {
  return Array.from({ length: size }, () => Array<Cell>(size).fill(null));
}

export function cellAt(grid: Grid, p: Pos): Cell {
  return grid[p.y]?.[p.x] ?? null;
}

export function withCell(grid: Grid, p: Pos, cell: Cell): Grid {
  return grid.map((row, y) => (y === p.y ? row.map((c, x) => (x === p.x ? cell : c)) : row));
}

export function emptyCells(grid: Grid): Pos[] {
  const out: Pos[] = [];
  grid.forEach((row, y) => row.forEach((c, x) => c === null && out.push({ x, y })));
  return out;
}

export function tiles(grid: Grid): { tile: Tile; at: Pos }[] {
  const out: { tile: Tile; at: Pos }[] = [];
  grid.forEach((row, y) => row.forEach((c, x) => c && out.push({ tile: c, at: { x, y } })));
  return out;
}

export function maxValue(grid: Grid): number {
  return Math.max(0, ...tiles(grid).map((t) => t.tile.value));
}

/**
 * Test/debug fixture: `fromRows(["2 . . 2", ". 4 . ."])`. Tokens are tile
 * values or `.` for empty. Ids are assigned in reading order from `firstId`.
 */
export function fromRows(rows: readonly string[], firstId = 1): Grid {
  let id = firstId;
  const grid = rows.map((row, y) =>
    row
      .trim()
      .split(/\s+/)
      .map((tok, x): Cell => {
        if (tok === ".") return null;
        const value = Number(tok);
        if (!Number.isInteger(value) || value < 2 || (value & (value - 1)) !== 0) {
          throw new SyntaxError(`bad tile "${tok}" at (${x}, ${y})`);
        }
        return { id: id++, value };
      }),
  );
  if (grid.some((r) => r.length !== grid.length)) throw new SyntaxError("fixture must be square");
  return grid;
}

/** Inverse of fromRows (values only), for readable assertions. */
export function toRows(grid: Grid): string[] {
  return grid.map((row) => row.map((c) => (c ? String(c.value) : ".")).join(" "));
}
