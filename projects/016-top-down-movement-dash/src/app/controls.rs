//! Keyboard and mouse -> core::InputFrame. The only place that knows about
//! key codes.
//!
//! WASD is a digital "left stick" and the arrow keys a digital "right
//! stick": exactly the 8-way, imprecise aim that aim assist exists for.
//! Moving the mouse switches aiming to the pointer; touching an arrow key
//! switches back, so whichever device you used last is in charge.

use macroquad::prelude::{
    is_key_down, is_key_pressed, is_mouse_button_down, is_mouse_button_pressed, mouse_position,
    KeyCode, MouseButton,
};
use topdown_dash::core::{AimSource, InputFrame, Vec2};

/// Pixels the mouse must travel in one frame to take over aiming.
const MOUSE_WAKE_DISTANCE: f32 = 2.0;

fn axis(negative: KeyCode, positive: KeyCode) -> f32 {
    let mut v = 0.0;
    if is_key_down(negative) {
        v -= 1.0;
    }
    if is_key_down(positive) {
        v += 1.0;
    }
    v
}

pub struct Controls {
    aim_source: AimSource,
    last_mouse: Vec2,
}

impl Controls {
    pub fn new() -> Controls {
        let (x, y) = mouse_position();
        Controls {
            aim_source: AimSource::Stick,
            last_mouse: Vec2::new(x, y),
        }
    }

    pub fn aim_source(&self) -> AimSource {
        self.aim_source
    }

    /// Samples held input. `player_screen` is where the player is drawn,
    /// so a pointer aim can be expressed relative to it.
    pub fn sample(&mut self, player_screen: Vec2) -> InputFrame {
        let keys_aim = Vec2::new(
            axis(KeyCode::Left, KeyCode::Right),
            axis(KeyCode::Up, KeyCode::Down),
        );
        let (mx, my) = mouse_position();
        let mouse = Vec2::new(mx, my);
        if keys_aim != Vec2::ZERO {
            self.aim_source = AimSource::Stick;
        } else if mouse.distance(self.last_mouse) > MOUSE_WAKE_DISTANCE
            || is_mouse_button_pressed(MouseButton::Left)
        {
            self.aim_source = AimSource::Pointer;
        }
        self.last_mouse = mouse;

        let aim_axis = match self.aim_source {
            AimSource::Stick => keys_aim,
            AimSource::Pointer => mouse - player_screen,
        };
        InputFrame {
            move_axis: Vec2::new(axis(KeyCode::A, KeyCode::D), axis(KeyCode::W, KeyCode::S)),
            aim_axis,
            aim_source: self.aim_source,
            fire: is_mouse_button_down(MouseButton::Left),
            dash_pressed: false,
        }
    }
}

/// Edge-triggered so that holding the key doesn't chain dashes.
pub fn dash_pressed() -> bool {
    is_key_pressed(KeyCode::Space)
        || is_key_pressed(KeyCode::LeftShift)
        || is_mouse_button_pressed(MouseButton::Right)
}

pub fn toggle_assist_pressed() -> bool {
    is_key_pressed(KeyCode::Key1) || is_key_pressed(KeyCode::F1)
}

pub fn cycle_curve_pressed() -> bool {
    is_key_pressed(KeyCode::Key2) || is_key_pressed(KeyCode::F2)
}

pub fn restart_pressed() -> bool {
    is_key_pressed(KeyCode::R)
}
