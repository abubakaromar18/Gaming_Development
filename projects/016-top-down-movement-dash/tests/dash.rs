mod common;

use common::*;
use topdown_dash::core::{Config, DashPhase, Event, InputFrame, Vec2};

const RIGHT: Vec2 = Vec2::new(1.0, 0.0);

/// A config where nothing moves the player after the dash, so its end
/// position measures the dash distance alone.
fn still_exit() -> Config {
    let mut c = config_with_drones(0);
    c.dash.exit_speed_fraction = 0.0;
    c
}

#[test]
fn dash_covers_its_distance_at_any_step_size() {
    for dt in [1.0 / 30.0, 1.0 / 60.0, 1.0 / 144.0, 0.037] {
        let cfg = still_exit();
        let mut w = world(cfg.clone(), 1);
        w.step(dt, &dash_press(Vec2::ZERO));
        run(&mut w, 0.5, dt, InputFrame::default());
        let moved = w.player().pos.x - centre(&cfg).x;
        assert!(
            approx(moved, cfg.dash.distance, 1e-2),
            "dt={dt}: moved {moved}"
        );
    }
}

#[test]
fn dash_uses_the_stick_direction_over_facing() {
    let cfg = still_exit();
    let mut w = world(cfg.clone(), 1);
    w.step(1.0 / 60.0, &dash_press(Vec2::new(0.0, 1.0)));
    run(&mut w, 0.5, 1.0 / 60.0, InputFrame::default());
    let moved = w.player().pos - centre(&cfg);
    assert!(approx(moved.y, cfg.dash.distance, 1e-2));
    assert!(approx(moved.x, 0.0, 1e-3));
}

#[test]
fn dash_exits_into_a_run() {
    let cfg = config_with_drones(0);
    let mut w = world(cfg.clone(), 1);
    let dt = 1.0 / 120.0;
    let mut events = w.step(dt, &dash_press(RIGHT));
    while !events.contains(&Event::DashEnded) {
        events = w.step(dt, &held(RIGHT));
    }
    let v = w.player().vel.x;
    assert!(v > cfg.movement.max_speed * cfg.dash.exit_speed_fraction - 1.0);
    assert!(v <= cfg.movement.max_speed);
}

#[test]
fn i_frames_follow_the_configured_window() {
    let mut cfg = still_exit();
    cfg.dash.iframe_start = 0.03;
    cfg.dash.iframe_end = 0.25;
    let mut w = world(cfg.clone(), 1);
    let dt = 0.01;
    w.step(0.0, &dash_press(RIGHT));
    let mut t = 0.0;
    while t < 0.4 {
        let inv = w.player().dash.is_invulnerable(&cfg.dash);
        let expected = (0.03 - 1e-4..0.25 - 1e-4).contains(&t);
        assert_eq!(inv, expected, "t={t:.2}");
        w.step(dt, &InputFrame::default());
        t += dt;
    }
}

#[test]
fn cooldown_blocks_a_second_dash() {
    let cfg = config_with_drones(0);
    let mut w = world(cfg.clone(), 1);
    let dt = 1.0 / 60.0;
    w.step(dt, &dash_press(RIGHT));
    run(&mut w, cfg.dash.duration + 0.05, dt, InputFrame::default());
    let events = w.step(dt, &dash_press(RIGHT));
    assert!(!events
        .iter()
        .any(|e| matches!(e, Event::DashStarted { .. })));
    assert!(matches!(
        w.player().dash.phase(),
        DashPhase::Cooldown { .. }
    ));
}

