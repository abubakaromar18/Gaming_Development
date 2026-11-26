//! The dash: a short, fixed-distance burst with invulnerability frames,
//! a cooldown, and an input buffer.
//!
//! ```text
//!   press ──► Ready ──► Active ──(duration)──► Cooldown ──(cooldown)──► Ready
//!                       │◄──── i-frames ────►│   (window may run past the end)
//! ```
//!
//! The state machine only tracks *time*. Moving the character is the
//! player's job; [`Dash::advance`] tells it how many seconds of the step
//! were spent dashing so the distance covered never depends on frame rate.

use super::config::DashConfig;
use super::math::Vec2;

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum DashPhase {
    Ready,
    Active { elapsed: f32, dir: Vec2 },
    Cooldown { remaining: f32 },
}

/// What happened during one call to [`Dash::advance`].
#[derive(Debug, Clone, Copy, PartialEq)]
pub struct DashStep {
    /// Seconds consumed by this call.
    pub consumed: f32,
    /// Set when the phase changed at the end of the consumed time.
    pub transition: Option<DashTransition>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DashTransition {
    Ended,
    Ready,
}

#[derive(Debug, Clone, PartialEq)]
pub struct Dash {
    phase: DashPhase,
    /// Seconds since the current/last dash began; `None` once Ready again.
    since_start: Option<f32>,
    /// Seconds left on a buffered press, if any.
    buffered: Option<f32>,
}

impl Default for Dash {
    fn default() -> Self {
        Dash::new()
    }
}

impl Dash {
    pub fn new() -> Dash {
        Dash {
            phase: DashPhase::Ready,
            since_start: None,
            buffered: None,
        }
    }

    pub fn phase(&self) -> DashPhase {
        self.phase
    }

    pub fn is_active(&self) -> bool {
        matches!(self.phase, DashPhase::Active { .. })
    }

    pub fn has_buffered_press(&self) -> bool {
        self.buffered.is_some()
    }

    /// Records a press. It fires as soon as the dash is ready, provided
    /// that happens within the buffer window.
    pub fn press(&mut self, cfg: &DashConfig) {
        self.buffered = Some(cfg.buffer_window);
    }

    /// True while inside the invulnerability window.
    pub fn is_invulnerable(&self, cfg: &DashConfig) -> bool {
        self.since_start
            .is_some_and(|t| t >= cfg.iframe_start && t < cfg.iframe_end)
    }

    /// 0 while dashing, rising to 1 when ready: handy for a HUD ring.
    pub fn readiness(&self, cfg: &DashConfig) -> f32 {
        match self.phase {
            DashPhase::Ready => 1.0,
            DashPhase::Active { .. } => 0.0,
            DashPhase::Cooldown { remaining } if cfg.cooldown > 0.0 => {
                1.0 - remaining / cfg.cooldown
            }
            DashPhase::Cooldown { .. } => 1.0,
        }
    }

    /// Starts a dash in `dir` if one is buffered and the dash is ready.
    /// `dir` must be a unit vector.
    pub fn try_start(&mut self, dir: Vec2) -> bool {
        if self.phase != DashPhase::Ready || self.buffered.is_none() {
            return false;
        }
        self.buffered = None;
        self.since_start = Some(0.0);
        self.phase = DashPhase::Active { elapsed: 0.0, dir };
        true
    }

    /// Advances by up to `dt`, stopping early at a phase boundary so the
    /// caller can react (start the next dash, change velocity) at the exact
    /// moment it happens. Call repeatedly until all of `dt` is consumed.
    pub fn advance(&mut self, dt: f32, cfg: &DashConfig) -> DashStep {
        let (consumed, transition) = match self.phase {
            DashPhase::Ready => (dt, None),
            DashPhase::Active { elapsed, dir } => {
                let left = cfg.duration - elapsed;
                if dt >= left {
                    self.phase = DashPhase::Cooldown {
                        remaining: cfg.cooldown,
                    };
                    (left, Some(DashTransition::Ended))
                } else {
                    self.phase = DashPhase::Active {
                        elapsed: elapsed + dt,
                        dir,
                    };
                    (dt, None)
                }
            }
            DashPhase::Cooldown { remaining } => {
                if dt >= remaining {
                    self.phase = DashPhase::Ready;
                    (remaining, Some(DashTransition::Ready))
                } else {
                    self.phase = DashPhase::Cooldown {
                        remaining: remaining - dt,
                    };
                    (dt, None)
                }
            }
        };

        if let Some(t) = self.since_start.as_mut() {
            *t += consumed;
        }
        if transition == Some(DashTransition::Ready) {
            self.since_start = None;
        }
        // A press with no time left in its window expires. A press made
        // while Ready survives a zero-length advance so it can still fire.
        if let Some(left) = self.buffered {
            let left = left - consumed;
            self.buffered = if left < 0.0 { None } else { Some(left) };
        }

        DashStep {
            consumed,
            transition,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn cannot_start_without_a_press() {
        let mut d = Dash::new();
        assert!(!d.try_start(Vec2::new(1.0, 0.0)));
    }

    #[test]
    fn advance_stops_at_the_end_of_the_dash() {
        let cfg = DashConfig::default();
        let mut d = Dash::new();
        d.press(&cfg);
        assert!(d.try_start(Vec2::new(1.0, 0.0)));
        let step = d.advance(1.0, &cfg);
        assert!((step.consumed - cfg.duration).abs() < 1e-6);
        assert_eq!(step.transition, Some(DashTransition::Ended));
    }

    #[test]
    fn readiness_rises_through_cooldown() {
        let cfg = DashConfig::default();
        let mut d = Dash::new();
        d.press(&cfg);
        d.try_start(Vec2::new(1.0, 0.0));
        d.advance(cfg.duration, &cfg);
        assert_eq!(d.readiness(&cfg), 0.0);
        d.advance(cfg.cooldown / 2.0, &cfg);
        assert!((d.readiness(&cfg) - 0.5).abs() < 1e-5);
    }
}
