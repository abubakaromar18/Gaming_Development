using Microsoft.Xna.Framework.Input;
using Platformer.Core.Controller;
using Platformer.Core.Input;

namespace Platformer.Desktop;

/// <summary>Non-gameplay commands from the keyboard, edge-detected.</summary>
public readonly record struct HostCommands(AssistFeature ToggledAssist, bool ToggleDebug, bool Respawn, bool Quit);

/// <summary>
/// Polls keyboard and the first gamepad and turns them into a <see cref="ControllerInput"/>
/// plus host commands. Edge detection (pressed this frame) happens here, so the core only
/// sees clean per-frame input.
/// </summary>
public sealed class DeviceInput
{
    private const float StickDeadzone = 0.2f;

    private static readonly Keys[] JumpKeys = { Keys.Space, Keys.Z, Keys.Up, Keys.W };
    private static readonly Keys[] AssistKeys = { Keys.D1, Keys.D2, Keys.D3, Keys.D4, Keys.D5 };

    private KeyboardState _previousKeys;
    private GamePadState _previousPad;

    public (ControllerInput Input, HostCommands Commands) Poll(KeyboardState keys, GamePadState pad)
    {
        var moveX = 0f;
        if (keys.IsKeyDown(Keys.Left) || keys.IsKeyDown(Keys.A))
        {
            moveX -= 1f;
        }
        if (keys.IsKeyDown(Keys.Right) || keys.IsKeyDown(Keys.D))
        {
            moveX += 1f;
        }
        if (pad.IsConnected)
        {
            // An analog stick beats the keyboard when it is actually being pushed.
            var stick = pad.ThumbSticks.Left.X;
            if (MathF.Abs(stick) > StickDeadzone)
            {
                moveX = stick;
            }
            else if (pad.DPad.Left == ButtonState.Pressed)
            {
                moveX = -1f;
            }
            else if (pad.DPad.Right == ButtonState.Pressed)
            {
                moveX = 1f;
            }
        }

        var jumpHeld = JumpKeys.Any(keys.IsKeyDown) || pad.IsButtonDown(Buttons.A);
        var jumpWasHeld = JumpKeys.Any(_previousKeys.IsKeyDown) || _previousPad.IsButtonDown(Buttons.A);

        var toggled = AssistFeature.None;
        for (var i = 0; i < AssistKeys.Length; i++)
        {
            if (Pressed(keys, AssistKeys[i]))
            {
                toggled = Assists.Ordered[i];
                break;
            }
        }

        var commands = new HostCommands(
            toggled,
            ToggleDebug: Pressed(keys, Keys.F3),
            Respawn: Pressed(keys, Keys.R) || (pad.IsButtonDown(Buttons.Back) && !_previousPad.IsButtonDown(Buttons.Back)),
            Quit: Pressed(keys, Keys.Escape));

        _previousKeys = keys;
        _previousPad = pad;
        return (new ControllerInput(moveX, jumpHeld, jumpHeld && !jumpWasHeld), commands);
    }

    /// <summary>Forget held state, e.g. after the window lost focus, so nothing sticks.</summary>
    public void Reset()
    {
        _previousKeys = default;
        _previousPad = default;
    }

    private bool Pressed(KeyboardState keys, Keys key) => keys.IsKeyDown(key) && !_previousKeys.IsKeyDown(key);
}
