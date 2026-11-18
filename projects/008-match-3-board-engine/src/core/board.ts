/**
 * Immutable grid of gems. Row 0 is the top; gravity pulls toward higher y.
 * Cells are stored row-major in a flat array, and `null` marks a hole that
 * gravity and refill will fill.
 */

/** A gem is its color index, 0 … colors-1. */
export type Gem = number;
export type Cell = Gem | null;

export interface Pos {
  readonly x: number;
  readonly y: number;
}

export interface Board {
  readonly width: number;
  readonly height: number;
  readonly cells: readonly Cell[];
}

export function createEmpty(width: number, height: number): Board {
  return { width, height, cells: new Array<Cell>(width * height).fill(null) };
}

export function inBounds(board: Board, p: Pos): boolean {
  return Number.isInteger(p.x) && Number.isInteger(p.y) && p.x >= 0 && p.y >= 0 && p.x < board.width && p.y < board.height;
}

export function indexOf(board: Board, p: Pos): number {
  return p.y * board.width + p.x;
}

/** Returns the cell at `p`, or null when `p` is off the board. */
export function cellAt(board: Board, p: Pos): Cell {
  return inBounds(board, p) ? (board.cells[indexOf(board, p)] ?? null) : null;
}

export function swapCells(board: Board, a: Pos, b: Pos): Board {
  const cells = board.cells.slice();
  const ia = indexOf(board, a);
  const ib = indexOf(board, b);
  [cells[ia], cells[ib]] = [cells[ib] ?? null, cells[ia] ?? null];
  return { ...board, cells };
}

export function samePos(a: Pos, b: Pos): boolean {
  return a.x === b.x && a.y === b.y;
}

const LETTERS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";

/**
 * Parses a board from text rows: letters A–Z are gems 0–25 and `.` is empty.
 * Used by tests and handy for reproducing bug reports.
 */
export function fromRows(rows: readonly string[]): Board {
  const height = rows.length;
  const width = rows[0]?.length ?? 0;
  if (height === 0 || width === 0) throw new RangeError("board text must not be empty");
  const cells: Cell[] = [];
  rows.forEach((row, y) => {
    if (row.length !== width) throw new RangeError(`row ${y} has length ${row.length}, expected ${width}`);
    for (const ch of row) {
      if (ch === ".") cells.push(null);
      else {
        const gem = LETTERS.indexOf(ch);
        if (gem < 0) throw new RangeError(`unexpected character '${ch}' in row ${y}`);
        cells.push(gem);
      }
    }
  });
  return { width, height, cells };
}

export function toRows(board: Board): string[] {
  const rows: string[] = [];
  for (let y = 0; y < board.height; y++) {
    let row = "";
    for (let x = 0; x < board.width; x++) {
      const c = cellAt(board, { x, y });
      row += c === null ? "." : LETTERS[c];
    }
    rows.push(row);
  }
  return rows;
}
