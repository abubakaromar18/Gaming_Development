//! macroquad shell: owns the window, the clock, the keyboard and mouse, and
//! feeds them into the pure `topdown_dash::core::World` at a fixed rate.

mod controls;
mod render;

use macroquad::prelude::*;
use topdown_dash::core::{AccelCurve, Config, SplitMix64, World};

use controls::Controls;
use render::{View, HUD_HEIGHT};

/// The simulation runs at a fixed 120 Hz regardless of display refresh.
const STEP: f32 = 1.0 / 120.0;
/// Longest frame accepted; a window drag shouldn't fast-forward the game.
const MAX_FRAME_SECONDS: f32 = 0.1;

/// The feels the demo cycles through with [2].
const CURVES: [AccelCurve; 3] = [
    AccelCurve::EaseOut { power: 2.0 },
    AccelCurve::Linear,
    AccelCurve::EaseIn { power: 2.0 },
];

fn window_conf() -> Conf {
    let arena = Config::default().arena;
    Conf {
        window_title: "Top-Down Movement & Dash".to_owned(),
        window_width: arena.width as i32,
        window_height: (arena.height + HUD_HEIGHT) as i32,
        window_resizable: false,
        ..Default::default()
    }
}

/// `--seed N` makes drone placement reproducible; otherwise the clock picks
/// one. This is the only place the wall clock feeds randomness.
fn seed_from_args() -> Result<Option<u64>, String> {
    let args: Vec<String> = std::env::args().skip(1).collect();
    match args.as_slice() {
        [] => Ok(None),
        [flag, value] if flag == "--seed" => value
            .parse::<u64>()
            .map(Some)
            .map_err(|e| format!("--seed expects an unsigned integer: {e}")),
        _ => Err("usage: topdown-dash [--seed N]".to_owned()),
    }
}

fn clock_seed() -> u64 {
    (miniquad::date::now() * 1_000_000.0) as u64
}

fn new_world(cfg: Config, seed: u64) -> World {
    World::new(cfg, Box::new(SplitMix64::new(seed))).expect("demo tuning is valid")
}

struct App {
    world: World,
    view: View,
    controls: Controls,
    curve_index: usize,
    accumulator: f32,
    /// A dash press waits here until a simulation step consumes it, so it
    /// is never lost on a frame that runs zero steps or counted twice on a
    /// frame that runs several.
    pending_dash: bool,
}

impl App {
    fn new(seed: u64) -> App {
        App {
            world: new_world(Config::default(), seed),
            view: View::new(),
            controls: Controls::new(),
            curve_index: 0,
            accumulator: 0.0,
            pending_dash: false,
        }
    }

    fn retune(&mut self, change: impl FnOnce(&mut Config)) {
        let mut cfg = self.world.config().clone();
        change(&mut cfg);
        // Only known-good presets are applied, so a failure is a bug.
        self.world.set_config(cfg).expect("preset tuning is valid");
    }

    fn frame(&mut self, frame_dt: f32) {
        if controls::restart_pressed() {
            let cfg = self.world.config().clone();
            self.world = new_world(cfg, clock_seed());
            self.view = View::new();
            self.accumulator = 0.0;
            return;
        }
        if controls::toggle_assist_pressed() {
            self.retune(|c| c.aim.enabled = !c.aim.enabled);
        }
        if controls::cycle_curve_pressed() {
            self.curve_index = (self.curve_index + 1) % CURVES.len();
            let curve = CURVES[self.curve_index];
            self.retune(|c| c.movement.curve = curve);
        }
        self.pending_dash |= controls::dash_pressed();

        self.accumulator += frame_dt.min(MAX_FRAME_SECONDS);
        while self.accumulator >= STEP {
            let mut input = self.controls.sample(View::player_screen(&self.world));
            input.dash_pressed = std::mem::take(&mut self.pending_dash);
            let events = self.world.step(STEP, &input);
            self.view.react(&self.world, &events);
            self.view.tick(&self.world, STEP);
            self.accumulator -= STEP;
        }
    }
}

#[macroquad::main(window_conf)]
async fn main() {
    let seed = match seed_from_args() {
        Ok(seed) => seed.unwrap_or_else(clock_seed),
        Err(msg) => {
            eprintln!("{msg}");
            std::process::exit(2);
        }
    };
    let mut app = App::new(seed);
    loop {
        app.frame(get_frame_time());
        app.view.draw(&app.world, app.controls.aim_source());
        next_frame().await;
    }
}
