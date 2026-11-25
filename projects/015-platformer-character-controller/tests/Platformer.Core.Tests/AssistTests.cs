using Platformer.Core.Controller;
using Platformer.Core.Input;

namespace Platformer.Core.Tests;

public class CoyoteTimeTests
{
    /// <summary>Run right off the ledge; return the controller on the first airborne step.</summary>
    private static CharacterController RunOffLedge(ControllerTuning tuning)
    {
        var c = Sim.Controller(Maps.Ledge(), tuning);
        for (var i = 0; i < 240 && c.Grounded; i++)
        {
            c.Step(Sim.Hold(1f), Maps.Dt);
        }
        Assert.False(c.Grounded);
        return c;
    }

    [Theory]
    [InlineData(1)]
    [InlineData(6)]
    [InlineData(11)]
    public void Jump_pressed_shortly_after_leaving_a_ledge_still_fires(int stepsLate)
    {
        var c = RunOffLedge(ControllerTuning.Default);
        Sim.Run(c, Sim.Hold(1f), stepsLate - 1);

        var events = Sim.Run(c, Sim.Press(1f), 1);

        var jump = Assert.Single(events.Where(e => e.Kind == ControllerEventKind.Jumped));
        Assert.Equal(JumpAssist.Coyote, jump.Assist);
        Assert.True(c.Velocity.Y < 0f);
    }

    [Fact]
    public void Jump_pressed_after_the_coyote_window_does_nothing()
    {
        var c = RunOffLedge(ControllerTuning.Default);
        var windowSteps = (int)MathF.Ceiling(ControllerTuning.Default.CoyoteTime / Maps.Dt);
        Sim.Run(c, Sim.Hold(1f), windowSteps);

        var events = Sim.Run(c, Sim.Press(1f), 1);

        Assert.DoesNotContain(ControllerEventKind.Jumped, events.Select(e => e.Kind));
    }

    [Fact]
    public void Without_coyote_time_the_first_airborne_step_is_too_late()
    {
        var tuning = Assists.Apply(ControllerTuning.Default, AssistFeature.All & ~AssistFeature.CoyoteTime);
        var c = RunOffLedge(tuning);

        var events = Sim.Run(c, Sim.Press(1f), 1);

        Assert.DoesNotContain(ControllerEventKind.Jumped, events.Select(e => e.Kind));
    }

    [Fact]
    public void Jumping_consumes_coyote_time_so_it_cannot_double_jump()
    {
        var c = RunOffLedge(ControllerTuning.Default);
        Sim.Run(c, Sim.Press(1f), 1);

        var events = Sim.Run(c, Sim.Press(1f), 1);

        Assert.DoesNotContain(ControllerEventKind.Jumped, events.Select(e => e.Kind));
    }
}

public class JumpBufferTests
{
    /// <summary>Steps from spawn until the mid-air character lands, with no input.</summary>
    private static int StepsToLand(ControllerTuning tuning)
    {
        var c = Sim.Controller(Maps.MidAir(), tuning);
        var steps = 0;
        while (!c.Grounded)
        {
            c.Step(ControllerInput.None, Maps.Dt);
            steps++;
        }
        return steps;
    }

    /// <summary>Fall, press jump <paramref name="stepsEarly"/> steps before the landing step, then keep holding.</summary>
    private static List<ControllerEvent> PressBeforeLanding(ControllerTuning tuning, int stepsEarly)
    {
        var landing = StepsToLand(tuning);
        var c = Sim.Controller(Maps.MidAir(), tuning);
        var events = new List<ControllerEvent>();
        for (var step = 1; step <= landing + 2; step++)
        {
            var input = step == landing - stepsEarly ? Sim.Press() : Sim.Hold(jump: step > landing - stepsEarly);
            c.Step(input, Maps.Dt);
            events.AddRange(c.Events);
        }
        return events;
    }

    [Theory]
    [InlineData(1)]
    [InlineData(5)]
    [InlineData(13)]
    public void Jump_pressed_just_before_landing_fires_on_touchdown(int stepsEarly)
    {
        var events = PressBeforeLanding(ControllerTuning.Default, stepsEarly);

        var kinds = events.Select(e => e.Kind).ToList();
        var jump = Assert.Single(events.Where(e => e.Kind == ControllerEventKind.Jumped));
        Assert.Equal(JumpAssist.Buffered, jump.Assist);
        Assert.True(kinds.IndexOf(ControllerEventKind.Landed) < kinds.IndexOf(ControllerEventKind.Jumped));
    }

