//! Top-down twin-stick movement, dash and aim assist.
//!
//! Everything under [`core`] is pure Rust with no dependencies: no window,
//! no clock, no global RNG. The macroquad front end in `src/app/` samples
//! the keyboard and mouse into a [`core::InputFrame`], steps the
//! [`core::World`] with an explicit `dt`, and draws whatever state it finds.

pub mod core;
