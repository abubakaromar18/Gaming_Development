#![allow(dead_code)] // each test binary uses a different subset

use topdown_dash::core::{Config, Event, InputFrame, SplitMix64, Vec2, World};

/// Default tuning with `drones` motionless drones.
pub fn config_with_drones(drones: usize) -> Config {
    let mut c = Config::default();
    c.arena.drone_count = drones;
    c.arena.drone_speed = 0.0;
    c
}

pub fn world(cfg: Config, seed: u64) -> World {
    World::new(cfg, Box::new(SplitMix64::new(seed))).expect("test config is valid")
}

pub fn centre(cfg: &Config) -> Vec2 {
    Vec2::new(cfg.arena.width / 2.0, cfg.arena.height / 2.0)
}

/// Runs `seconds` worth of identical input in steps of `dt`, collecting
/// every event. The final step is shortened so exactly `seconds` elapse.
pub fn run(world: &mut World, seconds: f32, dt: f32, input: InputFrame) -> Vec<Event> {
    let mut events = Vec::new();
    let mut left = seconds;
    while left > 1e-7 {
        let step = dt.min(left);
        events.extend(world.step(step, &input));
        left -= step;
    }
    events
}

pub fn held(move_axis: Vec2) -> InputFrame {
    InputFrame {
        move_axis,
        ..InputFrame::default()
    }
}

pub fn dash_press(move_axis: Vec2) -> InputFrame {
    InputFrame {
        move_axis,
        dash_pressed: true,
        ..InputFrame::default()
    }
}

pub fn approx(a: f32, b: f32, eps: f32) -> bool {
    (a - b).abs() <= eps
}
