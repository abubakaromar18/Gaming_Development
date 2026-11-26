//! Pure simulation: no window, no clock, no global RNG.

pub mod aim_assist;
pub mod config;
pub mod curve;
pub mod dash;
pub mod input;
pub mod math;
pub mod movement;
pub mod player;
pub mod rng;
pub mod world;

pub use aim_assist::{AimCandidate, AimSolution};
pub use config::{
    AimAssistConfig, ArenaConfig, CombatConfig, Config, ConfigError, DashConfig, MovementConfig,
};
pub use curve::AccelCurve;
pub use dash::{Dash, DashPhase};
pub use input::{AimSource, InputFrame};
pub use math::Vec2;
pub use player::Player;
pub use rng::{RandomSource, SplitMix64};
pub use world::{Bullet, Drone, Event, World};
