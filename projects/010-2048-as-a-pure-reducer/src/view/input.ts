/**
 * Input → intent mapping, kept pure so it can be tested without a DOM.
 * The shell feeds it key codes and pointer deltas.
 */
import type { Direction } from "../core/moves";

export type Intent = { readonly kind: "move"; readonly direction: Direction } | { readonly kind: "undo" } | { readonly kind: "redo" } | { readonly kind: "newGame" } | { readonly kind: "restart" } | { readonly kind: "keepPlaying" };

const KEYS: Readonly<Record<string, Intent>> = {
  ArrowUp: { kind: "move", direction: "up" },
  ArrowDown: { kind: "move", direction: "down" },
  ArrowLeft: { kind: "move", direction: "left" },
  ArrowRight: { kind: "move", direction: "right" },
  KeyW: { kind: "move", direction: "up" },
  KeyS: { kind: "move", direction: "down" },
  KeyA: { kind: "move", direction: "left" },
  KeyD: { kind: "move", direction: "right" },
  KeyU: { kind: "undo" },
  KeyY: { kind: "redo" },
  KeyN: { kind: "newGame" },
  KeyR: { kind: "restart" },
  KeyC: { kind: "keepPlaying" },
};

export function intentForKey(code: string): Intent | null {
  return KEYS[code] ?? null;
}

/** Minimum swipe length in CSS pixels before a drag counts as a move. */
export const SWIPE_THRESHOLD = 24;

/** A swipe picks the dominant axis; short or perfectly diagonal drags are ignored. */
export function directionForSwipe(dx: number, dy: number, threshold = SWIPE_THRESHOLD): Direction | null {
  const ax = Math.abs(dx);
  const ay = Math.abs(dy);
  if (Math.max(ax, ay) < threshold || ax === ay) return null;
  if (ax > ay) return dx > 0 ? "right" : "left";
  return dy > 0 ? "down" : "up";
}
