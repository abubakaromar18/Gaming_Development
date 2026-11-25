namespace Platformer.Core.Input;

/// <summary>
/// What the controller reads each simulation step. <see cref="MoveX"/> is an analog axis in
/// [-1, 1] (a keyboard gives -1, 0 or 1). <see cref="JumpPressed"/> is the edge: true only
/// on the step the button went down. <see cref="JumpHeld"/> is the level, used for
/// variable jump height and apex hang.
/// </summary>
public readonly record struct ControllerInput(float MoveX, bool JumpHeld, bool JumpPressed)
{
    public static readonly ControllerInput None = new(0f, false, false);

    /// <summary>The same input with the axis clamped to [-1, 1] and NaN treated as 0.</summary>
    public ControllerInput Sanitized() =>
        this with { MoveX = float.IsNaN(MoveX) ? 0f : Math.Clamp(MoveX, -1f, 1f) };
}

/// <summary>
/// Bridges render frames and fixed simulation steps. A frame can run zero, one or several
/// steps; a press that lands in a zero-step frame would be lost, and a press seen by
/// every step of a multi-step frame would fire twice. The latch keeps the press until
/// exactly one step consumes it, while the axis and held state always reflect the latest
/// frame.
/// </summary>
public sealed class InputLatch
{
    private ControllerInput _latest = ControllerInput.None;
    private bool _pendingPress;

    /// <summary>Record this render frame's input.</summary>
    public void Submit(ControllerInput frame)
    {
        _latest = frame.Sanitized();
        _pendingPress |= frame.JumpPressed;
    }

    /// <summary>The input for the next simulation step; the press edge is delivered once.</summary>
    public ControllerInput NextStep()
    {
        var step = _latest with { JumpPressed = _pendingPress };
        _pendingPress = false;
        return step;
    }

    public void Clear()
    {
        _latest = ControllerInput.None;
        _pendingPress = false;
    }
}
