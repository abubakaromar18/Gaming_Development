namespace SceneShell.Core.Transitions;

/// <summary>
/// Easing curves map linear progress t ∈ [0, 1] to eased progress. All of them clamp the
/// input and return exactly 0 at t = 0 and 1 at t = 1, so transitions land on fully
/// covered and fully revealed frames.
/// </summary>
public static class Easing
{
    public static float Linear(float t) => Math.Clamp(t, 0f, 1f);

    /// <summary>3t² − 2t³: zero slope at both ends.</summary>
    public static float SmoothStep(float t)
    {
        t = Math.Clamp(t, 0f, 1f);
        return t * t * (3f - 2f * t);
    }

    /// <summary>Cubic ease-in-out: slow start, fast middle, slow end.</summary>
    public static float EaseInOutCubic(float t)
    {
        t = Math.Clamp(t, 0f, 1f);
        if (t < 0.5f)
        {
            return 4f * t * t * t;
        }
        var f = -2f * t + 2f;
        return 1f - f * f * f / 2f;
    }
}
