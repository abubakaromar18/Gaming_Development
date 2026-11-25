using Platformer.Core.Controller;
using Platformer.Core.Effects;
using Platformer.Core.Input;
using Platformer.Core.Physics;
using Platformer.Core.Randomness;
using Platformer.Core.Rendering;
using Platformer.Core.Simulation;
using Platformer.Core.World;

namespace Platformer.Core.Tests;

public class FixedStepperTests
{
    [Fact]
    public void Converts_frame_time_into_whole_steps_and_keeps_the_remainder()
    {
        var stepper = new FixedStepper(0.01f, 8);

        Assert.Equal(0, stepper.Advance(0.004f));
        Assert.Equal(1, stepper.Advance(0.0075f));
        Assert.InRange(stepper.Alpha, 0.14f, 0.16f);
        Assert.Equal(3, stepper.Advance(0.03f));
    }

    [Fact]
    public void A_huge_frame_is_clamped_to_the_step_budget()
    {
        var stepper = new FixedStepper(0.01f, 5);

        Assert.Equal(5, stepper.Advance(10f));
        Assert.InRange(stepper.Alpha, 0f, 1f);
        Assert.Equal(0, stepper.Advance(0f));
    }

    [Fact]
    public void Rejects_invalid_arguments()
    {
        Assert.Throws<ArgumentOutOfRangeException>(() => new FixedStepper(0f, 1));
        Assert.Throws<ArgumentOutOfRangeException>(() => new FixedStepper(0.01f, 0));
        Assert.Throws<ArgumentOutOfRangeException>(() => new FixedStepper(0.01f, 1).Advance(-1f));
    }
}

public class InputLatchTests
{
    [Fact]
    public void Press_is_held_until_one_step_consumes_it()
    {
        var latch = new InputLatch();
        latch.Submit(new ControllerInput(1f, true, true));
        latch.Submit(new ControllerInput(1f, true, false));

        Assert.True(latch.NextStep().JumpPressed);
        var second = latch.NextStep();
        Assert.False(second.JumpPressed);
        Assert.True(second.JumpHeld);
        Assert.Equal(1f, second.MoveX);
    }

    [Fact]
    public void Clear_drops_pending_presses_and_held_state()
    {
        var latch = new InputLatch();
        latch.Submit(new ControllerInput(-1f, true, true));

        latch.Clear();

        Assert.Equal(ControllerInput.None, latch.NextStep());
    }
}

public class EffectsTests
{
    [Fact]
    public void Squash_stretches_on_jump_and_relaxes_back()
    {
        var squash = new SquashStretch();
        squash.React(new ControllerEvent(ControllerEventKind.Jumped, Vec2.Zero));
        Assert.Equal(SquashStretch.JumpScale, squash.Scale);

        for (var i = 0; i < 120; i++)
        {
            squash.Update(1f / 120f);
        }

        Assert.Equal(1.0, squash.Scale.X, 3);
        Assert.Equal(1.0, squash.Scale.Y, 3);
    }

    [Fact]
    public void Harder_landings_squash_more_up_to_a_limit()
    {
        static Vec2 After(float impact)
        {
            var s = new SquashStretch();
            s.React(new ControllerEvent(ControllerEventKind.Landed, Vec2.Zero, impact));
            return s.Scale;
        }

        Assert.True(After(100f).Y > After(300f).Y);
        Assert.Equal(SquashStretch.MaxLandScale, After(SquashStretch.FullSquashSpeed));
        Assert.Equal(SquashStretch.MaxLandScale, After(5000f));
    }

    [Fact]
    public void Squash_decay_is_frame_rate_independent()
    {
        var a = new SquashStretch();
        var b = new SquashStretch();
        a.React(new ControllerEvent(ControllerEventKind.Jumped, Vec2.Zero));
        b.React(new ControllerEvent(ControllerEventKind.Jumped, Vec2.Zero));

        a.Update(0.1f);
        for (var i = 0; i < 10; i++)
        {
            b.Update(0.01f);
        }

        Assert.Equal(a.Scale.Y, b.Scale.Y, 4);
    }

    [Fact]
    public void Dust_pool_recycles_and_particles_expire()
    {
        var dust = new DustSystem(8, new Pcg32(3));

        dust.Burst(Vec2.Zero, 12, 50f);
        Assert.Equal(8, dust.AliveCount);

        for (var i = 0; i < 60; i++)
        {
            dust.Update(1f / 120f);
        }
        Assert.Equal(0, dust.AliveCount);
        Assert.Throws<ArgumentOutOfRangeException>(() => dust.Burst(Vec2.Zero, -1, 1f));
    }

