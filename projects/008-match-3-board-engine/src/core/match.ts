/**
 * Match detection. A run is three or more equal gems in a straight line.
 * Runs are found with a single linear scan per row and per column, so the
 * cost is O(width × height) regardless of how many matches exist.
 */
import { cellAt, type Board, type Gem, type Pos } from "./board";

export const MIN_RUN = 3;

export interface Run {
  readonly gem: Gem;
  readonly orientation: "horizontal" | "vertical";
  readonly cells: readonly Pos[];
}

/** All maximal horizontal and vertical runs on the board. */
export function findRuns(board: Board): Run[] {
  const runs: Run[] = [];
  scanLines(board.height, board.width, (line, i) => ({ x: i, y: line }), "horizontal");
  scanLines(board.width, board.height, (line, i) => ({ x: line, y: i }), "vertical");
  return runs;

  function scanLines(lines: number, length: number, at: (line: number, i: number) => Pos, orientation: Run["orientation"]): void {
    for (let line = 0; line < lines; line++) {
      let start = 0;
      for (let i = 1; i <= length; i++) {
        const startGem = cellAt(board, at(line, start));
        // Close the current streak at the end of the line or when the gem changes.
        if (i === length || cellAt(board, at(line, i)) !== startGem) {
          if (startGem !== null && i - start >= MIN_RUN) {
            const cells: Pos[] = [];
            for (let k = start; k < i; k++) cells.push(at(line, k));
            runs.push({ gem: startGem, orientation, cells });
          }
          start = i;
        }
      }
    }
  }
}

export function hasAnyMatch(board: Board): boolean {
  return findRuns(board).length > 0;
}

/**
 * True if the gem at `p` is part of a run. Only looks along the row and the
 * column through `p`, which makes move searches cheap.
 */
export function isMatchedAt(board: Board, p: Pos): boolean {
  const gem = cellAt(board, p);
  if (gem === null) return false;
  const streak = (dx: number, dy: number): number => {
    let n = 0;
    for (let x = p.x + dx, y = p.y + dy; cellAt(board, { x, y }) === gem; x += dx, y += dy) n++;
    return n;
  };
  return 1 + streak(-1, 0) + streak(1, 0) >= MIN_RUN || 1 + streak(0, -1) + streak(0, 1) >= MIN_RUN;
}

/** Union of all run cells. L and T shapes share a corner; it appears once. */
export function uniqueCells(board: Board, runs: readonly Run[]): Pos[] {
  const seen = new Set<number>();
  const out: Pos[] = [];
  for (const run of runs) {
    for (const p of run.cells) {
      const key = p.y * board.width + p.x;
      if (!seen.has(key)) {
        seen.add(key);
        out.push(p);
      }
    }
  }
  return out;
}
