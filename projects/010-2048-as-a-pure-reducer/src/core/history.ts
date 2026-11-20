/**
 * Undo as a higher-order reducer. Because the game reducer never mutates and
 * returns the same reference for no-ops, history is just a list of previous
 * states: no command objects, no inverse operations, no deep copies.
 *
 * Note the RNG travels with each snapshot. Undoing and repeating a move spawns
 * the same tile in the same place, so undo can't be used to reroll spawns.
 */
import type { Reducer } from "./reducer";

export interface History<S> {
  readonly past: readonly S[];
  readonly present: S;
  readonly future: readonly S[];
}

export type HistoryAction = { readonly type: "undo" } | { readonly type: "redo" };

export interface UndoOptions<A> {
  /** Maximum stored past states. Older ones are dropped. */
  readonly limit: number;
  /** Actions that start a fresh history (e.g. a new game). */
  readonly resets?: (action: A) => boolean;
}

export function createHistory<S>(present: S): History<S> {
  return { past: [], present, future: [] };
}

export function undoable<S, A extends { readonly type: string }>(
  reducer: Reducer<S, A>,
  { limit, resets = () => false }: UndoOptions<A>,
): Reducer<History<S>, A | HistoryAction> {
  if (!Number.isInteger(limit) || limit < 1) throw new RangeError(`limit must be a positive integer, got ${limit}`);

  return (history, action) => {
    if (action.type === "undo") {
      const prev = history.past.at(-1);
      if (prev === undefined) return history;
      return { past: history.past.slice(0, -1), present: prev, future: [history.present, ...history.future] };
    }
    if (action.type === "redo") {
      const [next, ...rest] = history.future;
      if (next === undefined) return history;
      return { past: [...history.past, history.present], present: next, future: rest };
    }

    const gameAction = action as A;
    const next = reducer(history.present, gameAction);
    if (resets(gameAction)) return createHistory(next);
    if (next === history.present) return history;
    return { past: [...history.past, history.present].slice(-limit), present: next, future: [] };
  };
}
