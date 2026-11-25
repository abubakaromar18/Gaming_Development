namespace Platformer.Core.Controller;

/// <summary>
/// Every number that shapes how the character feels, in designer units: pixels, pixels
/// per second and seconds. The jump is specified as a height and a time to reach it, and
/// gravity and launch speed are derived from those, because "jumps 3 tiles in a third of a
/// second" is something a designer can reason about and "gravity 800" is not.
///
/// Setting a feature's value to its neutral setting turns it off, which is how the demo
/// toggles them live: <see cref="CoyoteTime"/> 0, <see cref="JumpBufferTime"/> 0,
/// <see cref="JumpCutMultiplier"/> 1, <see cref="ApexGravityMultiplier"/> 1 with
/// <see cref="ApexSpeedBonus"/> 1, <see cref="CornerCorrection"/> 0.
/// </summary>
public sealed record ControllerTuning
{
    // Body
    public float BodyWidth { get; init; } = 12f;
    public float BodyHeight { get; init; } = 14f;

    // Horizontal movement
    public float MaxRunSpeed { get; init; } = 150f;
    public float GroundAcceleration { get; init; } = 1400f;
    public float GroundDeceleration { get; init; } = 1800f;
    /// <summary>Used on the ground when input opposes the current velocity: snappy turnarounds.</summary>
    public float TurnAcceleration { get; init; } = 2800f;
    public float AirAcceleration { get; init; } = 1000f;
    public float AirDeceleration { get; init; } = 450f;

    // Jump arc
    /// <summary>Peak height of a full (held) jump, ignoring apex hang, in pixels.</summary>
    public float JumpHeight { get; init; } = 52f;
    /// <summary>Seconds from take-off to the peak of a full jump, ignoring apex hang.</summary>
    public float TimeToApex { get; init; } = 0.36f;
    /// <summary>Gravity multiplier while falling. Above 1 gives a snappier, less floaty descent.</summary>
    public float FallGravityMultiplier { get; init; } = 1.8f;
    public float MaxFallSpeed { get; init; } = 420f;

    // Forgiveness and feel
    /// <summary>Seconds after leaving a ledge during which a jump still counts as grounded.</summary>
    public float CoyoteTime { get; init; } = 0.10f;
    /// <summary>Seconds a jump press is remembered before landing.</summary>
    public float JumpBufferTime { get; init; } = 0.12f;
    /// <summary>Upward velocity is multiplied by this when jump is released early. 1 = fixed-height jumps.</summary>
    public float JumpCutMultiplier { get; init; } = 0.45f;
    /// <summary>Vertical speed (px/s) under which a held jump counts as being at its apex.</summary>
    public float ApexThreshold { get; init; } = 45f;
    /// <summary>Gravity multiplier at the apex of a held jump. Below 1 gives a brief hang.</summary>
    public float ApexGravityMultiplier { get; init; } = 0.45f;
    /// <summary>Max run speed multiplier at the apex, for a little extra air control.</summary>
    public float ApexSpeedBonus { get; init; } = 1.1f;
    /// <summary>Max horizontal pixels the body is nudged to slip past a ceiling corner.</summary>
    public int CornerCorrection { get; init; } = 4;

    /// <summary>Base gravity in px/s², from g = 2h / t².</summary>
    public float Gravity => 2f * JumpHeight / (TimeToApex * TimeToApex);

    /// <summary>Launch speed in px/s, from v = 2h / t (equivalently g·t).</summary>
    public float JumpVelocity => 2f * JumpHeight / TimeToApex;

    public static ControllerTuning Default { get; } = new();

    /// <summary>Throws <see cref="ArgumentException"/> naming the first invalid value.</summary>
    public ControllerTuning Validated()
    {
        Positive(BodyWidth, nameof(BodyWidth));
        Positive(BodyHeight, nameof(BodyHeight));
        Positive(MaxRunSpeed, nameof(MaxRunSpeed));
        Positive(GroundAcceleration, nameof(GroundAcceleration));
        Positive(GroundDeceleration, nameof(GroundDeceleration));
        Positive(TurnAcceleration, nameof(TurnAcceleration));
        Positive(AirAcceleration, nameof(AirAcceleration));
        Positive(AirDeceleration, nameof(AirDeceleration));
        Positive(JumpHeight, nameof(JumpHeight));
        Positive(TimeToApex, nameof(TimeToApex));
        Positive(MaxFallSpeed, nameof(MaxFallSpeed));
        Range(FallGravityMultiplier, 1f, 5f, nameof(FallGravityMultiplier));
        Range(CoyoteTime, 0f, 0.5f, nameof(CoyoteTime));
        Range(JumpBufferTime, 0f, 0.5f, nameof(JumpBufferTime));
        Range(JumpCutMultiplier, 0f, 1f, nameof(JumpCutMultiplier));
        Range(ApexThreshold, 0f, JumpVelocity, nameof(ApexThreshold));
        Range(ApexGravityMultiplier, 0.05f, 1f, nameof(ApexGravityMultiplier));
        Range(ApexSpeedBonus, 1f, 2f, nameof(ApexSpeedBonus));
        if (CornerCorrection < 0 || CornerCorrection >= BodyWidth / 2f)
        {
            throw new ArgumentException(
                $"{nameof(CornerCorrection)} must be in [0, {nameof(BodyWidth)}/2), got {CornerCorrection}.");
        }
        return this;
    }

    private static void Positive(float value, string name)
    {
        if (!(value > 0f) || float.IsInfinity(value))
        {
            throw new ArgumentException($"{name} must be a positive finite number, got {value}.");
        }
    }

    private static void Range(float value, float min, float max, string name)
    {
        if (!(value >= min && value <= max))
        {
            throw new ArgumentException($"{name} must be in [{min}, {max}], got {value}.");
        }
    }
}
