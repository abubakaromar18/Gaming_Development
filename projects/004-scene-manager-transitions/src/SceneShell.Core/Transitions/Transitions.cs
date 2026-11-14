using SceneShell.Core.Rendering;

namespace SceneShell.Core.Transitions;

/// <summary>An instant swap: no cover, no delay.</summary>
public sealed class CutTransition : ITransition
{
    public static readonly CutTransition Instance = new();

    private CutTransition() { }

    public float OutDuration => 0f;
    public float InDuration => 0f;

    public void Draw(ICanvas canvas, float coverage, TransitionPhase phase) { }
}

/// <summary>
/// Fade through a solid color, the classic "dip to black". The alpha follows a smoothstep
/// curve so the fade starts and lands gently instead of snapping at the ends.
/// </summary>
public sealed class FadeTransition : ITransition
{
    public FadeTransition(Rgba color, float outDuration, float inDuration)
    {
        Color = color;
        OutDuration = Durations.Validate(outDuration, nameof(outDuration));
        InDuration = Durations.Validate(inDuration, nameof(inDuration));
    }

    public Rgba Color { get; }
    public float OutDuration { get; }
    public float InDuration { get; }

    /// <summary>The alpha (0..1) drawn for a given coverage. Exposed so the curve is testable.</summary>
    public static float AlphaFor(float coverage) => Easing.SmoothStep(coverage);

    public void Draw(ICanvas canvas, float coverage, TransitionPhase phase)
    {
        var alpha = AlphaFor(coverage);
        if (alpha <= 0f)
        {
            return;
        }
        canvas.FillRect(0, 0, canvas.Width, canvas.Height, Color.WithAlpha(alpha));
    }
}

/// <summary>
/// A solid bar that sweeps across the screen left to right. During Out the bar grows from
/// the left edge; during In it keeps moving and leaves through the right edge, so the
/// motion reads as one continuous wipe instead of reversing at the swap.
/// </summary>
public sealed class WipeTransition : ITransition
{
    public WipeTransition(Rgba color, float outDuration, float inDuration)
    {
        Color = color;
        OutDuration = Durations.Validate(outDuration, nameof(outDuration));
        InDuration = Durations.Validate(inDuration, nameof(inDuration));
    }

    public Rgba Color { get; }
    public float OutDuration { get; }
    public float InDuration { get; }

    /// <summary>The covered span as fractions of the width: [start, end) with 0 = left edge.</summary>
    public static (float Start, float End) SpanFor(float coverage, TransitionPhase phase)
    {
        var eased = Easing.EaseInOutCubic(Math.Clamp(coverage, 0f, 1f));
        return phase == TransitionPhase.In
            ? (1f - eased, 1f)   // the bar's left edge moves right as coverage falls
            : (0f, eased);       // the bar's right edge moves right as coverage rises
    }

    public void Draw(ICanvas canvas, float coverage, TransitionPhase phase)
    {
        var (start, end) = SpanFor(coverage, phase);
        var width = (end - start) * canvas.Width;
        if (width <= 0f)
        {
            return;
        }
        canvas.FillRect(start * canvas.Width, 0, width, canvas.Height, Color);
    }
}

internal static class Durations
{
    public static float Validate(float seconds, string paramName) =>
        float.IsFinite(seconds) && seconds >= 0f
            ? seconds
            : throw new ArgumentOutOfRangeException(paramName, seconds, "Durations must be finite and non-negative.");
}
