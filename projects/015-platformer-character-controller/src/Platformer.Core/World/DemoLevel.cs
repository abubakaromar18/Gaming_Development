namespace Platformer.Core.World;

/// <summary>
/// The playground level, 40×22 tiles. Each area exercises one controller feature:
/// two four-tile pits whose take-off ledges reward coyote time; a two-tile corridor whose
/// ceiling has a one-tile notch to jump through (corner correction); a staircase of
/// floating platforms, each a three-tile hop up, that needs full-height held jumps; a
/// two-tile step that a tapped jump can't clear (variable height); and landings you can
/// chain into the next jump with an early press (jump buffer). Falling into a pit respawns.
/// </summary>
public static class DemoLevel
{
    public const int TileSize = 16;

    public const string Text = """
        ########################################
        #......................................#
        #......................................#
        #......................................#
        #..................................#####
        #......................................#
        #............................###.......#
        #......................................#
        #......................................#
        #......................####............#
        #......................................#
        #......................................#
        #................####..................#
        #......................................#
        #......................................#
        #...........##.###.....................#
        #.......................####...........#
        #.P.....................####...........#
        #######....#################....########
        #######....#################....########
        #######....#################....########
        #######....#################....########
        """;

    public static Level Load() => Level.Parse(Text, TileSize);
}
