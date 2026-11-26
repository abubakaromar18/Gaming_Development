//! The player character: position, velocity, facing, dash and health.
//!
//! [`Player::update`] interleaves the dash state machine with ordinary
//! steering inside a single step. If a dash ends 3 ms into a 10 ms step,
//! the remaining 7 ms are spent running, and if a buffered dash becomes
//! ready mid-step it starts right then. The result is that distance,
//! timing and i-frames are the same at 30, 60 or 240 steps per second.

use super::config::{Config, MovementConfig};
use super::dash::{Dash, DashPhase, DashTransition};
use super::math::Vec2;
use super::movement::steer;

/// Player-level happenings, folded into the world's event list.
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum PlayerEvent {
    DashStarted { dir: Vec2 },
    DashEnded,
    DashReady,
}

#[derive(Debug, Clone, PartialEq)]
pub struct Player {
    pub pos: Vec2,
    pub vel: Vec2,
    /// Unit vector the character faces. The world sets it from the aim, or
    /// the move stick when not aiming; a dash with no stick input goes this
    /// way.
    pub facing: Vec2,
    pub dash: Dash,
    pub health: u32,
    /// Mercy invulnerability left after being hit.
    pub hurt_timer: f32,
}

impl Player {
    pub fn new(pos: Vec2, health: u32) -> Player {
        Player {
            pos,
            vel: Vec2::ZERO,
            facing: Vec2::new(1.0, 0.0),
            dash: Dash::new(),
            health,
            hurt_timer: 0.0,
        }
    }

    pub fn is_invulnerable(&self, cfg: &Config) -> bool {
        self.hurt_timer > 0.0 || self.dash.is_invulnerable(&cfg.dash)
    }

    /// Dash speed implied by the configured distance and duration.
    pub fn dash_speed(cfg: &Config) -> f32 {
        cfg.dash.distance / cfg.dash.duration
    }

    /// Points the character along `dir` (ignored while dashing or if zero).
    pub fn face(&mut self, dir: Vec2) {
        if dir != Vec2::ZERO && !self.dash.is_active() {
            self.facing = dir.normalized_or_zero();
        }
    }

    /// Steps the character by `dt`. `stick` is the shaped move input.
    pub fn update(
        &mut self,
        dt: f32,
        stick: Vec2,
        dash_pressed: bool,
        cfg: &Config,
        events: &mut Vec<PlayerEvent>,
    ) {
        self.hurt_timer = (self.hurt_timer - dt).max(0.0);
        if dash_pressed {
            self.dash.press(&cfg.dash);
        }

        let mut remaining = dt;
        // Each pass handles one stretch of time with no phase change in it.
        // Every pass either consumes time or performs a transition that
        // cannot repeat at zero time, so this terminates.
        loop {
            let dir = if stick == Vec2::ZERO {
                self.facing
            } else {
                stick.normalized_or_zero()
            };
            if self.dash.try_start(dir) {
                self.facing = dir;
                self.vel = dir * Self::dash_speed(cfg);
                events.push(PlayerEvent::DashStarted { dir });
            }
            if remaining <= 0.0 {
                break;
            }

            let dashing = matches!(self.dash.phase(), DashPhase::Active { .. });
            let step = self.dash.advance(remaining, &cfg.dash);
            remaining -= step.consumed;

            if dashing {
                // Constant speed for the whole dash: no curve, no steering.
                self.pos += self.vel * step.consumed;
            } else {
                self.run(stick, step.consumed, &cfg.movement);
            }
            self.confine(cfg);

            match step.transition {
                Some(DashTransition::Ended) => {
                    let exit = cfg.movement.max_speed * cfg.dash.exit_speed_fraction;
                    self.vel = self.vel.normalized_or_zero() * exit;
                    events.push(PlayerEvent::DashEnded);
                }
                Some(DashTransition::Ready) => events.push(PlayerEvent::DashReady),
                None => {}
            }
            if step.transition.is_none() && remaining <= 0.0 {
                break;
            }
        }
    }

    /// Normal running: steer, then integrate with the average of old and
    /// new velocity (trapezoidal), which keeps stopping distance exact.
    fn run(&mut self, stick: Vec2, dt: f32, cfg: &MovementConfig) {
        if dt <= 0.0 {
            return;
        }
        let before = self.vel;
        self.vel = steer(self.vel, stick, cfg, dt);
        self.pos += (before + self.vel) * (0.5 * dt);
    }

    /// Keeps the circle inside the arena and kills velocity into the wall,
    /// so pushing against a wall doesn't build up hidden speed.
    fn confine(&mut self, cfg: &Config) {
        let r = cfg.arena.player_radius;
        let (lo_x, hi_x) = (r, cfg.arena.width - r);
        let (lo_y, hi_y) = (r, cfg.arena.height - r);
        if self.pos.x <= lo_x {
            self.pos.x = lo_x;
            self.vel.x = self.vel.x.max(0.0);
        } else if self.pos.x >= hi_x {
            self.pos.x = hi_x;
            self.vel.x = self.vel.x.min(0.0);
        }
        if self.pos.y <= lo_y {
            self.pos.y = lo_y;
            self.vel.y = self.vel.y.max(0.0);
        } else if self.pos.y >= hi_y {
            self.pos.y = hi_y;
            self.vel.y = self.vel.y.min(0.0);
        }
    }
}