    [Fact]
    public void Jump_pressed_long_before_landing_is_forgotten()
    {
        var events = PressBeforeLanding(ControllerTuning.Default, stepsEarly: 20);

        Assert.DoesNotContain(ControllerEventKind.Jumped, events.Select(e => e.Kind));
    }

    [Fact]
    public void Without_buffering_an_early_press_is_dropped_but_a_grounded_press_still_works()
    {
        var tuning = Assists.Apply(ControllerTuning.Default, AssistFeature.All & ~AssistFeature.JumpBuffer);

        Assert.DoesNotContain(ControllerEventKind.Jumped, PressBeforeLanding(tuning, 1).Select(e => e.Kind));

        var c = Sim.Controller(Maps.Room(), tuning);
        Assert.Contains(ControllerEventKind.Jumped, Sim.Run(c, Sim.Press(), 1).Select(e => e.Kind));
    }
}

public class VariableHeightTests
{
    [Fact]
    public void Tapping_jump_gives_a_much_lower_hop_than_holding_it()
    {
        var full = Sim.Jump(Sim.Controller(Maps.Room()), holdSteps: 1000);
        var tap = Sim.Jump(Sim.Controller(Maps.Room()), holdSteps: 1);

        Assert.True(tap.Peak < full.Peak * 0.35f, $"tap {tap.Peak} vs full {full.Peak}");
        Assert.True(tap.AirSteps < full.AirSteps);
    }

    [Fact]
    public void Releasing_later_gives_a_higher_jump()
    {
        var peaks = new[] { 1, 8, 16, 24 }
            .Select(hold => Sim.Jump(Sim.Controller(Maps.Room()), hold).Peak)
            .ToList();

        Assert.Equal(peaks.OrderBy(p => p).ToList(), peaks);
        Assert.Equal(peaks.Count, peaks.Distinct().Count());
    }

    [Fact]
    public void The_cut_happens_once_and_only_while_rising()
    {
        var c = Sim.Controller(Maps.Room());
        c.Step(Sim.Press(), Maps.Dt);

        var events = Sim.Run(c, Sim.Hold(), 200);

        Assert.Single(events.Where(e => e.Kind == ControllerEventKind.JumpCut));
    }

    [Fact]
    public void With_variable_height_off_every_jump_is_full_height()
    {
        // Apex hang also rewards holding, so switch it off too to isolate the jump cut.
        var tuning = Assists.Apply(ControllerTuning.Default, AssistFeature.CoyoteTime | AssistFeature.JumpBuffer);

        var full = Sim.Jump(Sim.Controller(Maps.Room(), tuning), holdSteps: 1000);
        var tap = Sim.Jump(Sim.Controller(Maps.Room(), tuning), holdSteps: 1);

        Assert.Equal(full.Peak, tap.Peak);
        Assert.Equal(full.AirSteps, tap.AirSteps);
    }
}

public class ApexHangTests
{
    private static readonly ControllerTuning NoHang =
        Assists.Apply(ControllerTuning.Default, AssistFeature.All & ~AssistFeature.ApexHang);

    [Fact]
    public void Apex_hang_lengthens_air_time_of_a_held_jump()
    {
        var hang = Sim.Jump(Sim.Controller(Maps.Room()), holdSteps: 1000);
        var plain = Sim.Jump(Sim.Controller(Maps.Room(), NoHang), holdSteps: 1000);

        Assert.True(hang.AirSteps > plain.AirSteps, $"hang {hang.AirSteps} vs plain {plain.AirSteps}");
        Assert.True(hang.Peak > plain.Peak);
    }

