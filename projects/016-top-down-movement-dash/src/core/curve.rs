//! Acceleration curves: the shape of the speed ramp from standstill to top
//! speed.
//!
//! A curve maps normalized ramp time `t` (0 = standing still, 1 = the full
//! `accel_time` has elapsed) to a fraction of top speed. Every curve passes
//! through (0, 0) and (1, 1), so "time to top speed" is a designer-facing
//! number that stays the same whichever feel is chosen; only the shape of
//! the ramp changes.
//!
//! The mover never stores `t`. Each step it *inverts* the curve at the
//! current speed to recover where on the ramp it is, advances by `dt`, and
//! evaluates the curve again. That keeps velocity the single source of
//! truth: a knockback or dash exit that changes speed is automatically
//! picked up at the right point of the ramp.

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum AccelCurve {
    /// Constant acceleration: speed grows evenly. Feels "floaty" at long
    /// accel times.
    Linear,
    /// `1 - (1 - t)^power`: most of the speed arrives immediately, then the
    /// last bit eases in. The usual choice for snappy action games.
    EaseOut { power: f32 },
    /// `t^power`: a slow start that builds up, like a heavy vehicle.
    EaseIn { power: f32 },
}

impl AccelCurve {
    /// Speed fraction at normalized ramp time `t` (clamped to `[0, 1]`).
    pub fn eval(self, t: f32) -> f32 {
        let t = t.clamp(0.0, 1.0);
        match self {
            AccelCurve::Linear => t,
            AccelCurve::EaseOut { power } => 1.0 - (1.0 - t).powf(power),
            AccelCurve::EaseIn { power } => t.powf(power),
        }
    }

    /// Ramp time at which the curve reaches speed fraction `s`.
    pub fn inverse(self, s: f32) -> f32 {
        let s = s.clamp(0.0, 1.0);
        match self {
            AccelCurve::Linear => s,
            AccelCurve::EaseOut { power } => 1.0 - (1.0 - s).powf(1.0 / power),
            AccelCurve::EaseIn { power } => s.powf(1.0 / power),
        }
    }

    pub fn is_valid(self) -> bool {
        match self {
            AccelCurve::Linear => true,
            AccelCurve::EaseOut { power } | AccelCurve::EaseIn { power } => {
                power.is_finite() && power >= 1.0
            }
        }
    }

    pub fn name(self) -> &'static str {
        match self {
            AccelCurve::Linear => "linear",
            AccelCurve::EaseOut { .. } => "ease-out",
            AccelCurve::EaseIn { .. } => "ease-in",
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const CURVES: [AccelCurve; 3] = [
        AccelCurve::Linear,
        AccelCurve::EaseOut { power: 2.5 },
        AccelCurve::EaseIn { power: 2.0 },
    ];

    #[test]
    fn every_curve_spans_zero_to_one() {
        for c in CURVES {
            assert_eq!(c.eval(0.0), 0.0, "{c:?}");
            assert!((c.eval(1.0) - 1.0).abs() < 1e-6, "{c:?}");
        }
    }

    #[test]
    fn inverse_round_trips() {
        for c in CURVES {
            for i in 0..=20 {
                let t = i as f32 / 20.0;
                assert!((c.inverse(c.eval(t)) - t).abs() < 1e-4, "{c:?} at {t}");
            }
        }
    }

    #[test]
    fn curves_are_monotonic() {
        for c in CURVES {
            let mut prev = -1.0;
            for i in 0..=100 {
                let v = c.eval(i as f32 / 100.0);
                assert!(v >= prev, "{c:?}");
                prev = v;
            }
        }
    }
}
