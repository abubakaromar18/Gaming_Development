//! What the player asked for this step, already in the core's vocabulary.
//!
//! The front end turns keys, mouse or gamepad sticks into an [`InputFrame`].
//! Sticks arrive raw: shaping them (dead zone, response curve) is game feel
//! and therefore lives in the core where it can be tested.

use super::math::Vec2;

/// Where the aim vector came from. Aim assist is tuned per source: a
/// thumbstick needs help, a mouse pointer usually does not.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum AimSource {
    /// An analog stick or digital keys: a direction, imprecise by nature.
    #[default]
    Stick,
    /// A pointer the player places exactly (mouse, touch).
    Pointer,
}

#[derive(Debug, Clone, Copy, PartialEq, Default)]
pub struct InputFrame {
    /// Left stick, raw: each axis in `[-1, 1]`, length up to ~1.41 on
    /// square-gated hardware or keyboards.
    pub move_axis: Vec2,
    /// Aim direction. For a stick, raw axes; for a pointer, the vector from
    /// the player to the cursor (only its direction matters).
    pub aim_axis: Vec2,
    pub aim_source: AimSource,
    /// Fire is held.
    pub fire: bool,
    /// Dash was pressed since the previous step (an edge, not a hold).
    pub dash_pressed: bool,
}

/// Radial dead zone with rescaling.
///
/// Below `inner` the stick reads zero (worn sticks never rest exactly at
/// the centre). Between `inner` and `outer` the magnitude is remapped onto
/// `[0, 1]` so there is no jump at the dead-zone edge; beyond `outer` it is
/// full tilt. The direction is preserved, unlike per-axis dead zones that
/// snap diagonals toward the cardinal directions.
pub fn radial_deadzone(raw: Vec2, inner: f32, outer: f32) -> Vec2 {
    let mag = raw.length();
    if mag <= inner {
        return Vec2::ZERO;
    }
    let scaled = ((mag - inner) / (outer - inner)).min(1.0);
    raw * (scaled / mag)
}

/// Applies the response exponent to an already dead-zoned stick: `1.0` is
/// linear, `2.0` gives finer control near the centre at the cost of a
/// steeper ramp near full tilt.
pub fn response_curve(stick: Vec2, exponent: f32) -> Vec2 {
    let mag = stick.length();
    if mag == 0.0 {
        return Vec2::ZERO;
    }
    stick * (mag.powf(exponent) / mag)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn deadzone_zeroes_small_input() {
        assert_eq!(radial_deadzone(Vec2::new(0.1, 0.05), 0.2, 0.95), Vec2::ZERO);
    }

    #[test]
    fn deadzone_rescales_without_a_jump() {
        let just_past = radial_deadzone(Vec2::new(0.2001, 0.0), 0.2, 0.95);
        assert!(just_past.length() < 0.001);
        let full = radial_deadzone(Vec2::new(0.95, 0.0), 0.2, 0.95);
        assert!((full.length() - 1.0).abs() < 1e-6);
    }

    #[test]
    fn deadzone_keeps_direction_and_clamps_diagonals() {
        let diag = radial_deadzone(Vec2::new(1.0, 1.0), 0.2, 0.95);
        assert!((diag.length() - 1.0).abs() < 1e-6);
        assert!((diag.x - diag.y).abs() < 1e-6);
    }

    #[test]
    fn response_curve_softens_partial_tilt() {
        let half = response_curve(Vec2::new(0.5, 0.0), 2.0);
        assert!((half.x - 0.25).abs() < 1e-6);
        let full = response_curve(Vec2::new(0.0, 1.0), 2.0);
        assert!((full.y - 1.0).abs() < 1e-6);
    }
}
