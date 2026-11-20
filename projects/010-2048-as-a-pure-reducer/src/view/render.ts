/**
 * Canvas drawing. Tile colors are generated from log2(value), so any board
 * size or target works without a hand-made palette.
 */
import type { GameState } from "../core/reducer";
import type { Sprite } from "./animation";

export interface Layout {
  readonly size: number;
  readonly cell: number;
  readonly gap: number;
  readonly originX: number;
  readonly originY: number;
}

export function tileColors(value: number): { fill: string; text: string } {
  const level = Math.log2(value); // 2 → 1, 2048 → 11
  const hue = (40 - level * 18 + 360) % 360;
  const light = Math.max(38, 86 - level * 4.5);
  return { fill: `hsl(${hue} 70% ${light}%)`, text: light > 62 ? "#3a3226" : "#fdfaf4" };
}

function cellOrigin(layout: Layout, x: number, y: number): [number, number] {
  return [layout.originX + layout.gap + x * (layout.cell + layout.gap), layout.originY + layout.gap + y * (layout.cell + layout.gap)];
}

export function drawBoard(ctx: CanvasRenderingContext2D, layout: Layout): void {
  const span = layout.size * (layout.cell + layout.gap) + layout.gap;
  ctx.fillStyle = "#1a1e27";
  ctx.beginPath();
  ctx.roundRect(layout.originX, layout.originY, span, span, 10);
  ctx.fill();
  ctx.fillStyle = "#252b38";
  for (let y = 0; y < layout.size; y++) {
    for (let x = 0; x < layout.size; x++) {
      const [px, py] = cellOrigin(layout, x, y);
      ctx.beginPath();
      ctx.roundRect(px, py, layout.cell, layout.cell, 6);
      ctx.fill();
    }
  }
}

export function drawSprites(ctx: CanvasRenderingContext2D, layout: Layout, sprites: readonly Sprite[]): void {
  for (const s of sprites) {
    if (s.scale <= 0) continue;
    const [px, py] = cellOrigin(layout, s.x, s.y);
    const half = layout.cell / 2;
    const { fill, text } = tileColors(s.value);
    ctx.save();
    ctx.translate(px + half, py + half);
    ctx.scale(s.scale, s.scale);
    ctx.fillStyle = fill;
    ctx.beginPath();
    ctx.roundRect(-half, -half, layout.cell, layout.cell, 6);
    ctx.fill();
    const digits = String(s.value).length;
    ctx.fillStyle = text;
    ctx.font = `bold ${Math.round(layout.cell * (digits <= 2 ? 0.45 : digits === 3 ? 0.36 : 0.28))}px ui-sans-serif, system-ui, sans-serif`;
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(String(s.value), 0, 2);
    ctx.restore();
  }
}

export function drawHud(ctx: CanvasRenderingContext2D, width: number, state: GameState, canUndo: boolean): void {
  ctx.fillStyle = "#cfd6e4";
  ctx.font = "bold 18px ui-monospace, monospace";
  ctx.textAlign = "left";
  ctx.textBaseline = "middle";
  ctx.fillText(`score ${state.score}`, 8, 22);
  ctx.textAlign = "right";
  const gained = state.turn.gained > 0 ? `+${state.turn.gained}  ` : "";
  ctx.fillText(`${gained}moves ${state.moves}${canUndo ? "  ⟲" : ""}`, width - 8, 22);
}

export function drawOverlay(ctx: CanvasRenderingContext2D, layout: Layout, state: GameState): void {
  if (state.status === "playing") return;
  const span = layout.size * (layout.cell + layout.gap) + layout.gap;
  ctx.fillStyle = "rgba(11, 13, 18, 0.72)";
  ctx.fillRect(layout.originX, layout.originY, span, span);
  ctx.fillStyle = "#fdfaf4";
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  const cx = layout.originX + span / 2;
  const cy = layout.originY + span / 2;
  ctx.font = "bold 36px ui-sans-serif, system-ui, sans-serif";
  ctx.fillText(state.status === "won" ? "You win!" : "No moves left", cx, cy - 18);
  ctx.font = "15px ui-monospace, monospace";
  ctx.fillText(state.status === "won" ? "C keep playing · N new game" : "U undo · N new game", cx, cy + 22);
}
