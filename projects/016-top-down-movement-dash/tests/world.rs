mod common;

use common::*;
use topdown_dash::core::{
    AimSource, Config, Event, InputFrame, Player, RandomSource, SplitMix64, Vec2, World,
};

/// Scripted pseudo-random input: the same seed yields the same "player".
fn scripted_input(rng: &mut SplitMix64) -> InputFrame {
    InputFrame {
        move_axis: Vec2::new(rng.range(-1.2, 1.2), rng.range(-1.2, 1.2)),
        aim_axis: Vec2::new(rng.range(-1.0, 1.0), rng.range(-1.0, 1.0)),
        aim_source: AimSource::Stick,
        fire: false,
        dash_pressed: rng.next_f32() < 0.03,
    }
}

fn play(seed: u64, steps: usize) -> World {
    let mut w = world(Config::default(), seed);
    let mut script = SplitMix64::new(99);
    for _ in 0..steps {
        let input = scripted_input(&mut script);
        w.step(1.0 / 120.0, &input);
    }
    w
}

#[test]
fn same_seed_and_inputs_give_identical_worlds() {
    let a = play(42, 2_000);
    let b = play(42, 2_000);
    assert_eq!(a.player(), b.player());
    assert_eq!(a.drones(), b.drones());
    assert_eq!(a.bullets(), b.bullets());
    assert_eq!(a.score(), b.score());
}

#[test]
fn different_seeds_place_drones_differently() {
    let a = world(Config::default(), 1);
    let b = world(Config::default(), 2);
    assert_ne!(a.drones()[0].pos, b.drones()[0].pos);
}

#[test]
fn drones_spawn_clear_of_the_player() {
    for seed in 0..50 {
        let cfg = Config::default();
        let w = world(cfg.clone(), seed);
        for d in w.drones() {
            assert!(d.pos.distance(centre(&cfg)) >= cfg.arena.drone_spawn_clearance);
        }
    }
}

#[test]
fn fire_rate_is_exact_at_any_step_size() {
    let shots = |dt: f32| {
        let mut w = world(config_with_drones(0), 1);
        let aim = InputFrame {
            aim_axis: Vec2::new(0.0, -1.0),
            ..InputFrame::default()
        };
        run(&mut w, 1.0, dt, aim)
            .iter()
            .filter(|e| matches!(e, Event::Shot { .. }))
            .count()
    };
    // Shots at t = 0, 0.11, ..., 0.99.
    assert_eq!(shots(1.0 / 30.0), 10);
    assert_eq!(shots(1.0 / 240.0), 10);
}

#[test]
fn destroyed_drones_respawn_after_the_delay() {
    let mut cfg = config_with_drones(1);
    cfg.arena.drone_health = 1;
    let mut w = world(cfg.clone(), 5);
    w.place_drone(0, centre(&cfg) + Vec2::new(200.0, 0.0), Vec2::new(1.0, 0.0));
    let aim = InputFrame {
        aim_axis: Vec2::new(1.0, 0.0),
        ..InputFrame::default()
    };
    let events = run(&mut w, 0.5, 1.0 / 120.0, aim);
    assert!(events.contains(&Event::DroneDestroyed { drone: 0 }));
    assert!(!w.drones()[0].alive);
    assert_eq!(w.score(), 1);

    let later = run(
        &mut w,
        cfg.arena.drone_respawn_delay,
        1.0 / 120.0,
        InputFrame::default(),
    );
    assert!(later.contains(&Event::DroneRespawned { drone: 0 }));
    assert!(w.drones()[0].alive);
}

#[test]
fn running_out_of_health_ends_the_run() {
    let mut cfg = config_with_drones(1);
    cfg.combat.max_health = 2;
    cfg.combat.hurt_invulnerability = 0.05;
    let mut w = world(cfg.clone(), 5);
    w.place_drone(0, centre(&cfg), Vec2::new(1.0, 0.0));
    let events = run(&mut w, 1.0, 1.0 / 120.0, InputFrame::default());
    assert!(events.contains(&Event::PlayerDefeated));
    assert!(w.is_over());

    let frozen = w.player().clone();
    assert!(w.step(0.1, &held(Vec2::new(1.0, 0.0))).is_empty());
    assert_eq!(w.player(), &frozen);
}

#[test]
fn bad_dt_is_ignored() {
    let cfg = config_with_drones(0);
    let mut w = world(cfg.clone(), 1);
    for dt in [f32::NAN, -1.0, f32::INFINITY] {
        w.step(dt, &held(Vec2::new(1.0, 0.0)));
    }
    assert_eq!(w.player().pos, centre(&cfg));
    assert_eq!(w.time(), 0.0);
}

#[test]
fn invalid_configs_are_rejected() {
    let mut bad = Config::default();
    bad.dash.duration = 0.0;
    assert!(World::new(bad.clone(), Box::new(SplitMix64::new(1))).is_err());

    let mut w = world(Config::default(), 1);
    assert!(w.set_config(bad).is_err());
    assert_eq!(w.config(), &Config::default());
}

/// Stress: a long session of erratic input must keep every actor in the
/// arena with finite state.
#[test]
fn long_random_session_stays_sane() {
    let cfg = Config::default();
    let mut w = world(cfg.clone(), 7);
    let mut script = SplitMix64::new(1234);
    let mut dt_rng = SplitMix64::new(77);
    for _ in 0..50_000 {
        let input = scripted_input(&mut script);
        // Uneven frame times, including long hitches.
        let dt = dt_rng.range(0.001, 0.05);
        w.step(dt, &input);
        let p = w.player();
        assert!(p.pos.x.is_finite() && p.pos.y.is_finite() && p.vel.length().is_finite());
        let r = cfg.arena.player_radius;
        assert!(p.pos.x >= r - 1e-3 && p.pos.x <= cfg.arena.width - r + 1e-3);
        assert!(p.pos.y >= r - 1e-3 && p.pos.y <= cfg.arena.height - r + 1e-3);
        assert!(p.vel.length() <= Player::dash_speed(&cfg) + 1e-2);
        for d in w.drones() {
            assert!(d.pos.x >= 0.0 && d.pos.x <= cfg.arena.width);
            assert!(d.pos.y >= 0.0 && d.pos.y <= cfg.arena.height);
        }
        assert!(w.bullets().len() < 200, "bullets are leaking");
        if w.is_over() {
            w = world(cfg.clone(), 7);
        }
    }
}
