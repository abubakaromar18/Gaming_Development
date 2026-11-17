//! Tunable rules, validated up front so the simulation never has to guard
//! against nonsense values mid-game.

use std::fmt;

#[derive(Debug, Clone, PartialEq)]
pub struct Config {
    /// Seconds a grounded piece may rest before it locks.
    pub lock_delay: f64,
    /// Moves/rotations that may reset the lock timer per piece ("extended
    /// placement"). Once spent, the timer keeps running however much the
    /// player wiggles, so a piece can't be stalled forever.
    pub max_lock_resets: u32,
    /// Delayed Auto Shift, seconds.
    pub das: f64,
    /// Auto Repeat Rate, seconds per column; 0 = instant to the wall.
    pub arr: f64,
    /// Gravity multiplier while soft drop is held.
    pub soft_drop_factor: f64,
    /// Number of upcoming pieces shown.
    pub preview_count: usize,
    pub start_level: u32,
    /// Lines needed per level.
    pub lines_per_level: u32,
}

impl Default for Config {
    fn default() -> Self {
        Config {
            lock_delay: 0.5,
            max_lock_resets: 15,
            das: 0.167,
            arr: 0.033,
            soft_drop_factor: 20.0,
            preview_count: 5,
            start_level: 1,
            lines_per_level: 10,
        }
    }
}

#[derive(Debug, Clone, PartialEq)]
pub struct ConfigError(pub String);

impl fmt::Display for ConfigError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "invalid config: {}", self.0)
    }
}

impl std::error::Error for ConfigError {}

impl Config {
    pub fn validate(&self) -> Result<(), ConfigError> {
        let finite_non_neg = |v: f64| v.is_finite() && v >= 0.0;
        if !finite_non_neg(self.lock_delay) {
            return Err(ConfigError(
                "lock_delay must be a non-negative number".into(),
            ));
        }
        if !finite_non_neg(self.das) || !finite_non_neg(self.arr) {
            return Err(ConfigError(
                "das and arr must be non-negative numbers".into(),
            ));
        }
        if !(self.soft_drop_factor.is_finite() && self.soft_drop_factor >= 1.0) {
            return Err(ConfigError("soft_drop_factor must be at least 1".into()));
        }
        if self.preview_count > 7 {
            return Err(ConfigError("preview_count must be at most 7".into()));
        }
        if !(1..=30).contains(&self.start_level) {
            return Err(ConfigError("start_level must be between 1 and 30".into()));
        }
        if self.lines_per_level == 0 {
            return Err(ConfigError("lines_per_level must be positive".into()));
        }
        Ok(())
    }
}

/// Guideline gravity: seconds per row at `level`,
/// `(0.8 - (level - 1) * 0.007) ^ (level - 1)`. Level 1 is one row per
/// second; by level 15 pieces fall about one row per frame. The curve's
/// base reaches zero past level 115, so levels are capped there.
pub fn seconds_per_row(level: u32) -> f64 {
    let l = level.clamp(1, 115) as f64 - 1.0;
    (0.8 - l * 0.007).powf(l)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn default_is_valid() {
        assert!(Config::default().validate().is_ok());
    }

    #[test]
    fn rejects_bad_values() {
        let bad = [
            Config {
                lock_delay: -1.0,
                ..Config::default()
            },
            Config {
                das: f64::NAN,
                ..Config::default()
            },
            Config {
                soft_drop_factor: 0.5,
                ..Config::default()
            },
            Config {
                preview_count: 8,
                ..Config::default()
            },
            Config {
                start_level: 0,
                ..Config::default()
            },
            Config {
                lines_per_level: 0,
                ..Config::default()
            },
        ];
        for c in bad {
            assert!(c.validate().is_err(), "{c:?}");
        }
    }

    #[test]
    fn gravity_speeds_up_with_level() {
        assert!((seconds_per_row(1) - 1.0).abs() < 1e-9);
        let mut prev = seconds_per_row(1);
        for level in 2..=20 {
            let s = seconds_per_row(level);
            assert!(s < prev);
            prev = s;
        }
        assert!(seconds_per_row(15) < 1.0 / 50.0);
    }
}
