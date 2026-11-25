using Platformer.Core.Controller;
using Platformer.Core.Game;
using Platformer.Core.Input;
using Platformer.Core.Physics;
using Platformer.Core.Randomness;
using Platformer.Core.Rendering;
using Platformer.Core.World;

namespace Platformer.Core.Tests;

public class SessionTests
{
    private static PlatformerSession NewSession(Level? level = null, ulong seed = 1) =>
        new(level ?? DemoLevel.Load(), ControllerTuning.Default, new Pcg32(seed));

    [Fact]
    public void A_press_during_a_zero_step_frame_is_delivered_on_the_next_step()
    {
        var session = NewSession(Maps.Room());

        // 1 ms frames: no step is due yet, so the press must be latched.
        Assert.Equal(0, session.Update(0.001f, Sim.Press()));
        Assert.Equal(1, session.Update(0.008f, Sim.Hold(jump: true)));

        Assert.Equal(1, session.Stats.Jumps);
    }

    [Fact]
    public void A_press_in_a_multi_step_frame_fires_only_once()
    {
        var session = NewSession(Maps.Room());

        var steps = session.Update(4f / 120f, Sim.Press());

        Assert.Equal(4, steps);
        Assert.Equal(1, session.Stats.Jumps);
    }

    [Fact]
    public void The_jump_arc_is_identical_at_different_frame_rates()
    {
        static List<Vec2> Arc(float fps)
        {
            var session = NewSession(Maps.Room());
            session.Update(1f / fps, Sim.Press(1f));
            while (session.StepCount < 90)
            {
                session.Update(1f / fps, new ControllerInput(1f, true, false));
            }
            return Enumerable.Range(0, 90).Select(i => session.Trail[i]).ToList();
        }

        Assert.Equal(Arc(30f), Arc(144f));
    }

    [Fact]
    public void Interpolated_feet_stay_between_the_last_two_steps()
    {
        var session = NewSession(Maps.Room());
        session.Update(10f / 120f, Sim.Hold(1f));
        var before = session.Controller.Feet.X;

        session.Update(1.5f / 120f, Sim.Hold(1f));

        Assert.InRange(session.InterpolatedFeet.X, before, session.Controller.Feet.X);
        Assert.True(session.InterpolatedFeet.X < session.Controller.Feet.X);
    }

    [Fact]
    public void Falling_into_a_pit_respawns_and_counts_a_fall()
    {
        var session = NewSession();

        // The demo level has a pit just right of the spawn.
        for (var i = 0; i < 600 && session.Stats.Falls == 0; i++)
        {
            session.Update(1f / 60f, Sim.Hold(1f));
        }

        Assert.Equal(1, session.Stats.Falls);
        Assert.Equal(session.Level.Spawn, session.Controller.Feet);
        Assert.True(session.Trail.Count <= 1);
    }

    [Fact]
    public void Same_seed_and_inputs_give_identical_runs()
    {
        static (Vec2 Feet, float[] Dust, SessionStats Stats) Play(ulong seed)
        {
            var session = NewSession(seed: seed);
            var script = new Pcg32(99);
            for (var frame = 0; frame < 900; frame++)
            {
                var move = script.NextInt(3) - 1;
                var press = script.NextInt(20) == 0;
                session.Update(1f / 60f, new ControllerInput(move, press || script.NextInt(2) == 0, press));
            }
            var dust = session.Dust.Particles.ToArray().SelectMany(p => new[] { p.Position.X, p.Position.Y }).ToArray();
            return (session.Controller.Feet, dust, session.Stats);
        }

        var a = Play(5);
        var b = Play(5);
        var c = Play(6);

        Assert.Equal(a.Feet, b.Feet);
        Assert.Equal(a.Dust, b.Dust);
        Assert.Equal(a.Stats, b.Stats);
        Assert.NotEqual(a.Dust.Sum(), c.Dust.Sum());
    }

    [Fact]
    public void The_body_never_ends_a_step_inside_solid_tiles()
    {
        // Stress test: an hour-long-feeling random walk through the demo level.
        var level = DemoLevel.Load();
        var c = new CharacterController(ControllerTuning.Default, level.Map, level.Spawn);
        var rng = new Pcg32(2024);
        var held = false;
        for (var step = 0; step < 60_000; step++)
        {
            if (rng.NextInt(30) == 0)
            {
                held = !held;
            }
            var move = rng.NextInt(3) - 1;
            c.Step(new ControllerInput(move, held, held && rng.NextInt(4) == 0), Maps.Dt);
            Assert.False(level.Map.Overlaps(c.Body), $"step {step}: body {c.Body} overlaps a tile");
            if (c.Body.Top > level.Map.PixelHeight)
            {
                c.Teleport(level.Spawn);
            }
        }
    }

    [Fact]
    public void Toggling_an_assist_changes_the_live_tuning_and_back()
    {
        var session = NewSession();

        session.ToggleAssist(AssistFeature.CoyoteTime);
        Assert.Equal(0f, session.Controller.Tuning.CoyoteTime);
        Assert.False(session.EnabledAssists.HasFlag(AssistFeature.CoyoteTime));

        session.ToggleAssist(AssistFeature.CoyoteTime);
        Assert.Equal(ControllerTuning.Default, session.Controller.Tuning);
        Assert.Throws<ArgumentOutOfRangeException>(() => session.ToggleAssist(AssistFeature.All));
    }

