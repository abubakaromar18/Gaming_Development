/**
 * Browser shell: wires pointer/keyboard input, requestAnimationFrame, and the
 * canvas to the pure engine in ./core. A turn is resolved by the engine in one
 * call; ./view/playback then animates the recorded steps.
 */
import { inBounds, type Pos } from "./core/board";
import { DEFAULT_CONFIG as cfg } from "./core/config";
import { hint, newGame, playTurn, type GameState } from "./core/game";
import { areAdjacent, type Move } from "./core/moves";
import { mulberry32, type Rng } from "./core/rng";
import { phasesForTurn, Playback } from "./view/playback";
import { drawBackground, drawCellOutline, drawSprites, type Layout } from "./view/render";
import { boardSprites, phaseSprites } from "./view/sprites";

const canvas = document.querySelector<HTMLCanvasElement>("#game");
const ctx = canvas?.getContext("2d");
if (!canvas || !ctx) throw new Error("canvas #game not found");

const HUD = 56;
const HINT_DELAY_SECONDS = 6;
const layout: Layout = { cell: 64, originX: 0, originY: HUD, cols: cfg.width, rows: cfg.height };
canvas.width = cfg.width * layout.cell;
canvas.height = HUD + cfg.height * layout.cell;

const seedParam = Number(new URLSearchParams(location.search).get("seed"));
let seed = Number.isInteger(seedParam) && seedParam > 0 ? seedParam : Date.now() % 1_000_000;
let rng: Rng = mulberry32(seed);
let state: GameState = newGame(cfg, rng);
const playback = new Playback();
let selected: Pos | null = null;
let dragFrom: Pos | null = null;
let idleSeconds = 0;
let shownHint: Move | null = null;
let lastPoints: { points: number; chain: number } | null = null;

function restart(nextSeed: number): void {
  seed = nextSeed;
  rng = mulberry32(seed);
  state = newGame(cfg, rng);
  playback.clear();
  selected = null;
  shownHint = null;
  lastPoints = null;
  idleSeconds = 0;
}

function cellFromEvent(e: PointerEvent): Pos | null {
  const rect = canvas!.getBoundingClientRect();
  const sx = canvas!.width / rect.width;
  const sy = canvas!.height / rect.height;
  const p = {
    x: Math.floor(((e.clientX - rect.left) * sx - layout.originX) / layout.cell),
    y: Math.floor(((e.clientY - rect.top) * sy - layout.originY) / layout.cell),
  };
  return inBounds(state.board, p) ? p : null;
}

function attempt(a: Pos, b: Pos): void {
  const before = state.board;
  const result = playTurn(state, a, b, rng, cfg);
  playback.enqueue(phasesForTurn(before, a, b, result));
  if (result.ok) {
    state = result.state;
    lastPoints = { points: result.points, chain: result.steps.length };
  }
  selected = null;
  shownHint = null;
  idleSeconds = 0;
}

canvas.addEventListener("pointerdown", (e) => {
  if (playback.busy) return;
  const p = cellFromEvent(e);
  if (!p) return;
  canvas.setPointerCapture(e.pointerId);
  dragFrom = p;
  if (selected && areAdjacent(selected, p)) {
    attempt(selected, p);
    dragFrom = null;
  } else {
    selected = selected && selected.x === p.x && selected.y === p.y ? null : p;
  }
});

canvas.addEventListener("pointermove", (e) => {
  if (!dragFrom || playback.busy) return;
  const p = cellFromEvent(e);
  // Dragging into a neighbor cell swaps immediately, like most mobile match-3s.
  if (p && areAdjacent(dragFrom, p)) {
    attempt(dragFrom, p);
    dragFrom = null;
  }
});

canvas.addEventListener("pointerup", () => {
  dragFrom = null;
});

addEventListener("keydown", (e) => {
  if (e.code === "KeyH" && !playback.busy) shownHint = hint(state);
  if (e.code === "KeyN") restart(Date.now() % 1_000_000);
  if (e.code === "KeyR") restart(seed);
});

let last = performance.now();
function frame(now: number): void {
  // rAF timestamps can precede the first performance.now() read; clamp both ends.
  const dt = Math.min(0.1, Math.max(0, (now - last) / 1000));
  last = now;
  playback.update(dt);
  if (!playback.busy) {
    idleSeconds += dt;
    if (idleSeconds > HINT_DELAY_SECONDS && !shownHint) shownHint = hint(state);
  }
  draw(now / 1000);
  requestAnimationFrame(frame);
}
requestAnimationFrame(frame);

function draw(seconds: number): void {
  const c = ctx!;
  c.fillStyle = "#0b0d12";
  c.fillRect(0, 0, canvas!.width, canvas!.height);
  drawBackground(c, layout);

  const current = playback.current;
  drawSprites(c, layout, current ? phaseSprites(current.phase, current.elapsed) : boardSprites(state.board));

  if (!current && selected) drawCellOutline(c, layout, selected, "#ffffff", 0.9);
  if (!current && shownHint) {
    const pulse = 0.55 + 0.4 * Math.sin(seconds * 6);
    drawCellOutline(c, layout, shownHint.a, "#ffd166", pulse);
    drawCellOutline(c, layout, shownHint.b, "#ffd166", pulse);
  }

  c.fillStyle = "#e8ecf4";
  c.font = "bold 22px ui-monospace, monospace";
  c.textAlign = "left";
  c.fillText(`${state.score}`, 12, 34);
  c.font = "13px ui-monospace, monospace";
  c.fillStyle = "#7f8aa3";
  c.fillText(`moves ${state.movesMade} · best chain ${state.longestChain} · shuffles ${state.shuffles}`, 12, 50);
  c.textAlign = "right";
  c.fillText(`seed ${seed}`, canvas!.width - 12, 50);

  const chain = current && current.phase.kind !== "swap" && current.phase.kind !== "shuffle" ? current.phase.step.chain : 0;
  c.font = "bold 18px ui-monospace, monospace";
  if (chain > 1) {
    c.fillStyle = "#ffd166";
    c.fillText(`chain ×${chain}`, canvas!.width - 12, 28);
  } else if (current?.phase.kind === "shuffle") {
    c.fillStyle = "#4cc9f0";
    c.fillText("no moves — shuffling", canvas!.width - 12, 28);
  } else if (lastPoints && !current) {
    c.fillStyle = "#06d6a0";
    c.fillText(`+${lastPoints.points}`, canvas!.width - 12, 28);
  }
}
