/**
 * Canvas drawing. Every gem is a procedurally drawn shape, and each color also
 * has a distinct silhouette, so the board stays readable for color-blind
 * players.
 */
import type { Gem, Pos } from "../core/board";
import type { Sprite } from "./sprites";

export interface Layout {
  readonly cell: number;
  readonly originX: number;
  readonly originY: number;
  readonly cols: number;
  readonly rows: number;
}

const PALETTE = ["#ef476f", "#f78c3b", "#ffd166", "#06d6a0", "#4cc9f0", "#9b5de5"];
type Shape = (c: CanvasRenderingContext2D, r: number) => void;

const polygon = (points: number, phase: number): Shape => (c, r) => {
  c.beginPath();
  for (let i = 0; i < points; i++) {
    const a = phase + (i * 2 * Math.PI) / points;
    c.lineTo(Math.cos(a) * r, Math.sin(a) * r);
  }
  c.closePath();
};

const SHAPES: readonly Shape[] = [
  (c, r) => {
    c.beginPath();
    c.arc(0, 0, r * 0.9, 0, Math.PI * 2);
  },
  (c, r) => {
    c.beginPath();
    c.roundRect(-r * 0.8, -r * 0.8, r * 1.6, r * 1.6, r * 0.25);
  },
  polygon(4, -Math.PI / 2),
  (c, r) => polygon(3, -Math.PI / 2)(c, r * 1.1),
  polygon(6, 0),
  (c, r) => {
    c.beginPath();
    for (let i = 0; i < 10; i++) {
      const a = -Math.PI / 2 + (i * Math.PI) / 5;
      const k = i % 2 === 0 ? r : r * 0.45;
      c.lineTo(Math.cos(a) * k, Math.sin(a) * k);
    }
    c.closePath();
  },
];

function colorOf(gem: Gem): string {
  return PALETTE[gem] ?? `hsl(${(gem * 137) % 360} 70% 60%)`;
}

export function drawBackground(c: CanvasRenderingContext2D, l: Layout): void {
  for (let y = 0; y < l.rows; y++) {
    for (let x = 0; x < l.cols; x++) {
      c.fillStyle = (x + y) % 2 === 0 ? "#161a24" : "#1b2030";
      c.fillRect(l.originX + x * l.cell, l.originY + y * l.cell, l.cell, l.cell);
    }
  }
}

export function drawSprites(c: CanvasRenderingContext2D, l: Layout, sprites: readonly Sprite[]): void {
  c.save();
  // Clip to the board so gems spawning above the top edge stay hidden.
  c.beginPath();
  c.rect(l.originX, l.originY, l.cols * l.cell, l.rows * l.cell);
  c.clip();
  for (const s of sprites) {
    if (s.scale <= 0 || s.alpha <= 0) continue;
    c.save();
    c.globalAlpha = s.alpha;
    c.translate(l.originX + (s.x + 0.5) * l.cell, l.originY + (s.y + 0.5) * l.cell);
    c.scale(s.scale, s.scale);
    const shape = SHAPES[s.gem % SHAPES.length] ?? SHAPES[0]!;
    shape(c, l.cell * 0.36);
    c.fillStyle = colorOf(s.gem);
    c.fill();
    c.lineWidth = 2;
    c.strokeStyle = "rgba(255,255,255,0.35)";
    c.stroke();
    // A small highlight gives the flat shapes a faceted, gem-like look.
    c.fillStyle = "rgba(255,255,255,0.28)";
    c.beginPath();
    c.ellipse(-l.cell * 0.1, -l.cell * 0.12, l.cell * 0.09, l.cell * 0.05, -0.6, 0, Math.PI * 2);
    c.fill();
    c.restore();
  }
  c.restore();
}

export function drawCellOutline(c: CanvasRenderingContext2D, l: Layout, p: Pos, color: string, alpha: number): void {
  c.save();
  c.globalAlpha = alpha;
  c.strokeStyle = color;
  c.lineWidth = 3;
  c.strokeRect(l.originX + p.x * l.cell + 3, l.originY + p.y * l.cell + 3, l.cell - 6, l.cell - 6);
  c.restore();
}
