/**
 * Browser shell: the only impure part of the game. It turns keyboard and
 * swipe input into actions, sends them through the reducer, and redraws.
 * All game state lives in one `History<GameState>` value.
 */
import { DEFAULT_CONFIG as config } from "./core/config";
import { createHistory, undoable, type History } from "./core/history";
import { createReducer, initialState, type Action, type GameState } from "./core/reducer";
import { spritesAt, TURN_DURATION } from "./view/animation";
import { directionForSwipe, intentForKey, type Intent } from "./view/input";
import { drawBoard, drawHud, drawOverlay, drawSprites, type Layout } from "./view/render";

const canvas = document.querySelector<HTMLCanvasElement>("#game");
const ctx = canvas?.getContext("2d");
if (!canvas || !ctx) throw new Error("canvas #game not found");

const HUD = 44;
const layout: Layout = { size: config.size, cell: 96, gap: 12, originX: 0, originY: HUD };
const span = config.size * (layout.cell + layout.gap) + layout.gap;
canvas.width = span;
canvas.height = HUD + span;

const UNDO_LIMIT = 64;
const reduce = undoable(createReducer(config), {
  limit: UNDO_LIMIT,
  resets: (a: Action) => a.type === "newGame",
});

const randomSeed = () => Math.floor(Math.random() * 1_000_000_000);
const seedParam = Number(new URLSearchParams(location.search).get("seed"));
const firstSeed = Number.isInteger(seedParam) && seedParam > 0 ? seedParam : randomSeed();

let history: History<GameState> = createHistory(initialState(config, firstSeed));
/** Seconds since the current state's turn began; drives the pure animation. */
let elapsed = 0;

function dispatch(intent: Intent): void {
  const seed = history.present.seed;
  const next =
    intent.kind === "move"
      ? reduce(history, { type: "move", direction: intent.direction })
      : intent.kind === "newGame"
        ? reduce(history, { type: "newGame", seed: randomSeed() })
        : intent.kind === "restart"
          ? reduce(history, { type: "newGame", seed })
          : reduce(history, { type: intent.kind });
  if (next === history) return;
  // Undo/redo jump straight to the settled board; real turns animate.
  elapsed = intent.kind === "undo" || intent.kind === "redo" ? TURN_DURATION : 0;
  history = next;
}

addEventListener("keydown", (e) => {
  const intent = intentForKey(e.code);
  if (!intent) return;
  e.preventDefault();
  dispatch(intent);
});

let swipeStart: { x: number; y: number } | null = null;
canvas.addEventListener("pointerdown", (e) => {
  swipeStart = { x: e.clientX, y: e.clientY };
  canvas.setPointerCapture(e.pointerId);
});
canvas.addEventListener("pointerup", (e) => {
  if (!swipeStart) return;
  const direction = directionForSwipe(e.clientX - swipeStart.x, e.clientY - swipeStart.y);
  swipeStart = null;
  if (direction) dispatch({ kind: "move", direction });
});

let last = performance.now();
function frame(now: number): void {
  // Clamp dt so a backgrounded tab doesn't skip straight past the animation logic.
  const dt = Math.min(0.1, (now - last) / 1000);
  last = now;
  elapsed += dt;
  const state = history.present;
  ctx!.fillStyle = "#0b0d12";
  ctx!.fillRect(0, 0, canvas!.width, canvas!.height);
  drawHud(ctx!, canvas!.width, state, history.past.length > 0);
  drawBoard(ctx!, layout);
  drawSprites(ctx!, layout, spritesAt(state.turn, state.grid, elapsed));
  if (elapsed >= TURN_DURATION) drawOverlay(ctx!, layout, state);
  requestAnimationFrame(frame);
}
requestAnimationFrame(frame);
