//! Every tunable number in one place, validated up front so the simulation
//! never has to guard against nonsense values mid-game.
//!
//! Units: pixels, seconds, radians.

use std::fmt;

use super::curve::AccelCurve;

#[derive(Debug, Clone, PartialEq)]
pub struct MovementConfig {
    /// Top running speed at full stick tilt.
    pub max_speed: f32,
    /// Seconds from standstill to `max_speed` (the curve shapes the ramp).
    pub accel_time: f32,
    /// Seconds from `max_speed` to standstill once the stick is released.
    pub decel_time: f32,
    /// How much harder the character brakes velocity that points *away*
    /// from the stick (reversing, strafing sideways) compared to releasing
    /// the stick. Above 1 makes turns crisp instead of drifty.
    pub turn_brake_multiplier: f32,
    pub curve: AccelCurve,
    pub deadzone_inner: f32,
    pub deadzone_outer: f32,
    /// Stick magnitude exponent; 1 = linear.
    pub response_exponent: f32,
}

#[derive(Debug, Clone, PartialEq)]
pub struct DashConfig {
    /// Distance covered by one dash, independent of frame rate.
    pub distance: f32,
    /// Seconds the dash lasts.
    pub duration: f32,
    /// Invulnerability window, in seconds since the dash began. The end
    /// may extend past `duration` to forgive late dodges.
    pub iframe_start: f32,
    pub iframe_end: f32,
    /// Seconds after the dash ends before another can start.
    pub cooldown: f32,
    /// A press this many seconds before the dash is ready still fires it.
    pub buffer_window: f32,
    /// Fraction of `max_speed` kept when the dash ends, so it flows back
    /// into running instead of stopping dead.
    pub exit_speed_fraction: f32,
}

#[derive(Debug, Clone, PartialEq)]
pub struct AimAssistConfig {
    pub enabled: bool,
    /// Targets farther than this are ignored.
    pub range: f32,
    /// Half-angle of the cone around the raw aim in which targets attract.
    pub cone_half_angle: f32,
    /// 0 = no correction, 1 = snap fully onto the chosen target.
    pub stick_strength: f32,
    pub pointer_strength: f32,
    /// Score discount for the target assisted last step. Prevents the
    /// assist flickering between two similarly placed targets.
    pub stickiness: f32,
    /// How much distance counts against a target relative to angle.
    pub distance_weight: f32,
}

#[derive(Debug, Clone, PartialEq)]
pub struct CombatConfig {
    pub max_health: u32,
    /// Mercy invulnerability after taking a hit.
    pub hurt_invulnerability: f32,
    pub fire_interval: f32,
    pub bullet_speed: f32,
    pub bullet_lifetime: f32,
    pub bullet_radius: f32,
}

#[derive(Debug, Clone, PartialEq)]
pub struct ArenaConfig {
    pub width: f32,
    pub height: f32,
    pub player_radius: f32,
    pub drone_count: usize,
    pub drone_radius: f32,
    pub drone_speed: f32,
    pub drone_health: u32,
    /// Drones pick a new heading after a random interval in this range.
    pub drone_turn_min: f32,
    pub drone_turn_max: f32,
    pub drone_respawn_delay: f32,
    /// Respawning drones keep at least this far from the player.
    pub drone_spawn_clearance: f32,
}

#[derive(Debug, Clone, PartialEq, Default)]
pub struct Config {
    pub movement: MovementConfig,
    pub dash: DashConfig,
    pub aim: AimAssistConfig,
    pub combat: CombatConfig,
    pub arena: ArenaConfig,
}

impl Default for MovementConfig {
    fn default() -> Self {
        MovementConfig {
            max_speed: 320.0,
            accel_time: 0.14,
            decel_time: 0.10,
            turn_brake_multiplier: 2.0,
            curve: AccelCurve::EaseOut { power: 2.0 },
            deadzone_inner: 0.18,
            deadzone_outer: 0.95,
            response_exponent: 1.5,
        }
    }
}

impl Default for DashConfig {
    fn default() -> Self {
        DashConfig {
            distance: 150.0,
            duration: 0.15,
            iframe_start: 0.0,
            iframe_end: 0.20,
            cooldown: 0.45,
            buffer_window: 0.12,
            exit_speed_fraction: 0.6,
        }
    }
}

impl Default for AimAssistConfig {
    fn default() -> Self {
        AimAssistConfig {
            enabled: true,
            range: 560.0,
            cone_half_angle: 0.30,
            stick_strength: 0.85,
            pointer_strength: 0.0,
            stickiness: 0.35,
            distance_weight: 0.3,
        }
    }
}

impl Default for CombatConfig {
    fn default() -> Self {
        CombatConfig {
            max_health: 5,
            hurt_invulnerability: 0.8,
            fire_interval: 0.11,
            bullet_speed: 900.0,
            bullet_lifetime: 0.7,
            bullet_radius: 4.0,
        }
    }
}

