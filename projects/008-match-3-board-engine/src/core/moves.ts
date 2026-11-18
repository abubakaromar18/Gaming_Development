/**
 * Swap validation and move search. A move is legal only if swapping two
 * orthogonally adjacent gems creates at least one run through either of them.
 */
import { cellAt, inBounds, swapCells, type Board, type Pos } from "./board";
import { isMatchedAt } from "./match";

export interface Move {
  readonly a: Pos;
  readonly b: Pos;
}

export type SwapRejection = "out-of-bounds" | "not-adjacent" | "empty-cell" | "no-match";

export function areAdjacent(a: Pos, b: Pos): boolean {
  return Math.abs(a.x - b.x) + Math.abs(a.y - b.y) === 1;
}

/** Returns why a swap is illegal, or null when it is a valid move. */
export function validateSwap(board: Board, a: Pos, b: Pos): SwapRejection | null {
  if (!inBounds(board, a) || !inBounds(board, b)) return "out-of-bounds";
  if (!areAdjacent(a, b)) return "not-adjacent";
  const ga = cellAt(board, a);
  const gb = cellAt(board, b);
  if (ga === null || gb === null) return "empty-cell";
  // Swapping two equal gems never changes the board, so it can't match.
  if (ga === gb) return "no-match";
  const swapped = swapCells(board, a, b);
  return isMatchedAt(swapped, a) || isMatchedAt(swapped, b) ? null : "no-match";
}

/**
 * Every legal move, scanning each cell's right and down neighbor so each
 * pair is considered once. O(cells) swaps, each checked in O(width + height).
 */
export function findMoves(board: Board): Move[] {
  const moves: Move[] = [];
  for (let y = 0; y < board.height; y++) {
    for (let x = 0; x < board.width; x++) {
      const a = { x, y };
      for (const b of [{ x: x + 1, y }, { x, y: y + 1 }]) {
        if (inBounds(board, b) && validateSwap(board, a, b) === null) moves.push({ a, b });
      }
    }
  }
  return moves;
}

/** Early-exit version of findMoves for the no-moves check after every turn. */
export function hasMoves(board: Board): boolean {
  for (let y = 0; y < board.height; y++) {
    for (let x = 0; x < board.width; x++) {
      const a = { x, y };
      if (x + 1 < board.width && validateSwap(board, a, { x: x + 1, y }) === null) return true;
      if (y + 1 < board.height && validateSwap(board, a, { x, y: y + 1 }) === null) return true;
    }
  }
  return false;
}
