import { describe, expect, it } from "vitest";
import { DEFAULT_CONFIG } from "../src/core/config";
import { fromRows } from "../src/core/grid";
import { createReducer, initialState } from "../src/core/reducer";
import { popScale, spritesAt, TIMING, TURN_DURATION } from "../src/view/animation";
import { directionForSwipe, intentForKey } from "../src/view/input";
import { tileColors } from "../src/view/render";

const reduce = createReducer({ ...DEFAULT_CONFIG, fourChance: 0 });

function turnFrom(rows: string[]) {
  const s = { ...initialState(DEFAULT_CONFIG, 3), grid: fromRows(rows), nextId: 100 };
  return reduce(s, { type: "move", direction: "left" });
}

describe("spritesAt", () => {
  const next = turnFrom([". . 2 2", ". . . .", ". . . .", ". . . ."]);

  it("starts every pre-move tile at its old cell and eases it to the new one", () => {
    const start = spritesAt(next.turn, next.grid, 0);
    expect(start.map((s) => s.x).sort()).toEqual([2, 3]);
    const mid = spritesAt(next.turn, next.grid, TIMING.slide / 2);
    const right = mid.find((s) => s.id === 2)!;
    // Ease-out: past the linear halfway point (3 → 0) at half time.
    expect(right.x).toBeLessThan(1.5);
    expect(right.x).toBeGreaterThan(0);
  });

  it("after the slide shows the final grid, popping merges and growing spawns", () => {
    const settle = spritesAt(next.turn, next.grid, TIMING.slide + TIMING.settle / 2);
    const merged = settle.find((s) => s.id === next.turn.merges[0]!.id)!;
    const spawned = settle.find((s) => s.id === next.turn.spawns[0]!.id)!;
    expect(merged).toMatchObject({ x: 0, y: 0, value: 4 });
    expect(merged.scale).toBeCloseTo(TIMING.popScale);
    expect(spawned.scale).toBeGreaterThan(0);
    expect(spawned.scale).toBeLessThan(1);
    expect(settle.some((s) => s.id === 1 || s.id === 2)).toBe(false);
  });

  it("is fully settled at the end of the turn", () => {
    const end = spritesAt(next.turn, next.grid, TURN_DURATION + 1);
    expect(end.every((s) => s.scale === 1)).toBe(true);
    expect(end).toHaveLength(2);
  });

  it("animates a fresh game's spawns from t = 0", () => {
    const s = initialState(DEFAULT_CONFIG, 8);
    expect(spritesAt(s.turn, s.grid, 0).every((sp) => sp.scale === 0)).toBe(true);
  });

  it("popScale peaks mid-phase and returns to 1", () => {
    expect(popScale(0)).toBe(1);
    expect(popScale(0.5)).toBeCloseTo(TIMING.popScale);
    expect(popScale(1)).toBeCloseTo(1);
  });
});

describe("input mapping", () => {
  it("maps arrows and WASD to the same moves", () => {
    expect(intentForKey("ArrowLeft")).toEqual({ kind: "move", direction: "left" });
    expect(intentForKey("KeyA")).toEqual(intentForKey("ArrowLeft"));
    expect(intentForKey("KeyU")).toEqual({ kind: "undo" });
    expect(intentForKey("KeyQ")).toBeNull();
  });

  it("swipes along the dominant axis and ignores taps and exact diagonals", () => {
    expect(directionForSwipe(60, 10)).toBe("right");
    expect(directionForSwipe(-5, -40)).toBe("up");
    expect(directionForSwipe(5, 5)).toBeNull();
    expect(directionForSwipe(40, 40)).toBeNull();
  });
});

describe("tileColors", () => {
  it("gives each power of two a distinct fill and readable text", () => {
    const fills = new Set([2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048].map((v) => tileColors(v).fill));
    expect(fills.size).toBe(11);
    expect(tileColors(2).text).not.toBe(tileColors(2048).text);
  });
});