#[test]
fn early_press_is_buffered_and_fires_the_moment_cooldown_ends() {
    let cfg = config_with_drones(0);
    let mut w = world(cfg.clone(), 1);
    let dt = 0.01;
    w.step(dt, &dash_press(RIGHT));
    // Press again just inside the buffer window before the dash is ready.
    let ready_at = cfg.dash.duration + cfg.dash.cooldown;
    let press_at = ready_at - cfg.dash.buffer_window + 0.02;
    run(&mut w, press_at - dt, dt, InputFrame::default());
    w.step(dt, &dash_press(RIGHT));
    assert!(w.player().dash.has_buffered_press());

    let mut started = 0.0;
    let mut t = press_at + dt;
    while started == 0.0 && t < ready_at + 0.1 {
        let events = w.step(dt, &InputFrame::default());
        t += dt;
        if events
            .iter()
            .any(|e| matches!(e, Event::DashStarted { .. }))
        {
            started = t;
        }
    }
    assert!(started > 0.0, "buffered dash never fired");
    assert!(started >= ready_at - 1e-4 && started <= ready_at + dt + 1e-4);
}

#[test]
fn a_press_too_early_is_dropped() {
    let cfg = config_with_drones(0);
    let mut w = world(cfg.clone(), 1);
    let dt = 0.01;
    w.step(dt, &dash_press(RIGHT));
    w.step(dt, &dash_press(RIGHT)); // mid-dash: far outside the window
    let events = run(&mut w, 1.0, dt, InputFrame::default());
    let starts = events
        .iter()
        .filter(|e| matches!(e, Event::DashStarted { .. }))
        .count();
    assert_eq!(starts, 0);
    assert!(events.contains(&Event::DashReady));
}

#[test]
fn dashing_through_a_drone_dodges_it_once() {
    let cfg = config_with_drones(1);
    let mut w = world(cfg.clone(), 1);
    w.place_drone(0, centre(&cfg) + Vec2::new(70.0, 0.0), RIGHT);
    let mut events = w.step(1.0 / 120.0, &dash_press(RIGHT));
    events.extend(run(
        &mut w,
        cfg.dash.duration,
        1.0 / 120.0,
        InputFrame::default(),
    ));

    let dodges = events
        .iter()
        .filter(|e| matches!(e, Event::Dodged { .. }))
        .count();
    assert_eq!(dodges, 1);
    assert!(!events.iter().any(|e| matches!(e, Event::PlayerHit { .. })));
    assert_eq!(w.player().health, cfg.combat.max_health);
    assert!(w.player().pos.x > centre(&cfg).x + 70.0 + cfg.arena.drone_radius);
}

#[test]
fn walking_into_a_drone_hurts_and_grants_mercy_frames() {
    let cfg = config_with_drones(1);
    let mut w = world(cfg.clone(), 1);
    w.place_drone(0, centre(&cfg) + Vec2::new(40.0, 0.0), RIGHT);
    let events = run(&mut w, 0.3, 1.0 / 120.0, held(RIGHT));
    let hits = events
        .iter()
        .filter(|e| matches!(e, Event::PlayerHit { .. }))
        .count();
    assert_eq!(hits, 1, "mercy frames should absorb repeat contact");
    assert_eq!(w.player().health, cfg.combat.max_health - 1);
}

#[test]
fn standing_in_a_drone_after_i_frames_end_hurts() {
    let mut cfg = config_with_drones(1);
    cfg.dash.exit_speed_fraction = 0.0;
    let mut w = world(cfg.clone(), 1);
    // Park the drone exactly where the dash ends.
    w.place_drone(0, centre(&cfg) + Vec2::new(cfg.dash.distance, 0.0), RIGHT);
    let mut events = w.step(1.0 / 120.0, &dash_press(RIGHT));
    events.extend(run(
        &mut w,
        cfg.dash.duration,
        1.0 / 120.0,
        InputFrame::default(),
    ));
    assert!(!events.iter().any(|e| matches!(e, Event::PlayerHit { .. })));
    let later = run(
        &mut w,
        cfg.dash.iframe_end,
        1.0 / 120.0,
        InputFrame::default(),
    );
    assert!(later.iter().any(|e| matches!(e, Event::PlayerHit { .. })));
}
