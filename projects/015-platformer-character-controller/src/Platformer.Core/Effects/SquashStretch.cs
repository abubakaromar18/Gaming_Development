using Platformer.Core.Controller;
using Platformer.Core.Physics;

namespace Platformer.Core.Effects;

/// <summary>
/// Cosmetic scale for the character sprite: stretch tall on take-off, squash wide on a
/// hard landing, then ease back to 1×1. It reacts to controller events only and never
/// touches the collision box, so feel can be exaggerated without changing physics.
/// </summary>
public sealed class SquashStretch
{
    /// <summary>Return rate toward 1×1 (per second, exponential).</summary>
    public const float Recovery = 14f;
    /// <summary>Scale applied on take-off.</summary>
    public static readonly Vec2 JumpScale = new(0.72f, 1.32f);
    /// <summary>Impact speed (px/s) at which a landing squash reaches full strength.</summary>
    public const float FullSquashSpeed = 420f;
    /// <summary>Strongest landing squash, at or above <see cref="FullSquashSpeed"/>.</summary>
    public static readonly Vec2 MaxLandScale = new(1.4f, 0.65f);

    public Vec2 Scale { get; private set; } = Vec2.One;

    public void React(ControllerEvent e)
    {
        switch (e.Kind)
        {
            case ControllerEventKind.Jumped:
                Scale = JumpScale;
                break;
            case ControllerEventKind.Landed:
                var strength = Math.Clamp(e.Value / FullSquashSpeed, 0f, 1f);
                Scale = Vec2.Lerp(Vec2.One, MaxLandScale, strength);
                break;
        }
    }

    public void Update(float dt)
    {
        // Frame-rate independent exponential decay: the same curve at any dt.
        var keep = MathF.Exp(-Recovery * dt);
        Scale = Vec2.One + (Scale - Vec2.One) * keep;
    }

    public void Reset() => Scale = Vec2.One;
}
