/** Tunable rules. Validated once when a reducer is created, never per action. */
export interface GameConfig {
  /** Board is size × size. */
  readonly size: number;
  /** Tile value that wins the game. Must be a power of two. */
  readonly target: number;
  /** Tiles placed on a fresh board. */
  readonly startTiles: number;
  /** Chance that a spawned tile is a 4 instead of a 2. */
  readonly fourChance: number;
}

export const DEFAULT_CONFIG: GameConfig = {
  size: 4,
  target: 2048,
  startTiles: 2,
  fourChance: 0.1,
};

const isPowerOfTwo = (n: number) => Number.isInteger(n) && n >= 4 && (n & (n - 1)) === 0;

export function validateConfig(config: GameConfig): GameConfig {
  const { size, target, startTiles, fourChance } = config;
  if (!Number.isInteger(size) || size < 2 || size > 16) throw new RangeError(`size must be an integer in [2, 16], got ${size}`);
  if (!isPowerOfTwo(target)) throw new RangeError(`target must be a power of two ≥ 4, got ${target}`);
  if (!Number.isInteger(startTiles) || startTiles < 1 || startTiles > size * size) {
    throw new RangeError(`startTiles must be in [1, ${size * size}], got ${startTiles}`);
  }
  if (!(fourChance >= 0 && fourChance <= 1)) throw new RangeError(`fourChance must be in [0, 1], got ${fourChance}`);
  return config;
}