    [Fact]
    public void Apex_phase_is_reported_only_near_the_top_of_a_held_jump()
    {
        var c = Sim.Controller(Maps.Room());
        c.Step(Sim.Press(), Maps.Dt);
        var phases = new List<MovementPhase> { c.Phase };
        while (!c.Grounded)
        {
            c.Step(Sim.Hold(jump: true), Maps.Dt);
            phases.Add(c.Phase);
        }

        var firstApex = phases.IndexOf(MovementPhase.Apex);
        var lastApex = phases.LastIndexOf(MovementPhase.Apex);
        Assert.True(firstApex > 0);
        Assert.All(phases.Take(firstApex), p => Assert.Equal(MovementPhase.Rising, p));
        Assert.Contains(MovementPhase.Falling, phases.Skip(lastApex));
    }

    [Fact]
    public void Releasing_jump_skips_the_hang()
    {
        var c = Sim.Controller(Maps.Room());
        c.Step(Sim.Press(), Maps.Dt);

        Sim.Run(c, Sim.Hold(), 200);

        Assert.False(c.InApex);
    }

    [Fact]
    public void Apex_speed_bonus_raises_max_air_speed_briefly()
    {
        var c = Sim.Controller(Maps.Room());
        Sim.Run(c, Sim.Hold(1f), 30);
        c.Step(Sim.Press(1f), Maps.Dt);
        var maxSpeed = 0f;
        while (!c.Grounded)
        {
            c.Step(new ControllerInput(1f, true, false), Maps.Dt);
            maxSpeed = MathF.Max(maxSpeed, c.Velocity.X);
        }

        Assert.True(maxSpeed > ControllerTuning.Default.MaxRunSpeed);
        Assert.True(maxSpeed <= ControllerTuning.Default.MaxRunSpeed * ControllerTuning.Default.ApexSpeedBonus + 0.001f);
    }
}

public class CornerCorrectionTests
{
    private static (CharacterController C, List<ControllerEvent> Events) JumpUnderCorner(ControllerTuning tuning)
    {
        var level = Maps.Notch();
        var c = new CharacterController(tuning, level.Map, level.Spawn);
        // Body x 62..74 clips the solid column ending at x 64 by 2 px.
        c.Teleport(new Physics.Vec2(68f, level.Spawn.Y));
        var events = Sim.Run(c, Sim.Press(), 1).Concat(Sim.Run(c, Sim.Hold(jump: true), 45)).ToList();
        return (c, events);
    }

    [Fact]
    public void A_head_that_clips_a_corner_slides_past_it()
    {
        var (c, events) = JumpUnderCorner(ControllerTuning.Default);

        var nudge = Assert.Single(events.Where(e => e.Kind == ControllerEventKind.CornerCorrected));
        Assert.Equal(2f, nudge.Value);
        Assert.DoesNotContain(ControllerEventKind.HeadBump, events.Select(e => e.Kind));
        Assert.True(c.Body.Bottom < 48f, "the body should have risen through the gap");
    }

    [Fact]
    public void Without_corner_correction_the_same_jump_bonks()
    {
        var tuning = Assists.Apply(ControllerTuning.Default, AssistFeature.All & ~AssistFeature.CornerCorrection);

        var (c, events) = JumpUnderCorner(tuning);

        Assert.Contains(ControllerEventKind.HeadBump, events.Select(e => e.Kind));
        Assert.True(c.Body.Top >= 48f);
    }
}

public class AssistToggleTests
{
    [Fact]
    public void Disabling_everything_sets_neutral_values()
    {
        var t = Assists.Apply(ControllerTuning.Default, AssistFeature.None);

        Assert.Equal(0f, t.CoyoteTime);
        Assert.Equal(0f, t.JumpBufferTime);
        Assert.Equal(1f, t.JumpCutMultiplier);
        Assert.Equal(1f, t.ApexGravityMultiplier);
        Assert.Equal(1f, t.ApexSpeedBonus);
        Assert.Equal(0, t.CornerCorrection);
        Assert.Equal(ControllerTuning.Default.MaxRunSpeed, t.MaxRunSpeed);
    }

    [Fact]
    public void Enabling_everything_keeps_the_base_tuning()
    {
        Assert.Equal(ControllerTuning.Default, Assists.Apply(ControllerTuning.Default, AssistFeature.All));
    }

    [Fact]
    public void Every_assist_has_a_label()
    {
        Assert.Equal(5, Assists.Ordered.Select(Assists.Label).Distinct().Count());
        Assert.Throws<ArgumentOutOfRangeException>(() => Assists.Label(AssistFeature.All));
    }
}
