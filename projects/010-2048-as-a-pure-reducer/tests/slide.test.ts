import { describe, expect, it } from "vitest";
import { validateConfig, DEFAULT_CONFIG } from "../src/core/config";
import { fromRows, toRows, type Cell } from "../src/core/grid";
import { applyMove, canMove, hasAnyMove, lines } from "../src/core/moves";
import { nextInt, nextRandom, seedRng } from "../src/core/rng";
import { slideLine } from "../src/core/slide";

/** Slide a line of values (0 = empty) and return the resulting values. */
function slideValues(values: number[]): number[] {
  let id = 1;
  const line: Cell[] = values.map((v) => (v ? { id: id++, value: v } : null));
  const { slots } = slideLine(line);
  const out = slots.map((s) => s.sources.reduce((sum, t) => sum + t.value, 0));
  return [...out, ...Array(values.length - out.length).fill(0)];
}

describe("slideLine", () => {
  it.each([
    [[0, 0, 0, 2], [2, 0, 0, 0]],
    [[2, 0, 2, 0], [4, 0, 0, 0]],
    [[2, 2, 2, 2], [4, 4, 0, 0]],
    [[2, 2, 4, 0], [4, 4, 0, 0]],
    [[4, 4, 8, 8], [8, 16, 0, 0]],
    [[2, 2, 2, 0], [4, 2, 0, 0]],
    [[8, 4, 2, 2], [8, 4, 4, 0]],
    [[2, 4, 2, 4], [2, 4, 2, 4]],
  ])("%j slides to %j", (input, expected) => {
    expect(slideValues(input)).toEqual(expected);
  });

  it("never merges a freshly merged tile again in the same move", () => {
    // [4, 4, 8] would be [16] if merges chained; the rule says [8, 8].
    expect(slideValues([4, 4, 8, 0])).toEqual([8, 8, 0, 0]);
  });

  it("reports unchanged for packed lines with no merges", () => {
    const line: Cell[] = [{ id: 1, value: 2 }, { id: 2, value: 4 }, null, null];
    expect(slideLine(line).changed).toBe(false);
    expect(slideLine([null, { id: 1, value: 2 }]).changed).toBe(true);
    expect(slideLine([]).changed).toBe(false);
  });
});

describe("lines", () => {
  it("orders each line from the edge tiles slide toward", () => {
    expect(lines(2, "left")[0]).toEqual([{ x: 0, y: 0 }, { x: 1, y: 0 }]);
    expect(lines(2, "right")[0]).toEqual([{ x: 1, y: 0 }, { x: 0, y: 0 }]);
    expect(lines(2, "up")[1]).toEqual([{ x: 1, y: 0 }, { x: 1, y: 1 }]);
    expect(lines(2, "down")[1]).toEqual([{ x: 1, y: 1 }, { x: 1, y: 0 }]);
  });
});

describe("applyMove", () => {
  const grid = fromRows([
    "2 . 2 4",
    ". 4 . 4",
    "2 2 2 2",
    ". . . 8",
  ]);

  it.each([
    ["left", ["4 4 . .", "8 . . .", "4 4 . .", "8 . . ."]],
    ["right", [". . 4 4", ". . . 8", ". . 4 4", ". . . 8"]],
    ["up", ["4 4 4 8", ". 2 . 2", ". . . 8", ". . . ."]],
    ["down", [". . . .", ". . . 8", ". 4 . 2", "4 2 4 8"]],
  ] as const)("moves %s", (dir, expected) => {
    expect(toRows(applyMove(grid, dir, 100).grid)).toEqual(expected);
  });

  it("scores the sum of every merged tile", () => {
    expect(applyMove(grid, "left", 100).gained).toBe(4 + 8 + 4 + 4);
  });

  it("gives merged tiles fresh ids and records both sources", () => {
    const g = fromRows(["2 2", ". ."]); // ids 1, 2
    const res = applyMove(g, "left", 50);
    expect(res.merges).toEqual([{ id: 50, value: 4, at: { x: 0, y: 0 }, from: [1, 2] }]);
    expect(res.nextId).toBe(51);
    expect(res.slides).toEqual([
      { id: 1, value: 2, from: { x: 0, y: 0 }, to: { x: 0, y: 0 } },
      { id: 2, value: 2, from: { x: 1, y: 0 }, to: { x: 0, y: 0 } },
    ]);
  });

  it("keeps tile identity for tiles that only slide", () => {
    const g = fromRows([". 2", ". ."]);
    const res = applyMove(g, "left", 9);
    expect(res.grid[0]![0]).toBe(g[0]![1]);
  });

  it("returns the input grid untouched when nothing can move", () => {
    const g = fromRows(["2 4", ". ."]);
    const res = applyMove(g, "left", 7);
    expect(res.moved).toBe(false);
    expect(res.grid).toBe(g);
    expect(res.nextId).toBe(7);
    expect(canMove(g, "left")).toBe(false);
    expect(canMove(g, "down")).toBe(true);
  });
});

describe("hasAnyMove", () => {
  it("is true with an empty cell or a matching neighbor, false otherwise", () => {
    expect(hasAnyMove(fromRows(["2 4", "4 ."]))).toBe(true);
    expect(hasAnyMove(fromRows(["2 4", "2 8"]))).toBe(true);
    expect(hasAnyMove(fromRows(["2 4", "8 2"]))).toBe(false);
  });
});

describe("fixtures and config", () => {
  it("round-trips and rejects malformed input", () => {
    expect(toRows(fromRows(["2 .", ". 1024"]))).toEqual(["2 .", ". 1024"]);
    expect(() => fromRows(["3 ."])).toThrow(SyntaxError);
    expect(() => fromRows(["2 2", "2"])).toThrow(SyntaxError);
  });

  it("validates config", () => {
    expect(validateConfig(DEFAULT_CONFIG)).toBe(DEFAULT_CONFIG);
    expect(() => validateConfig({ ...DEFAULT_CONFIG, size: 1 })).toThrow(RangeError);
    expect(() => validateConfig({ ...DEFAULT_CONFIG, target: 1000 })).toThrow(RangeError);
    expect(() => validateConfig({ ...DEFAULT_CONFIG, startTiles: 17 })).toThrow(RangeError);
    expect(() => validateConfig({ ...DEFAULT_CONFIG, fourChance: 1.5 })).toThrow(RangeError);
  });
});

describe("pure rng", () => {
  it("is a function of its state", () => {
    const s = seedRng(42);
    expect(nextRandom(s)).toEqual(nextRandom(s));
    const [a, s1] = nextRandom(s);
    const [b] = nextRandom(s1);
    expect(a).not.toBe(b);
    expect(() => seedRng(Number.NaN)).toThrow(RangeError);
  });

  it("nextInt stays in range", () => {
    let s = seedRng(7);
    for (let i = 0; i < 1000; i++) {
      const [v, next] = nextInt(s, 5);
      expect(v).toBeGreaterThanOrEqual(0);
      expect(v).toBeLessThan(5);
      s = next;
    }
  });
});
