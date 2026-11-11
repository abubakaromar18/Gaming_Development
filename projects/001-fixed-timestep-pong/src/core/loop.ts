/**
 * Fixed-timestep game loop ("Fix Your Timestep", Glenn Fiedler).
 *
 * The simulation always advances in constant `stepSeconds` increments, no
 * matter how fast or slow frames arrive. Leftover time is carried in an
 * accumulator, and the fraction of a step that remains (`alpha`) is handed to
 * the renderer so it can interpolate between the last two simulation states.
 *
 * The loop knows nothing about browsers: time comes in through `advance()`,
 * which makes it fully deterministic and unit-testable.
 */

export interface LoopOptions {
  /** Simulation rate in Hz, e.g. 60 → steps of 1/60 s. */
  readonly hz: number;
  /**
   * Upper bound on real time consumed per frame. Prevents the "spiral of
   * death" where a slow frame demands more steps, which makes the next frame
   * slower still. Time beyond this is dropped (the game briefly slows down).
   */
  readonly maxFrameSeconds?: number;
}

export interface FrameResult {
  /** Number of fixed steps executed this frame. */
  readonly steps: number;
  /** Interpolation factor in [0, 1) between previous and current state. */
  readonly alpha: number;
  /** Real time discarded by the frame clamp. */
  readonly droppedSeconds: number;
}

export class FixedTimestepLoop {
  private accumulator = 0;
  private stepSecondsValue: number;
  private readonly maxFrameSeconds: number;

  constructor(
    options: LoopOptions,
    private readonly onStep: (dt: number) => void,
  ) {
    this.stepSecondsValue = FixedTimestepLoop.toStep(options.hz);
    this.maxFrameSeconds = options.maxFrameSeconds ?? 0.25;
    if (this.maxFrameSeconds <= 0) {
      throw new RangeError("maxFrameSeconds must be positive");
    }
  }

  get stepSeconds(): number {
    return this.stepSecondsValue;
  }

  /** Change the simulation rate at runtime; keeps accumulated time. */
  setHz(hz: number): void {
    this.stepSecondsValue = FixedTimestepLoop.toStep(hz);
  }

  /** Feed real elapsed time since the previous frame. */
  advance(frameSeconds: number): FrameResult {
    if (!Number.isFinite(frameSeconds) || frameSeconds < 0) {
      throw new RangeError(`invalid frame time: ${frameSeconds}`);
    }
    const clamped = Math.min(frameSeconds, this.maxFrameSeconds);
    this.accumulator += clamped;

    let steps = 0;
    const dt = this.stepSecondsValue;
    // Small epsilon absorbs floating-point drift (e.g. 3 × 1/60 ≠ 0.05 exactly).
    while (this.accumulator + 1e-9 >= dt) {
      this.onStep(dt);
      this.accumulator -= dt;
      steps++;
    }
    if (this.accumulator < 0) this.accumulator = 0;

    return {
      steps,
      alpha: this.accumulator / dt,
      droppedSeconds: frameSeconds - clamped,
    };
  }

  private static toStep(hz: number): number {
    if (!Number.isFinite(hz) || hz <= 0) {
      throw new RangeError(`hz must be a positive number, got ${hz}`);
    }
    return 1 / hz;
  }
}
