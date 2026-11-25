namespace Platformer.Core.Simulation;

/// <summary>
/// The fixed-timestep accumulator (see project 001). Platformer feel is tuned in steps:
/// a coyote window of 0.1 s must be the same 12 steps at 30 fps and at 240 fps, and the
/// jump arc must not change shape with frame rate. So the controller only ever sees
/// <see cref="StepSeconds"/>, and render frames are converted into whole steps here.
/// </summary>
public sealed class FixedStepper
{
    private float _accumulator;

    public FixedStepper(float stepSeconds, int maxStepsPerFrame)
    {
        if (!(stepSeconds > 0f) || float.IsInfinity(stepSeconds))
        {
            throw new ArgumentOutOfRangeException(nameof(stepSeconds), stepSeconds, "Step must be positive and finite.");
        }
        if (maxStepsPerFrame < 1)
        {
            throw new ArgumentOutOfRangeException(nameof(maxStepsPerFrame), maxStepsPerFrame, "Need at least one step per frame.");
        }
        StepSeconds = stepSeconds;
        MaxStepsPerFrame = maxStepsPerFrame;
    }

    public float StepSeconds { get; }
    public int MaxStepsPerFrame { get; }

    /// <summary>
    /// How far the next, not-yet-simulated step is: 0 = exactly at the last step, near 1 =
    /// almost due. The renderer blends the previous and current state by this.
    /// </summary>
    public float Alpha => _accumulator / StepSeconds;

    /// <summary>
    /// Add a frame's elapsed time and return how many steps to run now. When a frame is so
    /// long that more than <see cref="MaxStepsPerFrame"/> steps are due (a breakpoint, a
    /// dragged window), the surplus is dropped instead of being simulated: the game slows
    /// down for a moment rather than freezing in a "spiral of death".
    /// </summary>
    public int Advance(float frameSeconds)
    {
        if (float.IsNaN(frameSeconds) || frameSeconds < 0f)
        {
            throw new ArgumentOutOfRangeException(nameof(frameSeconds), frameSeconds, "Frame time can't be negative.");
        }
        _accumulator += MathF.Min(frameSeconds, StepSeconds * MaxStepsPerFrame);
        var steps = (int)(_accumulator / StepSeconds);
        if (steps > MaxStepsPerFrame)
        {
            steps = MaxStepsPerFrame;
        }
        _accumulator -= steps * StepSeconds;
        if (_accumulator < 0f)
        {
            _accumulator = 0f;
        }
        return steps;
    }

    public void Reset() => _accumulator = 0f;
}
