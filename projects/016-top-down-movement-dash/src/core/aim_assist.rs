//! Aim assist: bend an imprecise aim direction toward the target the player
//! most plausibly meant.
//!
//! Three ideas, as used (with many variations) by console shooters:
//!
//! 1. **Candidate cone.** Only targets within range and inside a cone around
//!    the raw aim are considered. The cone is widened per target by its
//!    angular radius, so a big or nearby target is easier to "touch".
//! 2. **Scoring.** Candidates are ranked by how far off-angle they are,
//!    plus a smaller distance penalty. The previously assisted target gets a
//!    discount (*stickiness*): without it, two targets at similar angles
//!    make the assist flicker between them every frame.
//! 3. **Magnetism.** The aim rotates toward the winner by a fraction of the
//!    angle between them. Strength 1 snaps; lower values only nudge, so the
//!    player still feels in control.

use super::config::AimAssistConfig;
use super::math::Vec2;

#[derive(Debug, Clone, Copy, PartialEq)]
pub struct AimCandidate {
    pub id: u32,
    pub pos: Vec2,
    pub radius: f32,
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub struct AimSolution {
    /// Final aim direction (unit vector).
    pub dir: Vec2,
    /// Target the assist locked onto, if any.
    pub target: Option<u32>,
}

/// Picks a target and returns the corrected aim. `raw_dir` must be a unit
/// vector; `strength` is the per-source strength from the config.
pub fn solve(
    origin: Vec2,
    raw_dir: Vec2,
    candidates: &[AimCandidate],
    previous: Option<u32>,
    strength: f32,
    cfg: &AimAssistConfig,
) -> AimSolution {
    let unassisted = AimSolution {
        dir: raw_dir,
        target: None,
    };
    if !cfg.enabled || strength <= 0.0 {
        return unassisted;
    }

    let mut best: Option<(f32, f32, u32)> = None; // (score, signed angle, id)
    for c in candidates {
        let to = c.pos - origin;
        let dist = to.length();
        if dist > cfg.range || dist <= c.radius {
            continue;
        }
        let angle = raw_dir.angle_to(to);
        let angular_radius = (c.radius / dist).asin();
        let off = (angle.abs() - angular_radius).max(0.0);
        if off > cfg.cone_half_angle {
            continue;
        }
        let mut score = off / cfg.cone_half_angle + cfg.distance_weight * dist / cfg.range;
        if previous == Some(c.id) {
            score *= 1.0 - cfg.stickiness;
        }
        if best.is_none_or(|(s, _, _)| score < s) {
            best = Some((score, angle, c.id));
        }
    }

    match best {
        Some((_, angle, id)) => AimSolution {
            dir: raw_dir.rotate(angle * strength),
            target: Some(id),
        },
        None => unassisted,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn disabled_assist_is_a_no_op() {
        let cfg = AimAssistConfig {
            enabled: false,
            ..AimAssistConfig::default()
        };
        let c = [AimCandidate {
            id: 0,
            pos: Vec2::new(100.0, 10.0),
            radius: 10.0,
        }];
        let s = solve(Vec2::ZERO, Vec2::new(1.0, 0.0), &c, None, 1.0, &cfg);
        assert_eq!(s.target, None);
        assert_eq!(s.dir, Vec2::new(1.0, 0.0));
    }

    #[test]
    fn origin_inside_a_target_is_ignored() {
        let cfg = AimAssistConfig::default();
        let c = [AimCandidate {
            id: 0,
            pos: Vec2::new(2.0, 0.0),
            radius: 10.0,
        }];
        let s = solve(Vec2::ZERO, Vec2::new(1.0, 0.0), &c, None, 1.0, &cfg);
        assert_eq!(s.target, None);
    }
}
