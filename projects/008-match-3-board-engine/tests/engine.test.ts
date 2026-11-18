import { describe, expect, it } from "vitest";
import { fromRows, toRows, type Board } from "../src/core/board";
import { applyGravity, refill, resolveCascades, scoreRuns } from "../src/core/cascade";
import { DEFAULT_CONFIG, type Match3Config } from "../src/core/config";
import { generateBoard, shuffleBoard } from "../src/core/generate";
import { hint, newGame, playTurn, type GameState } from "../src/core/game";
import { findRuns, hasAnyMatch } from "../src/core/match";
import { hasMoves } from "../src/core/moves";
import { mulberry32, type Rng } from "../src/core/rng";

const SMALL: Match3Config = { ...DEFAULT_CONFIG, width: 3, height: 3, colors: 4 };

/** Rng that yields the given gems (for `colors`) first, then defers to `rest`. */
function scripted(gems: string, colors: number, rest: Rng = mulberry32(1)): Rng {
  const queue = [...gems].map((ch) => (ch.charCodeAt(0) - 65 + 0.5) / colors);
  return () => queue.shift() ?? rest();
}

const sorted = (b: Board) => [...b.cells].sort();

function assertPlayable(b: Board, cfg: Match3Config): void {
  expect(b.cells).toHaveLength(cfg.width * cfg.height);
  expect(b.cells.every((c) => c !== null && c >= 0 && c < cfg.colors)).toBe(true);
  expect(hasAnyMatch(b)).toBe(false);
  expect(hasMoves(b)).toBe(true);
}

describe("gravity", () => {
  it("compacts each column downward and keeps gem order", () => {
    const { board, drops } = applyGravity(fromRows(["A", ".", "B", "."]));
    expect(toRows(board)).toEqual([".", ".", "A", "B"]);
    expect(drops).toEqual([
      { x: 0, fromY: 2, toY: 3, gem: 1 },
      { x: 0, fromY: 0, toY: 2, gem: 0 },
    ]);
  });

  it("leaves settled columns alone", () => {
    expect(applyGravity(fromRows([".A", "BA"])).drops).toEqual([]);
  });
});

describe("refill", () => {
  it("fills top holes with gems stacked above the board", () => {
    const { board, spawns } = refill(fromRows([".", ".", "A"]), scripted("CD", 4), 4);
    expect(toRows(board)).toEqual(["C", "D", "A"]);
    expect(spawns).toEqual([
      { x: 0, fromY: -2, toY: 0, gem: 2 },
      { x: 0, fromY: -1, toY: 1, gem: 3 },
    ]);
  });
});

describe("cascade resolution", () => {
  const board = fromRows(["BCD", "CDB", "AAA"]);

  it("clears, drops, and refills in a single pass", () => {
    const res = resolveCascades(board, scripted("DBC", 4), SMALL);
    expect(res.steps).toHaveLength(1);
    const [step] = res.steps;
    expect(step?.cleared).toHaveLength(3);
    expect(step?.drops).toHaveLength(6);
    expect(step?.spawns.every((s) => s.fromY === -1 && s.toY === 0)).toBe(true);
    expect(toRows(res.board)).toEqual(["DBC", "BCD", "CDB"]);
    expect(res.points).toBe(30);
  });

  it("chains when the refill creates a new match, with a rising multiplier", () => {
    const res = resolveCascades(board, scripted("AAADBC", 4), SMALL);
    expect(res.steps.map((s) => [s.chain, s.points])).toEqual([
      [1, 30],
      [2, 60],
    ]);
    expect(res.steps[1]?.before).toBe(res.steps[0]?.after);
    expect(toRows(res.board)).toEqual(["DBC", "BCD", "CDB"]);
    expect(res.points).toBe(90);
  });

  it("does nothing on a stable board", () => {
    const stable = fromRows(["ABC", "BCA", "CAB"]);
    const res = resolveCascades(stable, mulberry32(1), SMALL);
    expect(res.steps).toEqual([]);
    expect(res.board).toBe(stable);
  });

  it("stops a runaway cascade instead of hanging", () => {
    const alwaysA: Rng = () => 0;
    expect(() => resolveCascades(board, alwaysA, { ...SMALL, maxCascades: 5 })).toThrow(/did not settle/);
  });

  it("scores long runs with a bonus", () => {
    const runs = findRuns(fromRows(["AAAAA"]));
    expect(scoreRuns(runs, 1, DEFAULT_CONFIG)).toBe(5 * 10 + 2 * 20);
    expect(scoreRuns(runs, 3, DEFAULT_CONFIG)).toBe(3 * 90);
  });
});

