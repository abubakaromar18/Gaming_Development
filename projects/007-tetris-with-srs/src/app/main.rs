//! macroquad shell: owns the window, the clock and the keyboard, and feeds
//! them into the pure `tetris_srs::core::Game`.

mod controls;
mod render;

use macroquad::prelude::*;
use tetris_srs::core::{Config, Event, Game, SevenBag, SplitMix64};

use render::{Popup, View};

/// Longest frame the simulation will accept. A window drag or breakpoint
/// shouldn't turn into seconds of gravity at once.
const MAX_FRAME_SECONDS: f64 = 0.1;
const POPUP_SECONDS: f32 = 1.6;

fn window_conf() -> Conf {
    Conf {
        window_title: "Tetris with SRS".to_owned(),
        window_width: 720,
        window_height: 760,
        window_resizable: false,
        ..Default::default()
    }
}

/// `--seed N` makes the piece order reproducible; otherwise the clock picks
/// one. This is the only place the wall clock is used for randomness.
fn seed_from_args() -> Result<Option<u64>, String> {
    let args: Vec<String> = std::env::args().skip(1).collect();
    match args.as_slice() {
        [] => Ok(None),
        [flag, value] if flag == "--seed" => value
            .parse::<u64>()
            .map(Some)
            .map_err(|e| format!("--seed expects an unsigned integer: {e}")),
        _ => Err("usage: tetris-srs [--seed N]".to_owned()),
    }
}

fn clock_seed() -> u64 {
    (miniquad::date::now() * 1_000_000.0) as u64
}

fn new_game(seed: u64) -> Game {
    let bag = SevenBag::new(Box::new(SplitMix64::new(seed)));
    Game::new(Config::default(), Box::new(bag)).expect("the default config is valid")
}

struct App {
    seed: u64,
    game: Game,
    paused: bool,
    popup: Option<Popup>,
}

impl App {
    fn new(seed: u64) -> App {
        App {
            seed,
            game: new_game(seed),
            paused: false,
            popup: None,
        }
    }

    fn frame(&mut self, dt: f64) {
        if controls::restart_pressed() {
            *self = App::new(clock_seed());
            return;
        }
        if controls::pause_pressed() {
            self.paused = !self.paused;
        }
        if !self.paused {
            let input = controls::read_input();
            self.game.update(dt.min(MAX_FRAME_SECONDS), &input);
        }
        for event in self.game.drain_events() {
            if let Event::Scored(report) = event {
                self.popup = Some(Popup {
                    text: render::describe(&report),
                    points: report.points,
                    remaining: POPUP_SECONDS,
                });
            }
        }
        if let Some(p) = self.popup.as_mut() {
            p.remaining -= dt as f32;
            if p.remaining <= 0.0 {
                self.popup = None;
            }
        }
    }

    fn draw(&self) {
        render::draw(&View {
            game: &self.game,
            popup: self.popup.as_ref(),
            paused: self.paused,
            seed: self.seed,
        });
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
        app.frame(get_frame_time() as f64);
        app.draw();
        next_frame().await;
    }
}
