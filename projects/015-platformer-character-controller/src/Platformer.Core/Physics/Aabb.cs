namespace Platformer.Core.Physics;

/// <summary>
/// An axis-aligned box given by its top-left corner and size. The box covers the half-open
/// ranges [Left, Right) × [Top, Bottom), so a box whose Bottom sits exactly on a tile row's
/// top edge is touching that row, not overlapping it.
/// </summary>
public readonly record struct Aabb(float X, float Y, float Width, float Height)
{
    public float Left => X;
    public float Top => Y;
    public float Right => X + Width;
    public float Bottom => Y + Height;

    public Aabb Offset(float dx, float dy) => this with { X = X + dx, Y = Y + dy };
}
