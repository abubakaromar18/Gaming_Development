namespace Platformer.Core.Controller;

/// <summary>The forgiveness and feel features, individually switchable for comparison.</summary>
[Flags]
public enum AssistFeature
{
    None = 0,
    CoyoteTime = 1 << 0,
    JumpBuffer = 1 << 1,
    VariableHeight = 1 << 2,
    ApexHang = 1 << 3,
    CornerCorrection = 1 << 4,
    All = CoyoteTime | JumpBuffer | VariableHeight | ApexHang | CornerCorrection,
}

public static class Assists
{
    /// <summary>Display order for the HUD and the 1–5 hotkeys.</summary>
    public static readonly IReadOnlyList<AssistFeature> Ordered = new[]
    {
        AssistFeature.CoyoteTime,
        AssistFeature.JumpBuffer,
        AssistFeature.VariableHeight,
        AssistFeature.ApexHang,
        AssistFeature.CornerCorrection,
    };

    public static string Label(AssistFeature feature) => feature switch
    {
        AssistFeature.CoyoteTime => "COYOTE TIME",
        AssistFeature.JumpBuffer => "JUMP BUFFER",
        AssistFeature.VariableHeight => "VARIABLE HEIGHT",
        AssistFeature.ApexHang => "APEX HANG",
        AssistFeature.CornerCorrection => "CORNER CORRECTION",
        _ => throw new ArgumentOutOfRangeException(nameof(feature), feature, "Not a single assist."),
    };

    /// <summary>
    /// <paramref name="tuning"/> with every feature not in <paramref name="enabled"/> set to
    /// its neutral value. Enabled features keep the values from <paramref name="tuning"/>,
    /// so this always derives from the designer's base tuning and toggles are reversible.
    /// </summary>
    public static ControllerTuning Apply(ControllerTuning tuning, AssistFeature enabled)
    {
        ArgumentNullException.ThrowIfNull(tuning);
        var result = tuning;
        if (!enabled.HasFlag(AssistFeature.CoyoteTime))
        {
            result = result with { CoyoteTime = 0f };
        }
        if (!enabled.HasFlag(AssistFeature.JumpBuffer))
        {
            result = result with { JumpBufferTime = 0f };
        }
        if (!enabled.HasFlag(AssistFeature.VariableHeight))
        {
            result = result with { JumpCutMultiplier = 1f };
        }
        if (!enabled.HasFlag(AssistFeature.ApexHang))
        {
            result = result with { ApexGravityMultiplier = 1f, ApexSpeedBonus = 1f };
        }
        if (!enabled.HasFlag(AssistFeature.CornerCorrection))
        {
            result = result with { CornerCorrection = 0 };
        }
        return result;
    }
}
