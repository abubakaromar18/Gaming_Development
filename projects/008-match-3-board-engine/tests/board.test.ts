import { describe, expect, it } from "vitest";
import { cellAt, fromRows, swapCells, toRows } from "../src/core/board";
import { DEFAULT_CONFIG, validateConfig } from "../src/core/config";
import { findRuns, isMatchedAt, uniqueCells } from "../src/core/match";
import { findMoves, hasMoves, validateSwap } from "../src/core/moves";

describe("board text format", () => {
  it("round-trips letters and holes", () => {
    const rows = ["AB.", "CAB"];
    const b = fromRows(rows);
    expect(cellAt(b, { x: 1, y: 0 })).toBe(1);
    expect(cellAt(b, { x: 2, y: 0 })).toBeNull();
    expect(toRows(b)).toEqual(rows);
  });

  it("rejects ragged rows and unknown characters", () => {
    expect(() => fromRows(["AB", "A"])).toThrow(RangeError);
    expect(() => fromRows(["A?"])).toThrow(RangeError);
  });

  it("swaps without mutating the original", () => {
    const b = fromRows(["AB"]);
    expect(toRows(swapCells(b, { x: 0, y: 0 }, { x: 1, y: 0 }))).toEqual(["BA"]);
    expect(toRows(b)).toEqual(["AB"]);
  });
});

describe("config validation", () => {
  it("accepts the defaults", () => expect(() => validateConfig(DEFAULT_CONFIG)).not.toThrow());
  it.each([
    { colors: 2 },
    { width: 2 },
    { height: 0 },
    { maxCascades: 1.5 },
    { pointsPerGem: -1 },
  ])("rejects %o", (patch) => {
    expect(() => validateConfig({ ...DEFAULT_CONFIG, ...patch })).toThrow(RangeError);
  });
});

describe("match detection", () => {
  it("finds horizontal and vertical runs", () => {
    const runs = findRuns(fromRows(["AAAB", "CDBB", "CDEF", "CDFE"]));
    expect(runs.map((r) => [r.orientation, r.gem, r.cells.length])).toEqual([
      ["horizontal", 0, 3],
      ["vertical", 2, 3],
      ["vertical", 3, 3],
    ]);
  });

  it("reports a run of five as one maximal run", () => {
    const runs = findRuns(fromRows(["AAAAA"]));
    expect(runs).toHaveLength(1);
    expect(runs[0]?.cells).toHaveLength(5);
  });

  it("counts the shared corner of an L shape once", () => {
    const b = fromRows(["ABC", "ACB", "AAA"]);
    const runs = findRuns(b);
    expect(runs).toHaveLength(2);
    expect(uniqueCells(b, runs)).toHaveLength(5);
  });

  it("ignores runs of holes", () => {
    expect(findRuns(fromRows(["...", "ABC"]))).toEqual([]);
  });

  it("checks a single cell along its row and column", () => {
    const b = fromRows(["ABA", "ABA", "CBC"]);
    expect(isMatchedAt(b, { x: 1, y: 2 })).toBe(true);
    expect(isMatchedAt(b, { x: 0, y: 0 })).toBe(false);
  });
});

describe("swap validation", () => {
  // Swapping (2,0)↔(3,0) puts the A at x=2, completing AAA.
  const b = fromRows(["AABA", "CDCD", "DCDC"]);

  it("accepts a swap that creates a run", () => {
    expect(validateSwap(b, { x: 2, y: 0 }, { x: 3, y: 0 })).toBeNull();
  });

  it("explains each rejection", () => {
    expect(validateSwap(b, { x: 0, y: 0 }, { x: 2, y: 0 })).toBe("not-adjacent");
    expect(validateSwap(b, { x: 3, y: 0 }, { x: 4, y: 0 })).toBe("out-of-bounds");
    expect(validateSwap(b, { x: 0, y: 1 }, { x: 0, y: 2 })).toBe("no-match");
    expect(validateSwap(b, { x: 0, y: 0 }, { x: 1, y: 0 })).toBe("no-match");
    expect(validateSwap(fromRows(["A.", "BC"]), { x: 0, y: 0 }, { x: 1, y: 0 })).toBe("empty-cell");
  });

  it("finds a match created by either swapped gem", () => {
    // Moving B down completes BBB in row 1; the A moving up matches nothing.
    const v = fromRows(["CBCD", "BABB", "DCDC"]);
    expect(validateSwap(v, { x: 1, y: 0 }, { x: 1, y: 1 })).toBeNull();
  });
});

describe("no-moves detection", () => {
  it("recognizes a dead board", () => {
    const dead = fromRows(["ABCD", "CDAB", "ABCD", "CDAB"]);
    expect(hasMoves(dead)).toBe(false);
    expect(findMoves(dead)).toEqual([]);
  });

  it("lists every legal move exactly once", () => {
    // Only the B at (2,0) dropping into the A's slot completes a run (AAA).
    const b = fromRows(["AABC", "CDAD", "BCBA"]);
    const moves = findMoves(b).map((m) => `${m.a.x},${m.a.y}-${m.b.x},${m.b.y}`);
    expect(moves).toEqual(["2,0-2,1"]);
    expect(hasMoves(b)).toBe(true);
  });
});