    [Fact]
    public void Dust_is_reproducible_from_the_seed()
    {
        static float[] Positions(ulong seed)
        {
            var dust = new DustSystem(16, new Pcg32(seed));
            dust.Burst(new Vec2(10f, 10f), 6, 50f);
            dust.Update(0.05f);
            return dust.Particles.ToArray().Select(p => p.Position.X).ToArray();
        }

        Assert.Equal(Positions(4), Positions(4));
        Assert.NotEqual(Positions(4)[0], Positions(5)[0]);
    }

    [Fact]
    public void Trail_keeps_the_newest_points_oldest_first()
    {
        var trail = new TrailBuffer(3);
        for (var i = 1; i <= 5; i++)
        {
            trail.Add(new Vec2(i, 0f));
        }

        Assert.Equal(3, trail.Count);
        Assert.Equal(new[] { 3f, 4f, 5f }, Enumerable.Range(0, 3).Select(i => trail[i].X).ToArray());
        Assert.Throws<ArgumentOutOfRangeException>(() => _ = trail[3]);

        trail.Clear();
        Assert.Equal(0, trail.Count);
    }
}

public class LevelTests
{
    [Fact]
    public void Parses_tiles_and_puts_the_spawn_at_the_bottom_centre_of_its_tile()
    {
        var level = Level.Parse("""

            #..#
            #P.#
            ####

            """, 16);

        Assert.Equal(4, level.Map.Width);
        Assert.Equal(3, level.Map.Height);
        Assert.True(level.Map.IsSolid(0, 0));
        Assert.False(level.Map.IsSolid(1, 1));
        Assert.Equal(new Vec2(24f, 32f), level.Spawn);
    }

    [Theory]
    [InlineData("", "empty")]
    [InlineData("#..\n#.", "Row 2")]
    [InlineData("#x#\n#P#", "Unknown tile 'x'")]
    [InlineData("P.P", "Second spawn")]
    [InlineData("#..#", "no spawn")]
    public void Rejects_malformed_levels_with_a_useful_message(string text, string expected)
    {
        var error = Assert.Throws<FormatException>(() => Level.Parse(text, 16));

        Assert.Contains(expected, error.Message);
    }

    [Fact]
    public void Demo_level_loads_with_a_valid_spawn()
    {
        var level = DemoLevel.Load();
        var c = new CharacterController(ControllerTuning.Default, level.Map, level.Spawn);

        Assert.True(c.Grounded);
        Assert.Equal(40, level.Map.Width);
    }
}

public class Pcg32Tests
{
    [Fact]
    public void Matches_the_reference_pcg32_sequence()
    {
        // First outputs of pcg32-demo from pcg-random.org: seed 42, stream 54.
        var rng = new Pcg32(42, 54);
        var expected = new uint[] { 0xa15c02b7, 0x7b47f409, 0xba1d3330, 0x83d2f293, 0xbfa4784b, 0xcbed606e };

        Assert.Equal(expected, expected.Select(_ => rng.NextUInt()).ToArray());
    }

    [Fact]
    public void Bounded_values_stay_in_range()
    {
        var rng = new Pcg32(1);
        for (var i = 0; i < 1000; i++)
        {
            Assert.InRange(rng.NextInt(3), 0, 2);
            Assert.InRange(rng.Range(-2f, 2f), -2f, 2f);
        }
        Assert.Throws<ArgumentOutOfRangeException>(() => rng.NextInt(0));
    }
}

public class PixelFontTests
{
    [Fact]
    public void Hud_strings_only_use_supported_glyphs()
    {
        var hud = "ARROWS/AD MOVE  SPACE JUMP  1-5 TOGGLE  F3 DEBUG  R RESET" +
                  string.Concat(Assists.Ordered.Select(Assists.Label)) + "ON OFF JUMPS COYOTE BUFFERED CUT CORNER FALLS PHASE VX VY G 0123456789-.";

        Assert.All(hud, c => Assert.True(PixelFont.Supports(c), $"'{c}' has no glyph"));
        Assert.Equal(7f, PixelFont.Measure("AB", 1f));
    }
}
