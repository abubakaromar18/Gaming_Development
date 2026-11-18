/** Tunable rules for a match-3 game. Everything the engine needs is passed in. */
export interface Match3Config {
  readonly width: number;
  readonly height: number;
  /** Number of distinct gem colors. Fewer colors means more matches and cascades. */
  readonly colors: number;
  /** Points awarded per gem in a run. */
  readonly pointsPerGem: number;
  /** Extra points for each gem beyond the third in a single run. */
  readonly longRunBonus: number;
  /** Safety valve: a resolution that cascades more often than this is treated as a bug. */
  readonly maxCascades: number;
  /** How many random boards or shuffles to try before giving up. */
  readonly maxGenerateAttempts: number;
}

export const DEFAULT_CONFIG: Match3Config = {
  width: 8,
  height: 8,
  colors: 6,
  pointsPerGem: 10,
  longRunBonus: 20,
  maxCascades: 50,
  maxGenerateAttempts: 200,
};

/** Throws a RangeError describing the first invalid field. */
export function validateConfig(cfg: Match3Config): void {
  const ints: (keyof Match3Config)[] = ["width", "height", "colors", "maxCascades", "maxGenerateAttempts"];
  for (const key of ints) {
    if (!Number.isInteger(cfg[key]) || cfg[key] <= 0) throw new RangeError(`${key} must be a positive integer`);
  }
  if (cfg.width < 3 || cfg.height < 3) throw new RangeError("board must be at least 3×3");
  // With 3+ colors, generation can always pick a gem that avoids both the
  // horizontal and the vertical neighbor pair, so it never dead-ends.
  if (cfg.colors < 3 || cfg.colors > 26) throw new RangeError("colors must be between 3 and 26");
  if (cfg.pointsPerGem < 0 || cfg.longRunBonus < 0) throw new RangeError("points must be non-negative");
}
