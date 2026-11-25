using Platformer.Core.Physics;

namespace Platformer.Core.World;

/// <summary>The box after a move along one axis, and whether a solid tile stopped it.</summary>
public readonly record struct MoveResult(Aabb Box, bool Hit);

/// <summary>
/// Axis-separated swept movement against a <see cref="TileMap"/>.
///
/// Moving X and Y independently is the classic tile-platformer approach: each move only has
/// to answer "how far along this one axis until a wall?", and the result never leaves the
/// box overlapping a tile. Every column (or row) the leading edge crosses is checked, not
/// only the destination, so a fast-moving character cannot tunnel through a thin floor.
/// </summary>
public static class TileCollider
{
    public static MoveResult MoveX(TileMap map, Aabb box, float dx)
    {
        ArgumentNullException.ThrowIfNull(map);
        if (dx == 0f)
        {
            return new MoveResult(box, false);
        }

        var ts = map.TileSize;
        var (row0, row1) = map.CoveredRange(box.Top, box.Bottom);
        if (dx > 0f)
        {
            var first = (int)MathF.Floor(box.Right / ts);
            var last = (int)MathF.Ceiling((box.Right + dx) / ts) - 1;
            for (var col = first; col <= last; col++)
            {
                if (ColumnBlocked(map, col, row0, row1))
                {
                    return new MoveResult(box with { X = col * ts - box.Width }, true);
                }
            }
        }
        else
        {
            var first = (int)MathF.Ceiling(box.Left / ts) - 1;
            var last = (int)MathF.Floor((box.Left + dx) / ts);
            for (var col = first; col >= last; col--)
            {
                if (ColumnBlocked(map, col, row0, row1))
                {
                    return new MoveResult(box with { X = (col + 1) * ts }, true);
                }
            }
        }
        return new MoveResult(box.Offset(dx, 0f), false);
    }

    public static MoveResult MoveY(TileMap map, Aabb box, float dy)
    {
        ArgumentNullException.ThrowIfNull(map);
        if (dy == 0f)
        {
            return new MoveResult(box, false);
        }

        var ts = map.TileSize;
        var (col0, col1) = map.CoveredRange(box.Left, box.Right);
        if (dy > 0f)
        {
            var first = (int)MathF.Floor(box.Bottom / ts);
            var last = (int)MathF.Ceiling((box.Bottom + dy) / ts) - 1;
            for (var row = first; row <= last; row++)
            {
                if (RowBlocked(map, row, col0, col1))
                {
                    return new MoveResult(box with { Y = row * ts - box.Height }, true);
                }
            }
        }
        else
        {
            var first = (int)MathF.Ceiling(box.Top / ts) - 1;
            var last = (int)MathF.Floor((box.Top + dy) / ts);
            for (var row = first; row >= last; row--)
            {
                if (RowBlocked(map, row, col0, col1))
                {
                    return new MoveResult(box with { Y = (row + 1) * ts }, true);
                }
            }
        }
        return new MoveResult(box.Offset(0f, dy), false);
    }

    /// <summary>
    /// Corner correction: when a rising box clips a ceiling corner by only a few pixels,
    /// find the smallest horizontal nudge (up to <paramref name="maxNudge"/> whole pixels)
    /// after which the full upward move <paramref name="dy"/> is clear. Returns null if no
    /// nudge works. Smaller nudges win; at equal size, left is tried before right so the
    /// result is deterministic.
    /// </summary>
    public static float? FindCeilingNudge(TileMap map, Aabb box, float dy, int maxNudge)
    {
        ArgumentNullException.ThrowIfNull(map);
        if (dy >= 0f || maxNudge <= 0)
        {
            return null;
        }
        for (var nudge = 1; nudge <= maxNudge; nudge++)
        {
            foreach (var offset in new float[] { -nudge, nudge })
            {
                var shifted = box.Offset(offset, 0f);
                if (!map.Overlaps(shifted) && !MoveY(map, shifted, dy).Hit)
                {
                    return offset;
                }
            }
        }
        return null;
    }

    private static bool ColumnBlocked(TileMap map, int col, int row0, int row1)
    {
        for (var row = row0; row <= row1; row++)
        {
            if (map.IsSolid(col, row))
            {
                return true;
            }
        }
        return false;
    }

    private static bool RowBlocked(TileMap map, int row, int col0, int col1)
    {
        for (var col = col0; col <= col1; col++)
        {
            if (map.IsSolid(col, row))
            {
                return true;
            }
        }
        return false;
    }
}
