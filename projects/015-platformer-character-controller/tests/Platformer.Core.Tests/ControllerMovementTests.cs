using Platformer.Core.Controller;
using Platformer.Core.Input;
using Platformer.Core.Physics;

namespace Platformer.Core.Tests;

public class ControllerMovementTests
{
    private static readonly ControllerTuning Plain = Assists.Apply(ControllerTuning.Default, AssistFeature.None);

    [Fact]
    public void Gravity_and_launch_speed_are_derived_from_height_and_time()
    {
        var t = ControllerTuning.Default with { JumpHeight = 48f, TimeToApex = 0.4f };

        Assert.Equal(600.0, t.Gravity, 3);       // 2·48 / 0.4²
        Assert.Equal(240.0, t.JumpVelocity, 3);  // 2·48 / 0.4
    }

    [Fact]
    public void Spawning_on_the_floor_starts_grounded()
    {
        var c = Sim.Controller(Maps.Room());

        Assert.True(c.Grounded);
        Assert.Equal(MovementPhase.Grounded, c.Phase);
        Assert.Equal(Maps.Room().Spawn, c.Feet);
    }

    [Fact]
    public void Standing_still_stays_grounded_and_in_place()
    {
        var c = Sim.Controller(Maps.Room());
        var start = c.Feet;

        Sim.Run(c, ControllerInput.None, 240);

        Assert.True(c.Grounded);
        Assert.Equal(start, c.Feet);
        Assert.Empty(c.Events);
    }

    [Fact]
    public void Running_reaches_max_speed_in_about_speed_over_acceleration()
    {
        var c = Sim.Controller(Maps.Room(), Plain);
        var expectedSteps = (int)MathF.Ceiling(Plain.MaxRunSpeed / Plain.GroundAcceleration / Maps.Dt);

        Sim.Run(c, Sim.Hold(1f), expectedSteps - 1);
        Assert.True(c.Velocity.X < Plain.MaxRunSpeed);

        Sim.Run(c, Sim.Hold(1f), 1);
        Assert.Equal(Plain.MaxRunSpeed, c.Velocity.X);
    }

    [Fact]
    public void Releasing_input_on_the_ground_decelerates_to_a_stop()
    {
        var c = Sim.Controller(Maps.Room(), Plain);
        Sim.Run(c, Sim.Hold(1f), 30);

        Sim.Run(c, ControllerInput.None, 1);
        Assert.Equal(Plain.MaxRunSpeed - Plain.GroundDeceleration * Maps.Dt, c.Velocity.X, 3);

        Sim.Run(c, ControllerInput.None, 20);
        Assert.Equal(0f, c.Velocity.X);
    }

    [Fact]
    public void Turning_around_on_the_ground_uses_the_faster_turn_rate()
    {
        var c = Sim.Controller(Maps.Room(), Plain);
        Sim.Run(c, Sim.Hold(1f), 30);

        Sim.Run(c, Sim.Hold(-1f), 1);

        Assert.Equal(Plain.MaxRunSpeed - Plain.TurnAcceleration * Maps.Dt, c.Velocity.X, 3);
        Assert.Equal(-1, c.Facing);
    }

    [Fact]
    public void Analog_input_targets_a_proportional_speed()
    {
        var c = Sim.Controller(Maps.Room(), Plain);

        Sim.Run(c, Sim.Hold(0.5f), 60);

        Assert.Equal(Plain.MaxRunSpeed * 0.5f, c.Velocity.X, 3);
    }

    [Fact]
    public void Out_of_range_or_NaN_axis_values_are_sanitised()
    {
        var c = Sim.Controller(Maps.Room(), Plain);

        Sim.Run(c, Sim.Hold(7f), 60);
        Assert.Equal(Plain.MaxRunSpeed, c.Velocity.X);

        Sim.Run(c, Sim.Hold(float.NaN), 60);
        Assert.Equal(0f, c.Velocity.X);
    }

    [Fact]
    public void Running_into_a_wall_stops_horizontal_velocity()
    {
        var c = Sim.Controller(Maps.Room(), Plain);

        Sim.Run(c, Sim.Hold(-1f), 60);

        Assert.Equal(0f, c.Velocity.X);
        Assert.Equal(16f, c.Body.Left);
    }

    [Fact]
    public void A_full_jump_peaks_at_the_designed_height()
    {
        var c = Sim.Controller(Maps.Room(), Plain);

        var (peak, _) = Sim.Jump(c, holdSteps: 1000);

        // Semi-implicit Euler at 120 Hz lands within a couple of pixels of the analytic arc.
        Assert.InRange(peak, Plain.JumpHeight - 2f, Plain.JumpHeight + 0.5f);
    }

