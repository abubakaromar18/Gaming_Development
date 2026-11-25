using Platformer.Core.Controller;
using Platformer.Core.Effects;
using Platformer.Core.Input;
using Platformer.Core.Physics;
using Platformer.Core.Randomness;
using Platformer.Core.Simulation;
using Platformer.Core.World;

namespace Platformer.Core.Game;

/// <summary>Running counts shown on the HUD, so you can see which assists actually fired.</summary>
public sealed record SessionStats(int Jumps, int CoyoteJumps, int BufferedJumps, int JumpCuts, int CornerCorrections, int Falls);

/// <summary>Session-level settings, separate from how the character feels.</summary>
public sealed record SessionConfig
{
    /// <summary>Simulation rate. 120 Hz keeps per-step movement small and input latency low.</summary>
    public float StepSeconds { get; init; } = 1f / 120f;
    public int MaxStepsPerFrame { get; init; } = 8;
    public int DustCapacity { get; init; } = 64;
    public int TrailLength { get; init; } = 180;

    public static SessionConfig Default { get; } = new();
}

/// <summary>
/// The composition root of the core: one level, one character, the fixed-step loop and the
/// cosmetic systems that react to the controller. The platform layer calls
/// <see cref="Update"/> once per render frame and reads state back to draw it.
/// </summary>
public sealed class PlatformerSession
{
    private readonly ControllerTuning _baseTuning;
    private readonly FixedStepper _stepper;
    private readonly InputLatch _input = new();
    private Vec2 _previousFeet;
    private int _jumps, _coyoteJumps, _bufferedJumps, _jumpCuts, _corrections, _falls;

    public PlatformerSession(Level level, ControllerTuning tuning, IRandom random, SessionConfig? config = null)
    {
        Level = level ?? throw new ArgumentNullException(nameof(level));
        ArgumentNullException.ThrowIfNull(tuning);
        ArgumentNullException.ThrowIfNull(random);
        Config = config ?? SessionConfig.Default;

        _baseTuning = tuning.Validated();
        _stepper = new FixedStepper(Config.StepSeconds, Config.MaxStepsPerFrame);
        Controller = new CharacterController(_baseTuning, level.Map, level.Spawn);
        Dust = new DustSystem(Config.DustCapacity, random);
        Trail = new TrailBuffer(Config.TrailLength);
        _previousFeet = Controller.Feet;
    }

    public Level Level { get; }
    public SessionConfig Config { get; }
    public CharacterController Controller { get; }
    public SquashStretch Squash { get; } = new();
    public DustSystem Dust { get; }
    public TrailBuffer Trail { get; }
    public AssistFeature EnabledAssists { get; private set; } = AssistFeature.All;

    /// <summary>Total simulated steps; with the fixed step this is the session's clock.</summary>
    public long StepCount { get; private set; }

    public SessionStats Stats => new(_jumps, _coyoteJumps, _bufferedJumps, _jumpCuts, _corrections, _falls);

    /// <summary>Feet position blended between the last two steps, for smooth rendering at any frame rate.</summary>
    public Vec2 InterpolatedFeet => Vec2.Lerp(_previousFeet, Controller.Feet, _stepper.Alpha);

    /// <summary>Advance by one render frame's worth of time. Returns the number of simulation steps run.</summary>
    public int Update(float frameSeconds, ControllerInput input)
    {
        _input.Submit(input);
        var steps = _stepper.Advance(frameSeconds);
        for (var i = 0; i < steps; i++)
        {
            Step(_input.NextStep());
        }
        return steps;
    }

    public void ToggleAssist(AssistFeature feature)
    {
        if (!Assists.Ordered.Contains(feature))
        {
            throw new ArgumentOutOfRangeException(nameof(feature), feature, "Toggle one assist at a time.");
        }
        SetAssists(EnabledAssists ^ feature);
    }

    public void SetAssists(AssistFeature enabled)
    {
        EnabledAssists = enabled & AssistFeature.All;
        Controller.Tuning = Assists.Apply(_baseTuning, EnabledAssists);
    }

    /// <summary>Put the character back at the spawn and clear motion and effects.</summary>
    public void Respawn()
    {
        Controller.Teleport(Level.Spawn);
        _previousFeet = Controller.Feet;
        _input.Clear();
        _stepper.Reset();
        Squash.Reset();
        Dust.Clear();
        Trail.Clear();
    }

    private void Step(ControllerInput input)
    {
        var dt = _stepper.StepSeconds;
        _previousFeet = Controller.Feet;
        Controller.Step(input, dt);
        StepCount++;

        foreach (var e in Controller.Events)
        {
            Squash.React(e);
            switch (e.Kind)
            {
                case ControllerEventKind.Jumped:
                    _jumps++;
                    if (e.Assist.HasFlag(JumpAssist.Coyote))
                    {
                        _coyoteJumps++;
                    }
                    if (e.Assist.HasFlag(JumpAssist.Buffered))
                    {
                        _bufferedJumps++;
                    }
                    Dust.Burst(e.Feet, 4, 50f);
                    break;
                case ControllerEventKind.Landed:
                    // Harder landings kick up more dust.
                    Dust.Burst(e.Feet, 3 + (int)(e.Value / 80f), 40f + e.Value * 0.15f);
                    break;
                case ControllerEventKind.JumpCut:
                    _jumpCuts++;
                    break;
                case ControllerEventKind.CornerCorrected:
                    _corrections++;
                    break;
            }
        }

        Squash.Update(dt);
        Dust.Update(dt);
        Trail.Add(Controller.Feet);

        if (Controller.Body.Top > Level.Map.PixelHeight)
        {
            _falls++;
            Respawn();
        }
    }
}
