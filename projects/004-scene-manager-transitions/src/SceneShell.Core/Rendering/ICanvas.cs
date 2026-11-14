namespace SceneShell.Core.Rendering;

/// <summary>
/// The only drawing primitive the core needs: axis-aligned filled rectangles on a
/// fixed logical resolution. Text is built from rectangles by <see cref="PixelFont"/>, so
/// the platform layer needs no font assets and tests can record exactly what was drawn.
/// </summary>
public interface ICanvas
{
    /// <summary>Logical width in pixels.</summary>
    int Width { get; }

    /// <summary>Logical height in pixels.</summary>
    int Height { get; }

    /// <summary>Fill a rectangle. <paramref name="color"/> alpha is straight (not premultiplied).</summary>
    void FillRect(float x, float y, float width, float height, Rgba color);
}

/// <summary>An 8-bit RGBA color with straight alpha.</summary>
public readonly record struct Rgba(byte R, byte G, byte B, byte A = 255)
{
    public static readonly Rgba Black = new(0, 0, 0);
    public static readonly Rgba White = new(255, 255, 255);

    /// <summary>The same color with alpha set from <paramref name="alpha"/> in [0, 1].</summary>
    public Rgba WithAlpha(float alpha) =>
        this with { A = (byte)MathF.Round(Math.Clamp(alpha, 0f, 1f) * 255f) };
}