describe("board generation", () => {
  it("always produces a match-free board with at least one move", () => {
    for (let seed = 0; seed < 300; seed++) assertPlayable(generateBoard(DEFAULT_CONFIG, mulberry32(seed)), DEFAULT_CONFIG);
  });

  it("works at the tightest setting: three colors", () => {
    const cfg = { ...DEFAULT_CONFIG, colors: 3 };
    for (let seed = 0; seed < 100; seed++) assertPlayable(generateBoard(cfg, mulberry32(seed)), cfg);
  });

  it("is reproducible from a seed", () => {
    expect(generateBoard(DEFAULT_CONFIG, mulberry32(42))).toEqual(generateBoard(DEFAULT_CONFIG, mulberry32(42)));
    expect(generateBoard(DEFAULT_CONFIG, mulberry32(42))).not.toEqual(generateBoard(DEFAULT_CONFIG, mulberry32(43)));
  });

  it("rejects invalid configs", () => {
    expect(() => generateBoard({ ...DEFAULT_CONFIG, colors: 2 }, mulberry32(1))).toThrow(RangeError);
  });
});

describe("shuffle", () => {
  const cfg = { ...DEFAULT_CONFIG, width: 4, height: 4, colors: 4 };
  const dead = fromRows(["ABCD", "CDAB", "ABCD", "CDAB"]);

  it("turns a dead board into a playable one with the same gems", () => {
    expect(hasMoves(dead)).toBe(false);
    const shuffled = shuffleBoard(dead, mulberry32(7), cfg);
    assertPlayable(shuffled, cfg);
    expect(sorted(shuffled)).toEqual(sorted(dead));
  });

  it("falls back to a new board when the gem mix can never be playable", () => {
    // Nine distinct gems on 3×3 can never line up three of a kind.
    const hopeless = fromRows(["ABC", "DEF", "GHI"]);
    const nine = { ...DEFAULT_CONFIG, width: 3, height: 3, colors: 9, maxGenerateAttempts: 20 };
    assertPlayable(shuffleBoard(hopeless, mulberry32(3), nine), nine);
  });
});

describe("turns", () => {
  const start: GameState = {
    board: fromRows(["BCD", "CDA", "AAD"]),
    score: 0,
    movesMade: 0,
    longestChain: 0,
    shuffles: 0,
  };

  it("rejects an illegal swap and leaves the state untouched", () => {
    const res = playTurn(start, { x: 0, y: 0 }, { x: 1, y: 0 }, mulberry32(1), SMALL);
    expect(res).toEqual({ ok: false, reason: "no-match" });
  });

  it("scores a legal swap and records it for animation", () => {
    const res = playTurn(start, { x: 2, y: 1 }, { x: 2, y: 2 }, scripted("DBC", 4), SMALL);
    if (!res.ok) throw new Error(res.reason);
    expect(toRows(res.swapped)).toEqual(["BCD", "CDD", "AAA"]);
    expect(res.steps[0]?.before).toBe(res.swapped);
    expect(res.state.score).toBe(30);
    expect(res.state.movesMade).toBe(1);
    expect(res.state.longestChain).toBe(1);
  });

  it("reshuffles when the turn leaves no legal moves", () => {
    // Refill "ABA" produces ABA / BCD / CDD, which has no moves.
    const res = playTurn(start, { x: 2, y: 1 }, { x: 2, y: 2 }, scripted("ABA", 4), SMALL);
    if (!res.ok) throw new Error(res.reason);
    const settled = res.steps.at(-1)!.after;
    expect(toRows(settled)).toEqual(["ABA", "BCD", "CDD"]);
    expect(hasMoves(settled)).toBe(false);
    expect(res.reshuffled).not.toBeNull();
    expect(res.state.board).toBe(res.reshuffled);
    expect(res.state.shuffles).toBe(1);
    assertPlayable(res.state.board, SMALL);
  });

  it("hints the move that clears the most gems", () => {
    // Several swaps work, but dropping the A at (2,1) into row 0 makes five in a row.
    const b = fromRows(["AABAA", "CDADC", "DCBCD"]);
    expect(hint({ ...start, board: b })).toEqual({ a: { x: 2, y: 0 }, b: { x: 2, y: 1 } });
    expect(hint({ ...start, board: fromRows(["ABCD", "CDAB", "ABCD", "CDAB"]) })).toBeNull();
  });
});

describe("long-run invariants", () => {
  function playMany(seed: number, turns: number): GameState {
    const rng = mulberry32(seed);
    let state = newGame(DEFAULT_CONFIG, rng);
    for (let t = 0; t < turns; t++) {
      const move = hint(state);
      if (!move) throw new Error("newGame/reshuffle left a dead board");
      const res = playTurn(state, move.a, move.b, rng, DEFAULT_CONFIG);
      if (!res.ok) throw new Error(`hinted move rejected: ${res.reason}`);
      expect(res.steps.map((s) => s.chain)).toEqual(res.steps.map((_, i) => i + 1));
      expect(res.state.score).toBeGreaterThan(state.score);
      assertPlayable(res.state.board, DEFAULT_CONFIG);
      state = res.state;
    }
    return state;
  }

  it("stays playable for hundreds of turns across seeds", () => {
    for (const seed of [1, 2, 3, 4, 5]) expect(playMany(seed, 200).movesMade).toBe(200);
  });

  it("replays identically from the same seed and moves", () => {
    expect(playMany(99, 50)).toEqual(playMany(99, 50));
  });
});