    [Fact]
    public void Falling_is_faster_than_rising_with_a_fall_gravity_multiplier()
    {
        var c = Sim.Controller(Maps.Room(), Plain);
        var riseSteps = 0;
        c.Step(Sim.Press(), Maps.Dt);
        while (c.Velocity.Y < 0f)
        {
            c.Step(Sim.Hold(jump: true), Maps.Dt);
            riseSteps++;
        }
        var fallSteps = 0;
        while (!c.Grounded)
        {
            c.Step(Sim.Hold(jump: true), Maps.Dt);
            fallSteps++;
        }

        Assert.True(fallSteps < riseSteps, $"fall {fallSteps} steps vs rise {riseSteps}");
        Assert.InRange(riseSteps * Maps.Dt, Plain.TimeToApex - 0.02f, Plain.TimeToApex + 0.02f);
    }

    [Fact]
    public void Fall_speed_is_capped_at_terminal_velocity()
    {
        var level = Maps.Parse(Enumerable.Range(0, 60).Select(i => i == 1 ? "#..P..#" : "#.....#").Append("#######").ToArray());
        var c = Sim.Controller(level, Plain);

        Sim.Run(c, ControllerInput.None, 120);

        Assert.Equal(Plain.MaxFallSpeed, c.Velocity.Y);
        Assert.Equal(MovementPhase.Falling, c.Phase);
    }

    [Fact]
    public void Landing_reports_the_impact_speed_once()
    {
        var c = Sim.Controller(Maps.MidAir(), Plain);

        var events = Sim.Run(c, ControllerInput.None, 240);

        var landing = Assert.Single(events.Where(e => e.Kind == ControllerEventKind.Landed));
        Assert.True(landing.Value > 100f);
        Assert.True(c.Grounded);
    }

    [Fact]
    public void Jumping_requires_the_ground_without_assists()
    {
        var c = Sim.Controller(Maps.MidAir(), Plain);

        var events = Sim.Run(c, Sim.Press(), 1);

        Assert.DoesNotContain(ControllerEventKind.Jumped, events.Select(e => e.Kind));
    }

    [Fact]
    public void Holding_jump_does_not_auto_repeat_on_landing()
    {
        var c = Sim.Controller(Maps.Room(), Plain);
        c.Step(Sim.Press(), Maps.Dt);

        var events = Sim.Run(c, Sim.Hold(jump: true), 240);

        Assert.True(c.Grounded);
        Assert.DoesNotContain(ControllerEventKind.Jumped, events.Select(e => e.Kind));
    }

    [Fact]
    public void Head_bump_stops_upward_motion()
    {
        var c = Sim.Controller(Maps.Notch(), Plain);

        var events = Sim.Run(c, Sim.Press(), 1).Concat(Sim.Run(c, Sim.Hold(jump: true), 40)).ToList();

        Assert.Contains(ControllerEventKind.HeadBump, events.Select(e => e.Kind));
        Assert.True(c.Body.Top >= 48f);
    }

    [Fact]
    public void Invalid_steps_and_placements_are_rejected()
    {
        var level = Maps.Room();
        var c = Sim.Controller(level);

        Assert.Throws<ArgumentOutOfRangeException>(() => c.Step(ControllerInput.None, 0f));
        Assert.Throws<ArgumentOutOfRangeException>(() => c.Step(ControllerInput.None, float.NaN));
        Assert.Throws<ArgumentException>(() => c.Teleport(new Vec2(8f, 100f)));
        Assert.Throws<ArgumentException>(() => c.Tuning = c.Tuning with { BodyWidth = 20f });
    }

    [Fact]
    public void Tuning_validation_names_the_bad_value()
    {
        var bad = ControllerTuning.Default with { TimeToApex = -1f };
        var cut = ControllerTuning.Default with { JumpCutMultiplier = 1.5f };
        var corner = ControllerTuning.Default with { CornerCorrection = 6 };

        Assert.Contains("TimeToApex", Assert.Throws<ArgumentException>(() => bad.Validated()).Message);
        Assert.Contains("JumpCutMultiplier", Assert.Throws<ArgumentException>(() => cut.Validated()).Message);
        Assert.Contains("CornerCorrection", Assert.Throws<ArgumentException>(() => corner.Validated()).Message);
        Assert.Throws<ArgumentException>(() => Sim.Controller(Maps.Room(), bad));
    }
}
