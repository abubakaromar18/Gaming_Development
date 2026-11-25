using Platformer.Core.World;

namespace Platformer.Core.Tests;

public class TileColliderTests
{
    private static readonly TileMap Map = Maps.Parse(
        "#........#",
        "#........#",
        "#...#....#",
        "#.P......#",
        "##########").Map;

    [Fact]
    public void Falling_onto_the_floor_stops_flush_with_its_top()
    {
        var box = Boxes.At(20, 40);

        var result = TileCollider.MoveY(Map, box, 30f);

        Assert.True(result.Hit);
        Assert.Equal(64f, result.Box.Bottom);
    }

    [Fact]
    public void A_huge_fall_cannot_tunnel_through_a_one_tile_floor()
    {
        var box = Boxes.At(20, 10);

        var result = TileCollider.MoveY(Map, box, 5000f);

        Assert.True(result.Hit);
        Assert.Equal(64f, result.Box.Bottom);
    }

    [Fact]
    public void Sliding_along_the_floor_does_not_snag_on_tile_seams()
    {
        // Bottom sits exactly on the floor's top edge: touching, not overlapping.
        var box = Boxes.At(20, 50);

        var result = TileCollider.MoveX(Map, box, 40f);

        Assert.False(result.Hit);
        Assert.Equal(60f, result.Box.X);
    }

    [Fact]
    public void Moving_into_a_wall_stops_flush_against_it()
    {
        var box = Boxes.At(20, 34);

        var right = TileCollider.MoveX(Map, box, 200f);
        var left = TileCollider.MoveX(Map, box, -200f);

        Assert.True(right.Hit);
        Assert.Equal(64f, right.Box.Right);   // the '#' at column 4, row 2
        Assert.True(left.Hit);
        Assert.Equal(16f, left.Box.Left);     // the left wall
    }

    [Fact]
    public void Box_already_touching_a_wall_reports_a_hit_without_moving()
    {
        var box = Boxes.At(52, 34);

        var result = TileCollider.MoveX(Map, box, 0.25f);

        Assert.True(result.Hit);
        Assert.Equal(52f, result.Box.X);
    }

    [Fact]
    public void Rising_into_a_ceiling_stops_under_it()
    {
        var box = Boxes.At(66, 50);

        var result = TileCollider.MoveY(Map, box, -30f);

        Assert.True(result.Hit);
        Assert.Equal(48f, result.Box.Top);
    }

    [Fact]
    public void Zero_moves_are_no_ops()
    {
        var box = Boxes.At(20, 30);

        Assert.Equal(new MoveResult(box, false), TileCollider.MoveX(Map, box, 0f));
        Assert.Equal(new MoveResult(box, false), TileCollider.MoveY(Map, box, 0f));
    }

    [Fact]
    public void Map_edges_are_walls_at_the_sides_and_open_above_and_below()
    {
        var map = Maps.Parse("..P..").Map;

        Assert.True(map.IsSolid(-1, 0));
        Assert.True(map.IsSolid(5, 0));
        Assert.False(map.IsSolid(2, -1));
        Assert.False(map.IsSolid(2, 1));
    }

    [Fact]
    public void Ceiling_nudge_finds_the_smallest_clear_offset()
    {
        var notch = Maps.Notch().Map;
        // Overlaps the solid column 3 (x < 64) by 2 px while rising toward the gap.
        var box = Boxes.At(62, 50);

        Assert.True(TileCollider.MoveY(notch, box, -4f).Hit);
        Assert.Equal(2f, TileCollider.FindCeilingNudge(notch, box, -4f, 4));
        Assert.Null(TileCollider.FindCeilingNudge(notch, box, -4f, 1));
        Assert.Null(TileCollider.FindCeilingNudge(notch, box, 4f, 4));
    }

    [Fact]
    public void Ceiling_nudge_refuses_when_the_whole_box_is_under_solid_ceiling()
    {
        var notch = Maps.Notch().Map;
        var box = Boxes.At(30, 50);

        Assert.Null(TileCollider.FindCeilingNudge(notch, box, -4f, 4));
    }

    [Fact]
    public void Map_constructor_rejects_bad_dimensions()
    {
        Assert.Throws<ArgumentOutOfRangeException>(() => new TileMap(0, 1, 16, Array.Empty<bool>()));
        Assert.Throws<ArgumentOutOfRangeException>(() => new TileMap(1, 1, 0, new[] { false }));
        Assert.Throws<ArgumentException>(() => new TileMap(2, 2, 16, new[] { false }));
    }
}
