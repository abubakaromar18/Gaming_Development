using SceneShell.Core.Rendering;
using SceneShell.Core.Transitions;

namespace SceneShell.Core.Tests;

public class TransitionTests
{
    [Theory]
    [InlineData(0f)]
    [InlineData(1f)]
    public void Every_easing_curve_lands_exactly_on_its_endpoints(float t)
    {
        Assert.Equal(t, Easing.Linear(t));
        Assert.Equal(t, Easing.SmoothStep(t));
        Assert.Equal(t, Easing.EaseInOutCubic(t));
    }

    [Fact]
    public void Easing_clamps_out_of_range_progress()
    {
        Assert.Equal(0f, Easing.SmoothStep(-3f));
        Assert.Equal(1f, Easing.SmoothStep(7f));
        Assert.Equal(1f, Easing.EaseInOutCubic(2f));
    }

    [Fact]
    public void Smoothstep_is_symmetric_and_gentle_at_the_ends()
    {
        Assert.Equal(0.5, Easing.SmoothStep(0.5f), 5);
        Assert.Equal(1.0, Easing.SmoothStep(0.2f) + Easing.SmoothStep(0.8f), 5);
        // Slow start: at 10% progress the fade is well under 10% opaque.
        Assert.True(Easing.SmoothStep(0.1f) < 0.05f);
    }

    [Fact]
    public void Fade_draws_nothing_when_uncovered_and_an_opaque_screen_when_covered()
    {
        var fade = new FadeTransition(Rgba.Black, 0.3f, 0.3f);
        var canvas = new RecordingCanvas(320, 180);

        fade.Draw(canvas, 0f, TransitionPhase.Out);
        Assert.Empty(canvas.Rects);

        fade.Draw(canvas, 1f, TransitionPhase.In);
        var rect = Assert.Single(canvas.Rects);
        Assert.Equal((0f, 0f, 320f, 180f), (rect.X, rect.Y, rect.W, rect.H));
        Assert.Equal((byte)255, rect.Color.A);
    }

    [Fact]
    public void Fade_alpha_follows_coverage_through_the_easing_curve()
    {
        var fade = new FadeTransition(new Rgba(10, 20, 30), 1f, 1f);
        var canvas = new RecordingCanvas();

        fade.Draw(canvas, 0.5f, TransitionPhase.Out);

        var color = Assert.Single(canvas.Rects).Color;
        Assert.Equal(new Rgba(10, 20, 30, 128), color);
    }

    [Fact]
    public void Wipe_sweeps_left_to_right_across_both_phases()
    {
        Assert.Equal((0f, 0f), WipeTransition.SpanFor(0f, TransitionPhase.Out));
        Assert.Equal((0f, 1f), WipeTransition.SpanFor(1f, TransitionPhase.Out));
        Assert.Equal((0f, 1f), WipeTransition.SpanFor(1f, TransitionPhase.In));
        Assert.Equal((1f, 1f), WipeTransition.SpanFor(0f, TransitionPhase.In));

        // Halfway through the reveal the bar covers the right half: it kept moving right.
        var (start, end) = WipeTransition.SpanFor(0.5f, TransitionPhase.In);
        Assert.Equal(0.5, start, 5);
        Assert.Equal(1f, end);
    }

    [Fact]
    public void Wipe_draws_a_full_height_bar_for_the_covered_span()
    {
        var wipe = new WipeTransition(Rgba.White, 1f, 1f);
        var canvas = new RecordingCanvas(200, 100);

        wipe.Draw(canvas, 0.5f, TransitionPhase.Out);

        var rect = Assert.Single(canvas.Rects);
        Assert.Equal((0f, 0f, 100f, 100f), (rect.X, rect.Y, rect.W, rect.H));
    }

    [Fact]
    public void Cut_is_instant_and_invisible()
    {
        var canvas = new RecordingCanvas();
        CutTransition.Instance.Draw(canvas, 1f, TransitionPhase.Out);

        Assert.Equal(0f, CutTransition.Instance.OutDuration);
        Assert.Equal(0f, CutTransition.Instance.InDuration);
        Assert.Empty(canvas.Rects);
    }

    [Theory]
    [InlineData(-0.1f)]
    [InlineData(float.NaN)]
    [InlineData(float.PositiveInfinity)]
    public void Transitions_reject_invalid_durations(float bad)
    {
        Assert.Throws<ArgumentOutOfRangeException>(() => new FadeTransition(Rgba.Black, bad, 1f));
        Assert.Throws<ArgumentOutOfRangeException>(() => new WipeTransition(Rgba.Black, 1f, bad));
    }

    [Fact]
    public void Rgba_with_alpha_clamps_and_rounds()
    {
        Assert.Equal((byte)0, Rgba.White.WithAlpha(-1f).A);
        Assert.Equal((byte)255, Rgba.White.WithAlpha(2f).A);
        Assert.Equal((byte)64, Rgba.White.WithAlpha(0.25f).A);
        Assert.Equal((byte)255, Rgba.White.WithAlpha(0.25f).R);
    }
}
