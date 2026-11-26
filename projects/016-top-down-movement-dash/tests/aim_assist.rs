mod common;

use common::*;
use topdown_dash::core::aim_assist::solve;
use topdown_dash::core::{AimAssistConfig, AimCandidate, AimSource, Event, InputFrame, Vec2};

const RIGHT: Vec2 = Vec2::new(1.0, 0.0);

fn target(id: u32, x: f32, y: f32) -> AimCandidate {
    AimCandidate {
        id,
        pos: Vec2::new(x, y),
        radius: 8.0,
    }
}

fn angle_between(a: Vec2, b: Vec2) -> f32 {
    a.angle_to(b).abs()
}

#[test]
fn full_strength_snaps_onto_the_target() {
    let cfg = AimAssistConfig::default();
    let t = target(7, 300.0, 40.0);
    let s = solve(Vec2::ZERO, RIGHT, &[t], None, 1.0, &cfg);
    assert_eq!(s.target, Some(7));
    assert!(angle_between(s.dir, t.pos) < 1e-5);
}

#[test]
fn partial_strength_only_nudges() {
    let cfg = AimAssistConfig::default();
    let t = target(1, 300.0, 40.0);
    let off = RIGHT.angle_to(t.pos);
    let s = solve(Vec2::ZERO, RIGHT, &[t], None, 0.5, &cfg);
    assert!((RIGHT.angle_to(s.dir) - off * 0.5).abs() < 1e-5);
    assert!((s.dir.length() - 1.0).abs() < 1e-5);
}

#[test]
fn targets_outside_the_cone_or_range_are_ignored() {
    let cfg = AimAssistConfig::default();
    let behind = target(1, -200.0, 0.0);
    let wide = target(2, 100.0, 100.0); // 45 degrees off
    let far = target(3, cfg.range + 50.0, 0.0);
    let s = solve(Vec2::ZERO, RIGHT, &[behind, wide, far], None, 1.0, &cfg);
    assert_eq!(s.target, None);
    assert_eq!(s.dir, RIGHT);
}

#[test]
fn big_targets_are_easier_to_catch() {
    let cfg = AimAssistConfig {
        cone_half_angle: 0.1,
        ..AimAssistConfig::default()
    };
    // 0.2 rad off-centre: outside the bare cone...
    let pos = Vec2::from_angle(0.2) * 200.0;
    let small = AimCandidate {
        id: 1,
        pos,
        radius: 2.0,
    };
    assert_eq!(
        solve(Vec2::ZERO, RIGHT, &[small], None, 1.0, &cfg).target,
        None
    );
    // ...but a wide target's edge reaches into it.
    let large = AimCandidate {
        id: 1,
        pos,
        radius: 30.0,
    };
    assert_eq!(
        solve(Vec2::ZERO, RIGHT, &[large], None, 1.0, &cfg).target,
        Some(1)
    );
}

#[test]
fn prefers_the_target_closest_to_the_aim_line() {
    let cfg = AimAssistConfig {
        distance_weight: 0.0,
        ..AimAssistConfig::default()
    };
    let near_line = target(1, 400.0, 20.0);
    let off_line = target(2, 150.0, 30.0);
    let s = solve(Vec2::ZERO, RIGHT, &[off_line, near_line], None, 1.0, &cfg);
    assert_eq!(s.target, Some(1));
}

#[test]
fn stickiness_keeps_the_current_target_in_a_near_tie() {
    let cfg = AimAssistConfig {
        distance_weight: 0.0,
        ..AimAssistConfig::default()
    };
    // Target 2 is very slightly closer to the aim line than target 1.
    let a = target(1, 300.0, 52.0);
    let b = target(2, 300.0, -48.0);
    assert_eq!(
        solve(Vec2::ZERO, RIGHT, &[a, b], None, 1.0, &cfg).target,
        Some(2)
    );
    assert_eq!(
        solve(Vec2::ZERO, RIGHT, &[a, b], Some(1), 1.0, &cfg).target,
        Some(1)
    );

    let no_stick = AimAssistConfig {
        stickiness: 0.0,
        ..cfg
    };
    assert_eq!(
        solve(Vec2::ZERO, RIGHT, &[a, b], Some(1), 1.0, &no_stick).target,
        Some(2)
    );
}

/// A drone sits 0.12 rad off the stick's (8-way keyboard) direction. With
/// assist the stream of bullets kills it; without, every bullet misses.
fn shoot_offset_drone(assist: bool) -> u32 {
    let mut cfg = config_with_drones(1);
    cfg.aim.enabled = assist;
    let mut w = world(cfg.clone(), 3);
    let drone_pos = centre(&cfg) + Vec2::from_angle(0.12) * 300.0;
    w.place_drone(0, drone_pos, RIGHT);
    let aim = InputFrame {
        aim_axis: RIGHT,
        aim_source: AimSource::Stick,
        ..InputFrame::default()
    };
    let events = run(&mut w, 1.0, 1.0 / 120.0, aim);
    assert!(events.iter().any(|e| matches!(e, Event::Shot { .. })));
    w.score()
}

#[test]
fn assist_turns_near_misses_into_hits_in_the_world() {
    assert_eq!(shoot_offset_drone(false), 0);
    assert_eq!(shoot_offset_drone(true), 1);
}

#[test]
fn pointer_aim_is_not_assisted_by_default() {
    let cfg = config_with_drones(1);
    let mut w = world(cfg.clone(), 3);
    w.place_drone(0, centre(&cfg) + Vec2::from_angle(0.12) * 300.0, RIGHT);
    let aim = InputFrame {
        aim_axis: RIGHT * 300.0,
        aim_source: AimSource::Pointer,
        fire: true,
        ..InputFrame::default()
    };
    w.step(1.0 / 120.0, &aim);
    let solution = w.aim().expect("pointer aim is active");
    assert_eq!(solution.target, None);
    assert_eq!(solution.dir, RIGHT);
}

#[test]
fn stick_aim_inside_the_dead_zone_does_not_fire() {
    let cfg = config_with_drones(0);
    let mut w = world(cfg, 3);
    let drift = InputFrame {
        aim_axis: Vec2::new(0.1, 0.05),
        ..InputFrame::default()
    };
    let events = run(&mut w, 0.5, 1.0 / 60.0, drift);
    assert!(w.aim().is_none());
    assert!(!events.iter().any(|e| matches!(e, Event::Shot { .. })));
}
