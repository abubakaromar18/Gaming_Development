namespace SceneShell.Core.Arcade;

/// <summary>
/// Every tunable number in the arcade, in one validated place. Sizes are in logical pixels
/// (the platform layer scales the 320×180 frame up), speeds in pixels per second and times
/// in seconds.
/// </summary>
public sealed record ArcadeConfig
{
    public static ArcadeConfig Default { get; } = new();

    public int Width { get; init; } = 320;
    public int Height { get; init; } = 180;

    /// <summary>Seconds for each half of the menu fade.</summary>
    public float FadeSeconds { get; init; } = 0.35f;

    /// <summary>Seconds for each half of the wipe into and out of games.</summary>
    public float WipeSeconds { get; init; } = 0.4f;

    /// <summary>Longest frame the simulation will accept; longer hitches are clamped.</summary>
    public float MaxFrameSeconds { get; init; } = 0.1f;

    public int StarCount { get; init; } = 48;

    public DodgeConfig Dodge { get; init; } = new();
    public CollectConfig Collect { get; init; } = new();

    /// <summary>Throws <see cref="ArgumentException"/> naming the first invalid setting.</summary>
    public void Validate()
    {
        Check.Positive(Width, nameof(Width));
        Check.Positive(Height, nameof(Height));
        Check.Positive(FadeSeconds, nameof(FadeSeconds));
        Check.Positive(WipeSeconds, nameof(WipeSeconds));
        Check.Positive(MaxFrameSeconds, nameof(MaxFrameSeconds));
        Check.NonNegative(StarCount, nameof(StarCount));
        ArgumentNullException.ThrowIfNull(Dodge, nameof(Dodge));
        ArgumentNullException.ThrowIfNull(Collect, nameof(Collect));
        Dodge.Validate(Width);
        Collect.Validate(Width);
    }
}

/// <summary>Dodge: slide along the floor and avoid falling blocks for as long as possible.</summary>
public sealed record DodgeConfig
{
    public float PlayerWidth { get; init; } = 16f;
    public float PlayerHeight { get; init; } = 8f;
    public float PlayerSpeed { get; init; } = 150f;

    public float StartSpawnInterval { get; init; } = 0.6f;
    public float MinSpawnInterval { get; init; } = 0.15f;

    /// <summary>Seconds shaved off the spawn interval per second survived.</summary>
    public float SpawnIntervalRamp { get; init; } = 0.015f;

    public float StartFallSpeed { get; init; } = 60f;
    public float FallAcceleration { get; init; } = 4f;
    public float MaxFallSpeed { get; init; } = 220f;

    public float MinBlockSize { get; init; } = 8f;
    public float MaxBlockSize { get; init; } = 20f;

    public float PointsPerSecond { get; init; } = 10f;

    internal void Validate(int worldWidth)
    {
        Check.Positive(PlayerWidth, $"Dodge.{nameof(PlayerWidth)}");
        Check.Positive(PlayerHeight, $"Dodge.{nameof(PlayerHeight)}");
        Check.Positive(PlayerSpeed, $"Dodge.{nameof(PlayerSpeed)}");
        Check.Positive(MinSpawnInterval, $"Dodge.{nameof(MinSpawnInterval)}");
        Check.AtLeast(StartSpawnInterval, MinSpawnInterval, $"Dodge.{nameof(StartSpawnInterval)}");
        Check.NonNegative(SpawnIntervalRamp, $"Dodge.{nameof(SpawnIntervalRamp)}");
        Check.Positive(StartFallSpeed, $"Dodge.{nameof(StartFallSpeed)}");
        Check.NonNegative(FallAcceleration, $"Dodge.{nameof(FallAcceleration)}");
        Check.AtLeast(MaxFallSpeed, StartFallSpeed, $"Dodge.{nameof(MaxFallSpeed)}");
        Check.Positive(MinBlockSize, $"Dodge.{nameof(MinBlockSize)}");
        Check.AtLeast(MaxBlockSize, MinBlockSize, $"Dodge.{nameof(MaxBlockSize)}");
        Check.Positive(PointsPerSecond, $"Dodge.{nameof(PointsPerSecond)}");
        Check.AtMost(PlayerWidth, worldWidth, $"Dodge.{nameof(PlayerWidth)}");
        Check.AtMost(MaxBlockSize, worldWidth, $"Dodge.{nameof(MaxBlockSize)}");
    }
}

/// <summary>Collect: catch falling gems in a basket before the clock runs out.</summary>
public sealed record CollectConfig
{
    public float BasketWidth { get; init; } = 26f;
    public float BasketHeight { get; init; } = 6f;
    public float BasketSpeed { get; init; } = 170f;

    public float GemSize { get; init; } = 6f;
    public float SpawnInterval { get; init; } = 0.55f;
    public float MinFallSpeed { get; init; } = 60f;
    public float MaxFallSpeed { get; init; } = 110f;

    public float RoundSeconds { get; init; } = 30f;
    public int MaxMisses { get; init; } = 3;

    internal void Validate(int worldWidth)
    {
        Check.Positive(BasketWidth, $"Collect.{nameof(BasketWidth)}");
        Check.Positive(BasketHeight, $"Collect.{nameof(BasketHeight)}");
        Check.Positive(BasketSpeed, $"Collect.{nameof(BasketSpeed)}");
        Check.Positive(GemSize, $"Collect.{nameof(GemSize)}");
        Check.Positive(SpawnInterval, $"Collect.{nameof(SpawnInterval)}");
        Check.Positive(MinFallSpeed, $"Collect.{nameof(MinFallSpeed)}");
        Check.AtLeast(MaxFallSpeed, MinFallSpeed, $"Collect.{nameof(MaxFallSpeed)}");
        Check.Positive(RoundSeconds, $"Collect.{nameof(RoundSeconds)}");
        Check.Positive(MaxMisses, $"Collect.{nameof(MaxMisses)}");
        Check.AtMost(BasketWidth, worldWidth, $"Collect.{nameof(BasketWidth)}");
        Check.AtMost(GemSize, worldWidth, $"Collect.{nameof(GemSize)}");
    }
}

internal static class Check
{
    public static void Positive(float value, string name)
    {
        if (!float.IsFinite(value) || value <= 0f)
        {
            throw new ArgumentException($"{name} must be a positive number (was {value}).", name);
        }
    }

    public static void NonNegative(float value, string name)
    {
        if (!float.IsFinite(value) || value < 0f)
        {
            throw new ArgumentException($"{name} must be zero or more (was {value}).", name);
        }
    }

    public static void AtLeast(float value, float minimum, string name)
    {
        if (!float.IsFinite(value) || value < minimum)
        {
            throw new ArgumentException($"{name} must be at least {minimum} (was {value}).", name);
        }
    }

    public static void AtMost(float value, float maximum, string name)
    {
        if (value > maximum)
        {
            throw new ArgumentException($"{name} must fit in the {maximum}px-wide screen (was {value}).", name);
        }
    }
}