    [Fact]
    public void Jumping_and_landing_kick_up_dust_and_squash()
    {
        var session = NewSession(Maps.Room());

        session.Update(1f / 120f, Sim.Press());
        Assert.True(session.Dust.AliveCount > 0);
        Assert.True(session.Squash.Scale.Y > 1f);

        for (var i = 0; i < 240; i++)
        {
            session.Update(1f / 120f, ControllerInput.None);
        }
        Assert.True(session.Controller.Grounded);
        Assert.Equal(1.0, session.Squash.Scale.X, 3);
    }

    [Fact]
    public void Renderer_draws_background_first_and_more_with_debug_on()
    {
        var session = NewSession();
        var map = session.Level.Map;
        var plain = new RecordingCanvas(map.PixelWidth, map.PixelHeight);
        var debug = new RecordingCanvas(map.PixelWidth, map.PixelHeight);

        SessionRenderer.Draw(plain, session, debug: false);
        SessionRenderer.Draw(debug, session, debug: true);

        Assert.Equal((0f, 0f, (float)map.PixelWidth, (float)map.PixelHeight, SessionRenderer.Sky), plain.Rects[0]);
        Assert.Contains(plain.Rects, r => r.Color == SessionRenderer.BodyColor(MovementPhase.Grounded));
        Assert.True(debug.Rects.Count > plain.Rects.Count);
    }

    [Fact]
    public void Config_validation_rejects_bad_steps()
    {
        var level = Maps.Room();
        Assert.Throws<ArgumentOutOfRangeException>(() =>
            new PlatformerSession(level, ControllerTuning.Default, new Pcg32(1), new SessionConfig { StepSeconds = 0f }));
        Assert.Throws<ArgumentOutOfRangeException>(() =>
            new PlatformerSession(level, ControllerTuning.Default, new Pcg32(1), new SessionConfig { DustCapacity = 0 }));
    }
}

public class DemoLevelTests
{
    private static readonly Level Demo = DemoLevel.Load();

    /// <summary>Teleport, press jump, then hold the given input until the first landing; returns the landing feet.</summary>
    private static Vec2 JumpAndLand(ControllerTuning tuning, Vec2 feet, ControllerInput hold)
    {
        var c = new CharacterController(tuning, Demo.Map, Demo.Spawn);
        c.Teleport(feet);
        c.Step(new ControllerInput(hold.MoveX, true, true), Maps.Dt);
        for (var i = 0; i < 600; i++)
        {
            c.Step(hold, Maps.Dt);
            if (c.Events.Any(e => e.Kind == ControllerEventKind.Landed))
            {
                return c.Feet;
            }
        }
        throw new InvalidOperationException("never landed");
    }

    [Fact]
    public void The_two_tile_step_needs_a_held_jump_not_a_tap()
    {
        var beforeStep = new Vec2(350f, 288f);

        Assert.Equal(256f, JumpAndLand(ControllerTuning.Default, beforeStep, Sim.Hold(1f, jump: true)).Y);
        Assert.Equal(288f, JumpAndLand(ControllerTuning.Default, beforeStep, Sim.Hold(1f)).Y);
    }

    [Fact]
    public void The_ceiling_notch_is_passable_thanks_to_corner_correction()
    {
        static float HighestHead(ControllerTuning tuning)
        {
            var c = new CharacterController(tuning, Demo.Map, Demo.Spawn);
            // Body x 223..235: clips the ceiling tile left of the one-tile notch by 1 px.
            c.Teleport(new Vec2(229f, 288f));
            c.Step(Sim.Press(), Maps.Dt);
            var highest = c.Body.Top;
            for (var i = 0; i < 120; i++)
            {
                c.Step(Sim.Hold(jump: true), Maps.Dt);
                highest = MathF.Min(highest, c.Body.Top);
            }
            return highest;
        }

        var noCorrection = Assists.Apply(ControllerTuning.Default, AssistFeature.All & ~AssistFeature.CornerCorrection);

        Assert.True(HighestHead(ControllerTuning.Default) < 240f, "head should rise above the ceiling's top");
        Assert.Equal(256f, HighestHead(noCorrection));   // bonks on the ceiling's underside
    }

    [Fact]
    public void Every_floating_platform_is_one_full_jump_above_the_previous_surface()
    {
        // (feet x to stand at, surface y, horizontal input) for each hop of the staircase.
        var hops = new (float X, float FromY, float ToY, float Move)[]
        {
            (260f, 240f, 192f, 1f),   // ceiling top -> first platform
            (326f, 192f, 144f, 1f),   // -> second
            (420f, 144f, 96f, 1f),    // -> third
            (505f, 96f, 64f, 1f),     // -> top ledge
        };

        foreach (var (x, fromY, toY, move) in hops)
        {
            Assert.Equal(toY, JumpAndLand(ControllerTuning.Default, new Vec2(x, fromY), Sim.Hold(move, jump: true)).Y);
        }
    }
}
