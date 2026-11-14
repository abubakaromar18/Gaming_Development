using SceneShell.Core.Arcade;
using SceneShell.Core.Input;
using SceneShell.Core.Randomness;
using SceneShell.Core.Rendering;

namespace SceneShell.Core.Tests;

public class Pcg32Tests
{
    [Fact]
    public void Matches_the_reference_pcg32_sequence()
    {
        // First outputs of pcg32-demo from pcg-random.org: seed 42, stream 54.
        var rng = new Pcg32(42, 54);
        var expected = new uint[] { 0xa15c02b7, 0x7b47f409, 0xba1d3330, 0x83d2f293, 0xbfa4784b, 0xcbed606e };

        var actual = expected.Select(_ => rng.NextUInt()).ToArray();

        Assert.Equal(expected, actual);
    }

    [Fact]
    public void Same_seed_same_sequence_and_different_streams_differ()
    {
        var a = new Pcg32(7);
        var b = new Pcg32(7);
        var other = new Pcg32(7, stream: 99);

        var sa = Enumerable.Range(0, 32).Select(_ => a.NextUInt()).ToList();
        var sb = Enumerable.Range(0, 32).Select(_ => b.NextUInt()).ToList();
        var so = Enumerable.Range(0, 32).Select(_ => other.NextUInt()).ToList();

        Assert.Equal(sa, sb);
        Assert.NotEqual(sa[0], so[0]);
    }

    [Fact]
    public void Bounded_values_stay_in_range_and_cover_it()
    {
        var rng = new Pcg32(1);
        var seen = new HashSet<int>();
        for (var i = 0; i < 2000; i++)
        {
            var value = rng.NextInt(6);
            Assert.InRange(value, 0, 5);
            seen.Add(value);

            var f = rng.NextFloat();
            Assert.True(f >= 0f && f < 1f, $"NextFloat returned {f}");

            var r = rng.Range(-2f, 3f);
            Assert.True(r >= -2f && r <= 3f, $"Range returned {r}");
        }
        Assert.Equal(6, seen.Count);
    }

    [Fact]
    public void Rejects_empty_ranges()
    {
        var rng = new Pcg32(1);
        Assert.Throws<ArgumentOutOfRangeException>(() => rng.NextInt(0));
        Assert.Throws<ArgumentOutOfRangeException>(() => rng.Range(2f, 1f));
    }
}

public class PixelFontTests
{
    [Fact]
    public void Measures_glyphs_plus_spacing_without_a_trailing_gap()
    {
        Assert.Equal(0f, PixelFont.Measure("", 2));
        Assert.Equal(3f, PixelFont.Measure("A", 1));
        Assert.Equal(14f, PixelFont.Measure("AB", 2));
    }

    [Fact]
    public void Draws_one_scaled_cell_per_inked_pixel()
    {
        var canvas = new RecordingCanvas();

        // 'I' is ### / .#. / .#. / .#. / ### : nine inked cells.
        PixelFont.Draw(canvas, "I", 10, 20, 2, Rgba.White);

        Assert.Equal(9, canvas.Rects.Count);
        Assert.True(canvas.Rects.All(r => r.W == 2f && r.H == 2f));
        Assert.Contains((10f, 20f, 2f, 2f, Rgba.White), canvas.Rects);
        Assert.Contains((12f, 22f, 2f, 2f, Rgba.White), canvas.Rects);
    }

    [Fact]
    public void Lowercase_draws_as_uppercase_and_unknown_characters_as_a_question_mark()
    {
        var lower = new RecordingCanvas();
        var upper = new RecordingCanvas();
        PixelFont.Draw(lower, "score", 0, 0, 1, Rgba.White);
        PixelFont.Draw(upper, "SCORE", 0, 0, 1, Rgba.White);
        Assert.Equal(upper.Rects, lower.Rects);

        var unknown = new RecordingCanvas();
        var question = new RecordingCanvas();
        PixelFont.Draw(unknown, "@", 0, 0, 1, Rgba.White);
        PixelFont.Draw(question, "?", 0, 0, 1, Rgba.White);
        Assert.Equal(question.Rects, unknown.Rects);
    }

    [Fact]
    public void Covers_every_letter_and_digit_the_screens_use()
    {
        foreach (var c in "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 .:-!?<>/")
        {
            Assert.True(PixelFont.Supports(c), $"missing glyph '{c}'");
        }
    }

    [Fact]
    public void Rejects_a_non_positive_scale()
    {
        Assert.Throws<ArgumentOutOfRangeException>(() => PixelFont.Draw(new RecordingCanvas(), "A", 0, 0, 0, Rgba.White));
    }
}

public class InputFrameTests
{
    [Fact]
    public void Pressed_is_the_rising_edge_of_held()
    {
        var frame = InputFrame.FromStates(
            previous: GameAction.Left | GameAction.Confirm,
            current: GameAction.Left | GameAction.Up);

        Assert.True(frame.IsHeld(GameAction.Left));
        Assert.False(frame.WasPressed(GameAction.Left));
        Assert.True(frame.WasPressed(GameAction.Up));
        Assert.False(frame.IsHeld(GameAction.Confirm));
        Assert.False(frame.WasPressed(GameAction.Confirm));
    }

    [Fact]
    public void Press_and_hold_helpers_build_the_expected_frames()
    {
        Assert.True(InputFrame.Press(GameAction.Back).WasPressed(GameAction.Back));
        Assert.True(InputFrame.Hold(GameAction.Right).IsHeld(GameAction.Right));
        Assert.False(InputFrame.Hold(GameAction.Right).WasPressed(GameAction.Right));
    }
}

public class SessionAndConfigTests
{
    [Fact]
    public void Session_tracks_best_scores_per_game()
    {
        var session = new ArcadeSession();

        Assert.True(session.Record(ArcadeGame.Dodge, 40).IsNewBest);
        Assert.False(session.Record(ArcadeGame.Dodge, 25).IsNewBest);
        Assert.False(session.Record(ArcadeGame.Dodge, 40).IsNewBest);
        Assert.True(session.Record(ArcadeGame.Collect, 3).IsNewBest);

        Assert.Equal(40, session.BestFor(ArcadeGame.Dodge));
        Assert.Equal(3, session.BestFor(ArcadeGame.Collect));
        Assert.Equal(new GameResult(ArcadeGame.Collect, 3, true), session.LastResult);
        Assert.Throws<ArgumentOutOfRangeException>(() => session.Record(ArcadeGame.Dodge, -1));
    }

    [Fact]
    public void A_zero_score_never_counts_as_a_new_best()
    {
        Assert.False(new ArcadeSession().Record(ArcadeGame.Collect, 0).IsNewBest);
    }

    [Fact]
    public void Default_config_is_valid()
    {
        ArcadeConfig.Default.Validate();
    }

    [Fact]
    public void Invalid_settings_are_rejected_with_the_setting_name()
    {
        var error = Assert.Throws<ArgumentException>(() => (ArcadeConfig.Default with { FadeSeconds = 0f }).Validate());
        Assert.Contains("FadeSeconds", error.Message);

        error = Assert.Throws<ArgumentException>(() =>
            (ArcadeConfig.Default with { Dodge = new DodgeConfig { MinBlockSize = 30f, MaxBlockSize = 10f } }).Validate());
        Assert.Contains("Dodge.MaxBlockSize", error.Message);

        error = Assert.Throws<ArgumentException>(() =>
            (ArcadeConfig.Default with { Collect = new CollectConfig { BasketWidth = 999f } }).Validate());
        Assert.Contains("Collect.BasketWidth", error.Message);
    }
}
