mod common;

use common::*;
use topdown_dash::core::movement::steer;
use topdown_dash::core::{AccelCurve, MovementConfig, Vec2};

const CURVES: [AccelCurve; 3] = [
    AccelCurve::Linear,
    AccelCurve::EaseOut { power: 2.0 },
    AccelCurve::EaseIn { power: 2.0 },
];

fn speed_after(cfg: &MovementConfig, seconds: f32, dt: f32) -> f32 {
    let mut v = Vec2::ZERO;
    let mut t = 0.0;
    while t + dt <= seconds + 1e-6 {
        v = steer(v, Vec2::new(1.0, 0.0), cfg, dt);
        t += dt;
    }
    v.length()
}

#[test]
fn every_curve_reaches_top_speed_in_exactly_accel_time() {
    for curve in CURVES {
        let cfg = MovementConfig {
            curve,
            accel_time: 0.25,
            ..MovementConfig::default()
        };
        let dt = 1.0 / 200.0;
        assert!(
            speed_after(&cfg, 0.2, dt) < cfg.max_speed,
            "{curve:?} arrived early"
        );
        assert!(
            approx(speed_after(&cfg, 0.25, dt), cfg.max_speed, 1e-2),
            "{curve:?} arrived late"
        );
    }
}

#[test]
fn ramp_does_not_depend_on_step_size() {
    for curve in CURVES {
        let cfg = MovementConfig {
            curve,
            accel_time: 0.3,
            ..MovementConfig::default()
        };
        let coarse = speed_after(&cfg, 0.15, 0.05);
        let fine = speed_after(&cfg, 0.15, 0.001);
        assert!(approx(coarse, fine, 0.5), "{curve:?}: {coarse} vs {fine}");
    }
}

#[test]
fn curve_shapes_the_ramp() {
    let at_half = |curve| {
        let cfg = MovementConfig {
            curve,
            accel_time: 0.2,
            ..MovementConfig::default()
        };
        speed_after(&cfg, 0.1, 0.001)
    };
    let ease_out = at_half(AccelCurve::EaseOut { power: 2.0 });
    let linear = at_half(AccelCurve::Linear);
    let ease_in = at_half(AccelCurve::EaseIn { power: 2.0 });
    assert!(
        ease_out > linear && linear > ease_in,
        "{ease_out} {linear} {ease_in}"
    );
    assert!(approx(linear, 160.0, 1.0));
    assert!(approx(ease_out, 240.0, 1.0));
}

#[test]
fn turn_brake_makes_reversals_quicker() {
    let reverse_time = |multiplier: f32| {
        let cfg = MovementConfig {
            turn_brake_multiplier: multiplier,
            ..MovementConfig::default()
        };
        let mut v = Vec2::new(cfg.max_speed, 0.0);
        let dt = 1.0 / 240.0;
        let mut t = 0.0;
        while v.x > -cfg.max_speed * 0.99 {
            v = steer(v, Vec2::new(-1.0, 0.0), &cfg, dt);
            t += dt;
        }
        t
    };
    assert!(reverse_time(3.0) < reverse_time(1.0) * 0.8);
}

#[test]
fn sharp_turns_shed_old_momentum() {
    let cfg = MovementConfig::default();
    let mut v = Vec2::new(cfg.max_speed, 0.0);
    let dt = 1.0 / 120.0;
    let brake_time = cfg.decel_time / cfg.turn_brake_multiplier;
    let steps = (brake_time / dt).ceil() as usize;
    for _ in 0..steps {
        v = steer(v, Vec2::new(0.0, 1.0), &cfg, dt);
    }
    assert_eq!(v.x, 0.0, "sideways drift remained: {v:?}");
    assert!(v.y > 0.0);
}

#[test]
fn dead_zone_filters_stick_drift() {
    let cfg = config_with_drones(0);
    let mut w = world(cfg.clone(), 1);
    run(&mut w, 1.0, 1.0 / 60.0, held(Vec2::new(0.1, -0.12)));
    assert_eq!(w.player().pos, centre(&cfg));
}

#[test]
fn stopping_point_is_frame_rate_independent() {
    let travel = |dt: f32| {
        let cfg = config_with_drones(0);
        let mut w = world(cfg.clone(), 1);
        run(&mut w, 0.4, dt, held(Vec2::new(1.0, 0.0)));
        run(&mut w, 0.4, dt, held(Vec2::ZERO));
        w.player().pos.x - centre(&cfg).x
    };
    let slow = travel(1.0 / 30.0);
    let fast = travel(1.0 / 240.0);
    assert!(approx(slow, fast, 1.5), "{slow} vs {fast}");
}

#[test]
fn walls_stop_the_player_without_storing_speed() {
    let cfg = config_with_drones(0);
    let mut w = world(cfg.clone(), 1);
    run(&mut w, 3.0, 1.0 / 60.0, held(Vec2::new(1.0, 0.0)));
    let p = w.player();
    assert!(approx(
        p.pos.x,
        cfg.arena.width - cfg.arena.player_radius,
        1e-3
    ));
    assert_eq!(p.vel.x, 0.0, "{p:?}");
    // Turning away is immediate: there's no hidden velocity to cancel.
    w.step(1.0 / 60.0, &held(Vec2::new(-1.0, 0.0)));
    assert!(w.player().pos.x < cfg.arena.width - cfg.arena.player_radius);
}

#[test]
fn partial_tilt_with_response_curve_moves_slower() {
    let cfg = config_with_drones(0);
    let mut w = world(cfg.clone(), 1);
    run(&mut w, 0.5, 1.0 / 120.0, held(Vec2::new(0.6, 0.0)));
    let expected_tilt = ((0.6f32 - 0.18) / (0.95 - 0.18)).powf(1.5);
    assert!(approx(
        w.player().vel.x,
        cfg.movement.max_speed * expected_tilt,
        0.5
    ));
}
