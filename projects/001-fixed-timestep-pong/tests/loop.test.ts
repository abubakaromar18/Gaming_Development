import { describe, expect, it } from "vitest";
import { FixedTimestepLoop } from "../src/core/loop";

function counter(hz: number, maxFrameSeconds?: number) {
  const dts: number[] = [];
  const loop = new FixedTimestepLoop({ hz, maxFrameSeconds }, (dt) => dts.push(dt));
  return { loop, dts };
}

describe("FixedTimestepLoop", () => {
  it("runs a whole number of fixed steps and reports the remainder as alpha", () => {
    const { loop, dts } = counter(60);
    const r = loop.advance(2.5 / 60);
    expect(r.steps).toBe(2);
    expect(dts).toEqual([1 / 60, 1 / 60]);
    expect(r.alpha).toBeCloseTo(0.5, 6);
  });

  it("carries leftover time between frames", () => {
    const { loop } = counter(60);
    expect(loop.advance(0.5 / 60).steps).toBe(0);
    expect(loop.advance(0.5 / 60).steps).toBe(1);
  });

  it("produces the same number of steps regardless of how time is sliced", () => {
    // 1 second of play delivered as 144 Hz frames vs 30 Hz frames.
    const fast = counter(60);
    const slow = counter(60);
    for (let i = 0; i < 144; i++) fast.loop.advance(1 / 144);
    for (let i = 0; i < 30; i++) slow.loop.advance(1 / 30);
    expect(fast.dts.length).toBe(60);
    expect(slow.dts.length).toBe(60);
  });

  it("clamps huge frames to avoid the spiral of death", () => {
    const { loop } = counter(60, 0.1);
    const r = loop.advance(5);
    expect(r.steps).toBe(6);
    expect(r.droppedSeconds).toBeCloseTo(4.9, 6);
  });

  it("keeps alpha in [0, 1)", () => {
    const { loop } = counter(60);
    for (const t of [0.001, 0.016, 0.017, 0.033, 0.0001, 0.05]) {
      const { alpha } = loop.advance(t);
      expect(alpha).toBeGreaterThanOrEqual(0);
      expect(alpha).toBeLessThan(1);
    }
  });

  it("can change rate at runtime", () => {
    const { loop, dts } = counter(60);
    loop.setHz(10);
    loop.advance(0.25);
    expect(dts).toEqual([0.1, 0.1]);
  });

  it("rejects invalid input", () => {
    expect(() => new FixedTimestepLoop({ hz: 0 }, () => {})).toThrow(RangeError);
    const { loop } = counter(60);
    expect(() => loop.advance(-1)).toThrow(RangeError);
    expect(() => loop.advance(Number.NaN)).toThrow(RangeError);
  });
});
