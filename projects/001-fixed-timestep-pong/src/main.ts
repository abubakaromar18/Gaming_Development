/**
 * Browser shell: wires the pure core to requestAnimationFrame, the keyboard,
 * and a 2D canvas. All game rules live in ./core.
 */
import { cpuIntent } from "./core/ai";
import { interpolate, type RenderView } from "./core/interpolate";
import { FixedTimestepLoop } from "./core/loop";
import { createState, DEFAULT_CONFIG as cfg, leftPaddleX, rightPaddleX, step, type PongState } from "./core/pong";
import { mulberry32 } from "./core/rng";

const canvas = document.querySelector<HTMLCanvasElement>("#game");
const ctx = canvas?.getContext("2d");
if (!canvas || !ctx) throw new Error("canvas #game not found");

const RATES = [10, 20, 60, 120] as const;
const keys = new Set<string>();
let serveQueued = false;
let interpolationOn = true;

addEventListener("keydown", (e) => {
  keys.add(e.code);
  if (e.code === "Space") serveQueued = true;
  if (e.code === "KeyI") interpolationOn = !interpolationOn;
  const rateIndex = ["Digit1", "Digit2", "Digit3", "Digit4"].indexOf(e.code);
  const hz = RATES[rateIndex];
  if (hz !== undefined) loop.setHz(hz);
});
addEventListener("keyup", (e) => keys.delete(e.code));

const rng = mulberry32(Date.now());
let prev: PongState = createState(cfg);
let curr: PongState = prev;

const loop = new FixedTimestepLoop({ hz: 60 }, (dt) => {
  const up = keys.has("KeyW") || keys.has("ArrowUp");
  const down = keys.has("KeyS") || keys.has("ArrowDown");
  prev = curr;
  curr = step(curr, { left: up === down ? 0 : up ? -1 : 1, right: cpuIntent(curr, cfg), serve: serveQueued }, dt, rng, cfg);
  serveQueued = false; // a serve press is consumed by exactly one step
});

let last = performance.now();
let lastSteps = 0;
function frame(now: number): void {
  const { steps, alpha } = loop.advance((now - last) / 1000);
  last = now;
  if (steps > 0) lastSteps = steps;
  draw(interpolationOn ? interpolate(prev, curr, alpha) : interpolate(curr, curr, 0), curr);
  requestAnimationFrame(frame);
}
requestAnimationFrame(frame);

function draw(view: RenderView, state: PongState): void {
  const c = ctx!;
  c.fillStyle = "#11141b";
  c.fillRect(0, 0, cfg.width, cfg.height);

  c.fillStyle = "#2a3040";
  for (let y = 0; y < cfg.height; y += 24) c.fillRect(cfg.width / 2 - 2, y, 4, 12);

  c.fillStyle = "#e8ecf4";
  c.fillRect(leftPaddleX(cfg), view.leftY, cfg.paddleWidth, cfg.paddleHeight);
  c.fillRect(rightPaddleX(cfg), view.rightY, cfg.paddleWidth, cfg.paddleHeight);
  c.fillStyle = "#ffd166";
  c.fillRect(view.ball.x, view.ball.y, cfg.ballSize, cfg.ballSize);

  c.fillStyle = "#e8ecf4";
  c.font = "48px ui-monospace, monospace";
  c.textAlign = "center";
  c.fillText(String(state.score.left), cfg.width / 2 - 60, 60);
  c.fillText(String(state.score.right), cfg.width / 2 + 60, 60);

  c.font = "14px ui-monospace, monospace";
  if (state.phase === "serving") c.fillText("press SPACE to serve", cfg.width / 2, cfg.height - 30);
  if (state.phase === "gameOver") c.fillText(`${state.winner === "left" ? "YOU WIN" : "CPU WINS"} — SPACE to restart`, cfg.width / 2, cfg.height / 2 - 40);

  c.textAlign = "left";
  c.fillStyle = "#7f8aa3";
  const hz = Math.round(1 / loop.stepSeconds);
  c.fillText(`sim ${hz} Hz · steps/frame ${lastSteps} · interpolation ${interpolationOn ? "on" : "off"}`, 12, cfg.height - 12);
}
