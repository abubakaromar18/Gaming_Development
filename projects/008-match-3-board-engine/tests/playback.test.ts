import { describe, expect, it } from "vitest";
import { fromRows } from "../src/core/board";
import { DEFAULT_CONFIG } from "../src/core/config";
import { playTurn, type GameState } from "../src/core/game";
import { mulberry32 } from "../src/core/rng";
import { fallDistance, fallTime, phasesForTurn, Playback, TIMING } from "../src/view/playback";
import { phaseSprites } from "../src/view/sprites";

const cfg = { ...DEFAULT_CONFIG, width: 3, height: 3, colors: 4 };
const state: GameState = { board: fromRows(["BCD", "CDA", "AAD"]), score: 0, movesMade: 0, longestChain: 0, shuffles: 0 };
const a = { x: 2, y: 1 };
const b = { x: 2, y: 2 };

describe("turn playback", () => {
  it("maps a legal turn to swap, then clear + fall per cascade step", () => {
    const result = playTurn(state, a, b, mulberry32(5), cfg);
    if (!result.ok) throw new Error(result.reason);
    const kinds = phasesForTurn(state.board, a, b, result).map((p) => p.kind);
    expect(kinds.slice(0, 3)).toEqual(["swap", "clear", "fall"]);
    expect(kinds.filter((k) => k === "clear")).toHaveLength(result.steps.length);
  });

  it("bounces an illegal swap and ignores out-of-bounds input", () => {
    const bad = { x: 0, y: 0 };
    const near = { x: 1, y: 0 };
    const phases = phasesForTurn(state.board, bad, near, playTurn(state, bad, near, mulberry32(1), cfg));
    expect(phases).toMatchObject([{ kind: "swap", bounce: true, duration: TIMING.bounce }]);
    const off = { x: -1, y: 0 };
    expect(phasesForTurn(state.board, bad, off, playTurn(state, bad, off, mulberry32(1), cfg))).toEqual([]);
  });

  it("returns a bounced gem to where it started", () => {
    const phase = phasesForTurn(state.board, { x: 0, y: 0 }, { x: 1, y: 0 }, { ok: false, reason: "no-match" })[0]!;
    const mid = phaseSprites(phase, phase.duration / 2);
    const end = phaseSprites(phase, phase.duration);
    // The B at (0,0) reaches its neighbor's cell halfway, then returns home.
    expect(mid.find((s) => s.gem === 1 && s.y === 0)?.x).toBeCloseTo(1);
    expect(end.find((s) => s.gem === 1 && s.y === 0)?.x).toBeCloseTo(0);
  });

  it("consumes phases in order and carries leftover time", () => {
    const pb = new Playback();
    const board = state.board;
    pb.enqueue([
      { kind: "swap", board, a, b, bounce: false, duration: 0.2 },
      { kind: "swap", board, a, b, bounce: true, duration: 0.3 },
    ]);
    pb.update(0.25);
    expect(pb.current?.phase.duration).toBe(0.3);
    expect(pb.current?.elapsed).toBeCloseTo(0.05);
    pb.update(1);
    expect(pb.busy).toBe(false);
    expect(() => pb.update(-1)).toThrow(RangeError);
  });

  it("falls with constant acceleration and lands exactly", () => {
    expect(fallDistance(fallTime(4), 4)).toBeCloseTo(4);
    expect(fallDistance(fallTime(4) / 2, 4)).toBeCloseTo(1); // quarter distance at half time
    expect(fallDistance(10, 4)).toBe(4);
  });
});
