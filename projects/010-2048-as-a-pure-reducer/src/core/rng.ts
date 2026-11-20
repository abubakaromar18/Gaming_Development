/**
 * A *pure* seedable PRNG (mulberry32). Instead of a closure that mutates hidden
 * state, `nextRandom` takes the current state and returns the value together
 * with the next state. That lets the RNG state live inside `GameState`, so the
 * reducer stays a pure function: same state + same action = same result,
 * including where the next tile spawns.
 */
export type RngState = number;

export function seedRng(seed: number): RngState {
  if (!Number.isFinite(seed)) throw new RangeError(`seed must be a finite number, got ${seed}`);
  return Math.trunc(seed) >>> 0;
}

/** Returns a float in [0, 1) and the advanced RNG state. */
export function nextRandom(state: RngState): [value: number, next: RngState] {
  const next = (state + 0x6d2b79f5) >>> 0;
  let t = next;
  t = Math.imul(t ^ (t >>> 15), t | 1);
  t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
  return [((t ^ (t >>> 14)) >>> 0) / 4294967296, next];
}

/** Uniform integer in [0, n) plus the advanced RNG state. */
export function nextInt(state: RngState, n: number): [value: number, next: RngState] {
  const [r, next] = nextRandom(state);
  return [Math.min(n - 1, Math.floor(r * n)), next];
}