impl Default for ArenaConfig {
    fn default() -> Self {
        ArenaConfig {
            width: 960.0,
            height: 640.0,
            player_radius: 14.0,
            drone_count: 6,
            drone_radius: 16.0,
            drone_speed: 70.0,
            drone_health: 3,
            drone_turn_min: 0.8,
            drone_turn_max: 2.2,
            drone_respawn_delay: 1.5,
            drone_spawn_clearance: 220.0,
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

fn positive(name: &str, v: f32) -> Result<(), ConfigError> {
    if v.is_finite() && v > 0.0 {
        Ok(())
    } else {
        Err(ConfigError(format!(
            "{name} must be a positive number, got {v}"
        )))
    }
}

fn non_negative(name: &str, v: f32) -> Result<(), ConfigError> {
    if v.is_finite() && v >= 0.0 {
        Ok(())
    } else {
        Err(ConfigError(format!("{name} must be zero or more, got {v}")))
    }
}

fn unit(name: &str, v: f32) -> Result<(), ConfigError> {
    if (0.0..=1.0).contains(&v) {
        Ok(())
    } else {
        Err(ConfigError(format!("{name} must be in [0, 1], got {v}")))
    }
}

impl MovementConfig {
    pub fn validate(&self) -> Result<(), ConfigError> {
        positive("movement.max_speed", self.max_speed)?;
        positive("movement.accel_time", self.accel_time)?;
        positive("movement.decel_time", self.decel_time)?;
        positive("movement.response_exponent", self.response_exponent)?;
        if !(self.turn_brake_multiplier.is_finite() && self.turn_brake_multiplier >= 1.0) {
            return Err(ConfigError(
                "movement.turn_brake_multiplier must be at least 1".into(),
            ));
        }
        if !self.curve.is_valid() {
            return Err(ConfigError(
                "movement.curve power must be at least 1".into(),
            ));
        }
        if !(0.0 <= self.deadzone_inner
            && self.deadzone_inner < self.deadzone_outer
            && self.deadzone_outer <= 1.5)
        {
            return Err(ConfigError(
                "movement dead zone needs 0 <= inner < outer <= 1.5".into(),
            ));
        }
        Ok(())
    }
}

impl DashConfig {
    pub fn validate(&self) -> Result<(), ConfigError> {
        positive("dash.distance", self.distance)?;
        positive("dash.duration", self.duration)?;
        non_negative("dash.cooldown", self.cooldown)?;
        non_negative("dash.buffer_window", self.buffer_window)?;
        non_negative("dash.iframe_start", self.iframe_start)?;
        unit("dash.exit_speed_fraction", self.exit_speed_fraction)?;
        if !(self.iframe_end.is_finite() && self.iframe_end >= self.iframe_start) {
            return Err(ConfigError(
                "dash.iframe_end must not precede iframe_start".into(),
            ));
        }
        if self.iframe_end > self.duration + self.cooldown {
            return Err(ConfigError(
                "dash.iframe_end must not outlast the cooldown, or dashes could chain invulnerability".into(),
            ));
        }
        Ok(())
    }
}

impl AimAssistConfig {
    pub fn validate(&self) -> Result<(), ConfigError> {
        positive("aim.range", self.range)?;
        if !(self.cone_half_angle > 0.0 && self.cone_half_angle < std::f32::consts::FRAC_PI_2) {
            return Err(ConfigError(
                "aim.cone_half_angle must be in (0, PI/2)".into(),
            ));
        }
        unit("aim.stick_strength", self.stick_strength)?;
        unit("aim.pointer_strength", self.pointer_strength)?;
        unit("aim.stickiness", self.stickiness)?;
        non_negative("aim.distance_weight", self.distance_weight)
    }
}

impl CombatConfig {
    pub fn validate(&self) -> Result<(), ConfigError> {
        if self.max_health == 0 {
            return Err(ConfigError("combat.max_health must be at least 1".into()));
        }
        non_negative("combat.hurt_invulnerability", self.hurt_invulnerability)?;
        positive("combat.fire_interval", self.fire_interval)?;
        positive("combat.bullet_speed", self.bullet_speed)?;
        positive("combat.bullet_lifetime", self.bullet_lifetime)?;
        positive("combat.bullet_radius", self.bullet_radius)
    }
}

impl ArenaConfig {
    pub fn validate(&self) -> Result<(), ConfigError> {
        positive("arena.width", self.width)?;
        positive("arena.height", self.height)?;
        positive("arena.player_radius", self.player_radius)?;
        positive("arena.drone_radius", self.drone_radius)?;
        non_negative("arena.drone_speed", self.drone_speed)?;
        non_negative("arena.drone_respawn_delay", self.drone_respawn_delay)?;
        non_negative("arena.drone_spawn_clearance", self.drone_spawn_clearance)?;
        if self.drone_health == 0 {
            return Err(ConfigError("arena.drone_health must be at least 1".into()));
        }
        if !(self.drone_turn_min > 0.0 && self.drone_turn_min <= self.drone_turn_max) {
            return Err(ConfigError(
                "arena drone turn interval needs 0 < min <= max".into(),
            ));
        }
        let smallest = self.width.min(self.height);
        if 2.0 * self.player_radius.max(self.drone_radius) >= smallest {
            return Err(ConfigError("arena is too small for its actors".into()));
        }
        Ok(())
    }
}

impl Config {
    pub fn validate(&self) -> Result<(), ConfigError> {
        self.movement.validate()?;
        self.dash.validate()?;
        self.aim.validate()?;
        self.combat.validate()?;
        self.arena.validate()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn defaults_are_valid() {
        assert_eq!(Config::default().validate(), Ok(()));
    }

    #[test]
    fn rejects_inverted_deadzone() {
        let mut c = Config::default();
        c.movement.deadzone_inner = 0.9;
        c.movement.deadzone_outer = 0.5;
        assert!(c.validate().is_err());
    }

    #[test]
    fn rejects_iframes_that_outlast_the_cooldown() {
        let mut c = Config::default();
        c.dash.iframe_end = c.dash.duration + c.dash.cooldown + 0.01;
        let err = c.validate().unwrap_err();
        assert!(err.to_string().contains("iframe_end"));
    }

    #[test]
    fn rejects_nan() {
        let mut c = Config::default();
        c.movement.max_speed = f32::NAN;
        assert!(c.validate().is_err());
    }
}
