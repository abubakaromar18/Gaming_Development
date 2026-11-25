using Platformer.Core.Physics;

namespace Platformer.Core.Controller;

public enum ControllerEventKind
{
    /// <summary>Left the ground (or coyote window) under jump power.</summary>
    Jumped,
    /// <summary>Touched down after being airborne. Value is the impact speed in px/s.</summary>
    Landed,
    /// <summary>Jump released while rising; upward speed was cut.</summary>
    JumpCut,
    /// <summary>Rising body hit a ceiling and stopped.</summary>
    HeadBump,
    /// <summary>Rising body was nudged sideways around a ceiling corner. Value is the nudge in px.</summary>
    CornerCorrected,
}

/// <summary>How a jump was triggered; the demo's HUD and tests use this to prove an assist fired.</summary>
[Flags]
public enum JumpAssist
{
    None = 0,
    /// <summary>The character had already left the ledge (coyote time).</summary>
    Coyote = 1,
    /// <summary>The press happened before landing and was remembered (jump buffer).</summary>
    Buffered = 2,
}

/// <summary>
/// Something the controller did this step. The controller only reports facts; squash and
/// stretch, dust and sound react to them elsewhere, which keeps feel effects out of the
/// physics code.
/// </summary>
public readonly record struct ControllerEvent(
    ControllerEventKind Kind,
    Vec2 Feet,
    float Value = 0f,
    JumpAssist Assist = JumpAssist.None);
