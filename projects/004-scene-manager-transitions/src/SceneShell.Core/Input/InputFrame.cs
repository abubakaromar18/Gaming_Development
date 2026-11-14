namespace SceneShell.Core.Input;

/// <summary>Abstract actions. The platform layer decides which keys or buttons produce them.</summary>
[Flags]
public enum GameAction
{
    None = 0,
    Up = 1 << 0,
    Down = 1 << 1,
    Left = 1 << 2,
    Right = 1 << 3,
    Confirm = 1 << 4,
    Back = 1 << 5,
    ToggleDebug = 1 << 6,
}

/// <summary>
/// One frame of input: what is held down, and what went down this frame. Edge detection
/// happens once in the platform layer, so scenes never track "was it pressed last frame".
/// </summary>
public readonly record struct InputFrame(GameAction Held, GameAction Pressed)
{
    public static readonly InputFrame None = new(GameAction.None, GameAction.None);

    public bool IsHeld(GameAction action) => (Held & action) != 0;

    public bool WasPressed(GameAction action) => (Pressed & action) != 0;

    /// <summary>A frame where <paramref name="action"/> went down this frame (and is therefore held).</summary>
    public static InputFrame Press(GameAction action) => new(action, action);

    /// <summary>A frame where <paramref name="action"/> is held but was already down last frame.</summary>
    public static InputFrame Hold(GameAction action) => new(action, GameAction.None);

    /// <summary>Build the frame for <paramref name="current"/> given what was held last frame.</summary>
    public static InputFrame FromStates(GameAction previous, GameAction current) =>
        new(current, current & ~previous);
}
