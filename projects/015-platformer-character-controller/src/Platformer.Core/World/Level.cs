using Platformer.Core.Physics;

namespace Platformer.Core.World;

/// <summary>A tile map plus where the character starts.</summary>
public sealed record Level(TileMap Map, Vec2 Spawn)
{
    public const char Solid = '#';
    public const char Empty = '.';
    public const char SpawnMarker = 'P';

    /// <summary>
    /// Parse an ASCII level: '#' solid, '.' empty, 'P' the spawn tile (exactly one).
    /// Rows must all be the same width; blank leading/trailing lines are ignored. The spawn
    /// position is the bottom-centre of the 'P' tile, so a character of any size placed
    /// there stands on whatever is below it.
    /// </summary>
    public static Level Parse(string text, int tileSize)
    {
        ArgumentNullException.ThrowIfNull(text);
        var rows = text.Replace("\r", string.Empty, StringComparison.Ordinal)
            .Split('\n')
            .Select(r => r.Trim())
            .SkipWhile(r => r.Length == 0)
            .Reverse().SkipWhile(r => r.Length == 0).Reverse()
            .ToList();
        if (rows.Count == 0)
        {
            throw new FormatException("Level is empty.");
        }

        var width = rows[0].Length;
        var solid = new bool[width * rows.Count];
        Vec2? spawn = null;
        for (var y = 0; y < rows.Count; y++)
        {
            if (rows[y].Length != width)
            {
                throw new FormatException($"Row {y + 1} has {rows[y].Length} tiles, expected {width}.");
            }
            for (var x = 0; x < width; x++)
            {
                switch (rows[y][x])
                {
                    case Solid:
                        solid[y * width + x] = true;
                        break;
                    case Empty:
                        break;
                    case SpawnMarker:
                        if (spawn is not null)
                        {
                            throw new FormatException($"Second spawn 'P' at column {x + 1}, row {y + 1}.");
                        }
                        spawn = new Vec2((x + 0.5f) * tileSize, (y + 1) * tileSize);
                        break;
                    default:
                        throw new FormatException($"Unknown tile '{rows[y][x]}' at column {x + 1}, row {y + 1}.");
                }
            }
        }

        if (spawn is null)
        {
            throw new FormatException("Level has no spawn 'P'.");
        }
        return new Level(new TileMap(width, rows.Count, tileSize, solid), spawn.Value);
    }
}
