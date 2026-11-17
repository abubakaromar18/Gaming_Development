//! Guideline-style Tetris simulation.
//!
//! Everything under [`core`] is pure Rust with no dependencies: no window,
//! no clock, no global RNG. The macroquad front end in `src/app/` only reads
//! game state and translates keys into [`core::Input`].

pub mod core;
