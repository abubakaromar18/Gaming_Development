using Platformer.Core.Physics;

namespace Platformer.Core.World;

/// <summary>
/// A grid of solid or empty square tiles. Outside the grid, the left and right edges act
/// as walls and the space above the map is open; below the map is open too, which is how
/// pits work (the session respawns a character that falls out).
/// </summary>
public sealed class TileMap
{
    private readonly bool[] _solid;

    public TileMap(int width, int height, int tileSize, IReadOnlyList<bool> solid)
    {
        if (width <= 0 || height <= 0)
        {
            throw new ArgumentOutOfRangeException(nameof(width), $"Map must be at least 1×1, got {width}×{height}.");
        }
        if (tileSize <= 0)
        {
            throw new ArgumentOutOfRangeException(nameof(tileSize), tileSize, "Tile size must be positive.");
        }
        ArgumentNullException.ThrowIfNull(solid);
        if (solid.Count != width * height)
        {
            throw new ArgumentException($"Expected {width * height} tiles, got {solid.Count}.", nameof(solid));
        }

        Width = width;
        Height = height;
        TileSize = tileSize;
        _solid = solid.ToArray();
    }

    public int Width { get; }
    public int Height { get; }
    public int TileSize { get; }
    public int PixelWidth => Width * TileSize;
    public int PixelHeight => Height * TileSize;

    public bool IsSolid(int tileX, int tileY)
    {
        if (tileX < 0 || tileX >= Width)
        {
            return true;
        }
        if (tileY < 0 || tileY >= Height)
        {
            return false;
        }
        return _solid[tileY * Width + tileX];
    }

    /// <summary>True if any solid tile intersects the interior of <paramref name="box"/>.</summary>
    public bool Overlaps(Aabb box)
    {
        var (x0, x1) = CoveredRange(box.Left, box.Right);
        var (y0, y1) = CoveredRange(box.Top, box.Bottom);
        for (var ty = y0; ty <= y1; ty++)
        {
            for (var tx = x0; tx <= x1; tx++)
            {
                if (IsSolid(tx, ty))
                {
                    return true;
                }
            }
        }
        return false;
    }

    /// <summary>
    /// Tile indices whose cells intersect the half-open span [min, max). A span that ends
    /// exactly on a tile boundary does not include the tile beyond it.
    /// </summary>
    internal (int First, int Last) CoveredRange(float min, float max) =>
        ((int)MathF.Floor(min / TileSize), (int)MathF.Ceiling(max / TileSize) - 1);
}
