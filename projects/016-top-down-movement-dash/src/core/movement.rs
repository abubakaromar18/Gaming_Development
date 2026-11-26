//! Velocity steering: turns a shaped stick into a new velocity.
//!
//! Velocity is split into the component *along* the stick and the component
//! *across* it, and each is handled with its own rule:
//!
//! * **Along, slower than wanted:** accelerate following the [`AccelCurve`].
//! * **Along, faster than wanted** (stick eased off, dash exit): bleed off
//!   at the normal deceleration rate.
//! * **Along, but backwards** (reversing): brake at the boosted turn rate.
//! * **Across** (the leftover from the previous heading): brake at the
//!   boosted turn rate, so turns are crisp instead of drifting like ice.
//!
//! With no stick input the whole vector decelerates toward zero.
//!
//! [`AccelCurve`]: super::curve::AccelCurve

use super::config::MovementConfig;
use super::math::{approach, Vec2};

/// Returns the velocity after `dt` seconds of steering with `stick`, which
/// must already be dead-zoned and response-curved (length in `[0, 1]`).
pub fn steer(velocity: Vec2, stick: Vec2, cfg: &MovementConfig, dt: f32) -> Vec2 {
    let decel = cfg.max_speed / cfg.decel_time;
    let tilt = stick.length().min(1.0);
    if tilt == 0.0 {
        return velocity.move_towards(Vec2::ZERO, decel * dt);
    }

    let dir = stick * (1.0 / stick.length());
    let target_speed = cfg.max_speed * tilt;
    let brake = decel * cfg.turn_brake_multiplier;

    let along = velocity.dot(dir);
    let across = (velocity - dir * along).move_towards(Vec2::ZERO, brake * dt);

    let along = if along < 0.0 {
        approach(along, 0.0, brake * dt)
    } else if along < target_speed {
        // Find where on the ramp the current speed sits, step forward in
        // ramp time, and read the curve again.
        let t = cfg.curve.inverse(along / cfg.max_speed) + dt / cfg.accel_time;
        (cfg.curve.eval(t) * cfg.max_speed).min(target_speed)
    } else {
        approach(along, target_speed, decel * dt)
    };

    dir * along + across
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::core::curve::AccelCurve;

    fn cfg(curve: AccelCurve) -> MovementConfig {
        MovementConfig {
            curve,
            ..MovementConfig::default()
        }
    }

    #[test]
    fn no_input_coasts_to_a_stop() {
        let c = cfg(AccelCurve::Linear);
        let mut v = Vec2::new(c.max_speed, 0.0);
        let steps = (c.decel_time * 120.0).ceil() as usize;
        for _ in 0..steps {
            v = steer(v, Vec2::ZERO, &c, 1.0 / 120.0);
        }
        assert_eq!(v, Vec2::ZERO);
    }

    #[test]
    fn partial_tilt_caps_speed() {
        let c = cfg(AccelCurve::EaseOut { power: 2.0 });
        let mut v = Vec2::ZERO;
        for _ in 0..240 {
            v = steer(v, Vec2::new(0.0, 0.5), &c, 1.0 / 120.0);
        }
        assert!((v.length() - c.max_speed * 0.5).abs() < 1e-3);
    }
}
