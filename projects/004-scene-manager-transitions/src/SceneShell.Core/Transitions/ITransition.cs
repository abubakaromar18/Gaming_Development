using SceneShell.Core.Rendering;

namespace SceneShell.Core.Transitions;

/// <summary>Which half of a scene change is running.</summary>
public enum TransitionPhase
{
    /// <summary>No transition: the current scene owns the screen and the input.</summary>
    None,

    /// <summary>Covering the old scene.</summary>
    Out,

    /// <summary>Uncovering the new scene.</summary>
    In,
}

/// <summary>
/// A stateless description of how a scene change looks. The manager owns the timing and
/// passes in a coverage value, so one transition instance can be shared freely and a
/// transition's drawing is a pure function of (coverage, phase).
/// </summary>
public interface ITransition
{
    /// <summary>Seconds to cover the old scene. Zero swaps immediately.</summary>
    float OutDuration { get; }

    /// <summary>Seconds to reveal the new scene. Zero reveals immediately.</summary>
    float InDuration { get; }

    /// <summary>
    /// Draw the cover over the already-drawn scene.
    /// </summary>
    /// <param name="coverage">0 = scene fully visible, 1 = scene fully hidden.</param>
    /// <param name="phase">Lets directional effects continue across the swap instead of reversing.</param>
    void Draw(ICanvas canvas, float coverage, TransitionPhase phase);
}
