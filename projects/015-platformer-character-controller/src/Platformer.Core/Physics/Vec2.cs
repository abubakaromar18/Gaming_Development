namespace Platformer.Core.Physics;

/// <summary>
/// A 2D vector in world pixels. The core defines its own instead of using MonoGame's
/// Vector2 so it has no framework dependency. +Y points down, matching screen space.
/// </summary>
public readonly record struct Vec2(float X, float Y)
{
    public static readonly Vec2 Zero = new(0f, 0f);
    public static readonly Vec2 One = new(1f, 1f);

    public static Vec2 operator +(Vec2 a, Vec2 b) => new(a.X + b.X, a.Y + b.Y);
    public static Vec2 operator -(Vec2 a, Vec2 b) => new(a.X - b.X, a.Y - b.Y);
    public static Vec2 operator *(Vec2 v, float s) => new(v.X * s, v.Y * s);

    /// <summary>Linear interpolation; <paramref name="t"/> = 0 gives <paramref name="a"/>.</summary>
    public static Vec2 Lerp(Vec2 a, Vec2 b, float t) => new(a.X + (b.X - a.X) * t, a.Y + (b.Y - a.Y) * t);

    public override string ToString() => $"({X:0.###}, {Y:0.###})";
}
