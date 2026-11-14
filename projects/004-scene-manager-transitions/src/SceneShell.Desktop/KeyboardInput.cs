using Microsoft.Xna.Framework.Input;
using SceneShell.Core.Input;

namespace SceneShell.Desktop;

/// <summary>
/// Maps physical keys to abstract actions and does the per-frame edge detection, so the
/// core only ever sees <see cref="InputFrame"/>s.
/// </summary>
public sealed class KeyboardInput
{
    private static readonly (Keys Key, GameAction Action)[] Bindings =
    {
        (Keys.Up, GameAction.Up), (Keys.W, GameAction.Up),
        (Keys.Down, GameAction.Down), (Keys.S, GameAction.Down),
        (Keys.Left, GameAction.Left), (Keys.A, GameAction.Left),
        (Keys.Right, GameAction.Right), (Keys.D, GameAction.Right),
        (Keys.Enter, GameAction.Confirm), (Keys.Space, GameAction.Confirm),
        (Keys.Escape, GameAction.Back), (Keys.Back, GameAction.Back),
        (Keys.F3, GameAction.ToggleDebug),
    };

    private GameAction _previous;

    public InputFrame Poll(KeyboardState keyboard)
    {
        var current = GameAction.None;
        foreach (var (key, action) in Bindings)
        {
            if (keyboard.IsKeyDown(key))
            {
                current |= action;
            }
        }
        var frame = InputFrame.FromStates(_previous, current);
        _previous = current;
        return frame;
    }

    /// <summary>Forget held keys, e.g. after the window lost focus, so nothing sticks.</summary>
    public void Reset() => _previous = GameAction.None;
}
