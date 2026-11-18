/**
 * Turn-level API: the only entry point the UI needs. A turn is
 * validate swap → resolve cascades → reshuffle if the board is dead.
 * The result carries everything required to animate the turn.
 */
import { swapCells, type Board, type Pos } from "./board";
import { resolveCascades, type CascadeStep } from "./cascade";
import { validateConfig, type Match3Config } from "./config";
import { generateBoard, shuffleBoard } from "./generate";
import { findRuns, uniqueCells } from "./match";
import { findMoves, hasMoves, validateSwap, type Move, type SwapRejection } from "./moves";
import type { Rng } from "./rng";

export interface GameState {
  readonly board: Board;
  readonly score: number;
  readonly movesMade: number;
  readonly longestChain: number;
  readonly shuffles: number;
}

export type TurnResult =
  | { readonly ok: false; readonly reason: SwapRejection }
  | {
      readonly ok: true;
      readonly state: GameState;
      /** Board right after the swap, before anything is cleared. */
      readonly swapped: Board;
      readonly steps: readonly CascadeStep[];
      readonly points: number;
      /** The reshuffled board if the turn left no legal moves, otherwise null. */
      readonly reshuffled: Board | null;
    };

export function newGame(cfg: Match3Config, rng: Rng): GameState {
  return { board: generateBoard(cfg, rng), score: 0, movesMade: 0, longestChain: 0, shuffles: 0 };
}

export function playTurn(state: GameState, a: Pos, b: Pos, rng: Rng, cfg: Match3Config): TurnResult {
  validateConfig(cfg);
  const rejection = validateSwap(state.board, a, b);
  if (rejection !== null) return { ok: false, reason: rejection };

  const swapped = swapCells(state.board, a, b);
  const resolution = resolveCascades(swapped, rng, cfg);
  const reshuffled = hasMoves(resolution.board) ? null : shuffleBoard(resolution.board, rng, cfg);

  return {
    ok: true,
    swapped,
    steps: resolution.steps,
    points: resolution.points,
    reshuffled,
    state: {
      board: reshuffled ?? resolution.board,
      score: state.score + resolution.points,
      movesMade: state.movesMade + 1,
      longestChain: Math.max(state.longestChain, resolution.steps.length),
      shuffles: state.shuffles + (reshuffled ? 1 : 0),
    },
  };
}

/**
 * The move that clears the most gems immediately (ties go to the first found,
 * scanning top-left to bottom-right), or null on a dead board. Cascades are
 * ignored: they depend on random refills the hint can't know.
 */
export function hint(state: GameState): Move | null {
  let best: Move | null = null;
  let bestCleared = 0;
  for (const move of findMoves(state.board)) {
    const swapped = swapCells(state.board, move.a, move.b);
    const cleared = uniqueCells(swapped, findRuns(swapped)).length;
    if (cleared > bestCleared) {
      best = move;
      bestCleared = cleared;
    }
  }
  return best;
}
