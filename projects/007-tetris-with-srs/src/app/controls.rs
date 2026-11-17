//! Keyboard -> core::Input. The only place that knows about key codes.

use macroquad::prelude::{is_key_down, is_key_pressed, KeyCode};
use tetris_srs::core::Input;

fn any_pressed(keys: &[KeyCode]) -> bool {
    keys.iter().any(|&k| is_key_pressed(k))
}

/// Directions and soft drop are sampled as held; everything else as a
/// press this frame, so the OS key-repeat never reaches the game.
pub fn read_input() -> Input {
    Input {
        left: is_key_down(KeyCode::Left),
        right: is_key_down(KeyCode::Right),
        soft_drop: is_key_down(KeyCode::Down),
        hard_drop: is_key_pressed(KeyCode::Space),
        rotate_cw: any_pressed(&[KeyCode::Up, KeyCode::X]),
        rotate_ccw: any_pressed(&[KeyCode::Z, KeyCode::LeftControl, KeyCode::RightControl]),
        hold: any_pressed(&[KeyCode::C, KeyCode::LeftShift, KeyCode::RightShift]),
    }
}

pub fn pause_pressed() -> bool {
    any_pressed(&[KeyCode::P, KeyCode::Escape])
}

pub fn restart_pressed() -> bool {
    is_key_pressed(KeyCode::R)
}
