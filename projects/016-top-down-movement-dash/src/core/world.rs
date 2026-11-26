//! The arena: the player, wandering training drones, and bullets.
//!
//! [`World::step`] is the whole simulation. It takes an explicit `dt` and an
//! [`InputFrame`], and returns the [`Event`]s that happened so the front end
//! can play feedback (screen shake, flashes) without the core knowing how.
//! Drones hurt on contact, so the dash's i-frames are how you pass through
//! them; shooting them shows the aim assist at work.

use super::aim_assist::{self, AimCandidate, AimSolution};
use super::config::{Config, ConfigError};
use super::input::{radial_deadzone, response_curve, AimSource, InputFrame};
use super::math::Vec2;
use super::player::{Player, PlayerEvent};
use super::rng::RandomSource;

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum Event {
    DashStarted {
        dir: Vec2,
    },
    DashEnded,
    DashReady,
    /// The player passed through a drone thanks to dash i-frames.
    Dodged {
        drone: u32,
    },
    PlayerHit {
        health_left: u32,
    },
    PlayerDefeated,
    Shot {
        dir: Vec2,
    },
    DroneHit {
        drone: u32,
    },
    DroneDestroyed {
        drone: u32,
    },
    DroneRespawned {
        drone: u32,
    },
}

impl From<PlayerEvent> for Event {
    fn from(e: PlayerEvent) -> Event {
        match e {
            PlayerEvent::DashStarted { dir } => Event::DashStarted { dir },
            PlayerEvent::DashEnded => Event::DashEnded,
            PlayerEvent::DashReady => Event::DashReady,
        }
    }
}

#[derive(Debug, Clone, PartialEq)]
pub struct Drone {
    pub id: u32,
    pub pos: Vec2,
    /// Unit heading; speed comes from the config.
    pub heading: Vec2,
    pub health: u32,
    pub alive: bool,
    turn_timer: f32,
    respawn_timer: f32,
}

#[derive(Debug, Clone, PartialEq)]
pub struct Bullet {
    pub pos: Vec2,
    pub vel: Vec2,
    pub life: f32,
}

pub struct World {
    cfg: Config,
    rng: Box<dyn RandomSource>,
    player: Player,
    drones: Vec<Drone>,
    bullets: Vec<Bullet>,
    aim: Option<AimSolution>,
    fire_cooldown: f32,
    /// Drones already dodged during the current dash, so one pass-through
    /// reports once rather than on every overlapping step.
    dodged: Vec<u32>,
    score: u32,
    time: f64,
}

impl World {
    pub fn new(cfg: Config, rng: Box<dyn RandomSource>) -> Result<World, ConfigError> {
        cfg.validate()?;
        let centre = Vec2::new(cfg.arena.width / 2.0, cfg.arena.height / 2.0);
        let player = Player::new(centre, cfg.combat.max_health);
        let mut world = World {
            cfg,
            rng,
            player,
            drones: Vec::new(),
            bullets: Vec::new(),
            aim: None,
            fire_cooldown: 0.0,
            dodged: Vec::new(),
            score: 0,
            time: 0.0,
        };
        for id in 0..world.cfg.arena.drone_count as u32 {
            let drone = world.spawn_drone(id);
            world.drones.push(drone);
        }
        Ok(world)
    }

    pub fn config(&self) -> &Config {
        &self.cfg
    }

    /// Changes tuning between steps (the demo toggles assist and curves).
    pub fn set_config(&mut self, cfg: Config) -> Result<(), ConfigError> {
        cfg.validate()?;
        self.cfg = cfg;
        Ok(())
    }

    /// Scenario setup: moves a drone and points it along `heading`. Used by
    /// tests and practice drills to build exact situations; returns false
    /// for an unknown id.
    pub fn place_drone(&mut self, id: u32, pos: Vec2, heading: Vec2) -> bool {
        match self.drones.iter_mut().find(|d| d.id == id) {
            Some(d) => {
                d.pos = pos;
                d.heading = heading.normalized_or_zero();
                true
            }
            None => false,
        }
    }

    pub fn player(&self) -> &Player {
        &self.player
    }

    pub fn drones(&self) -> &[Drone] {
        &self.drones
    }

    pub fn bullets(&self) -> &[Bullet] {
        &self.bullets
    }

    /// The aim used this step, or `None` when the player isn't aiming.
    pub fn aim(&self) -> Option<AimSolution> {
        self.aim
    }

    pub fn score(&self) -> u32 {
        self.score
    }

    pub fn time(&self) -> f64 {
        self.time
    }

    pub fn is_over(&self) -> bool {
        self.player.health == 0
    }

    /// Advances the simulation. A negative or non-finite `dt` is treated as
    /// zero: the clock belongs to the caller, and a bad frame should not
    /// corrupt the state.
    pub fn step(&mut self, dt: f32, input: &InputFrame) -> Vec<Event> {
        let dt = if dt.is_finite() && dt > 0.0 { dt } else { 0.0 };
        let mut events = Vec::new();
        if self.is_over() {
            return events;
        }
        self.time += dt as f64;

        let m = &self.cfg.movement;
        let stick = response_curve(
            radial_deadzone(input.move_axis, m.deadzone_inner, m.deadzone_outer),
            m.response_exponent,
        );
        let raw_aim = self.raw_aim(input);
        self.player.face(raw_aim.unwrap_or(stick));

        let mut player_events = Vec::new();
        self.player
            .update(dt, stick, input.dash_pressed, &self.cfg, &mut player_events);
        for e in player_events {
            if let PlayerEvent::DashStarted { .. } = e {
                self.dodged.clear();
            }
            events.push(e.into());
        }

        self.aim = raw_aim.map(|dir| self.assist(dir, input.aim_source));
        self.fire(dt, input, &mut events);
        self.update_drones(dt, &mut events);
        self.update_bullets(dt, &mut events);
        self.resolve_contacts(&mut events);
        events
    }

    /// Aim direction before assist, or `None` if the player isn't aiming.
    fn raw_aim(&self, input: &InputFrame) -> Option<Vec2> {
        let dir = match input.aim_source {
            AimSource::Stick => {
                let m = &self.cfg.movement;
                radial_deadzone(input.aim_axis, m.deadzone_inner, m.deadzone_outer)
            }
            AimSource::Pointer => input.aim_axis,
        }
        .normalized_or_zero();
        (dir != Vec2::ZERO).then_some(dir)
    }

    fn assist(&self, raw: Vec2, source: AimSource) -> AimSolution {
        let candidates: Vec<AimCandidate> = self
            .drones
            .iter()
            .filter(|d| d.alive)
            .map(|d| AimCandidate {
                id: d.id,
                pos: d.pos,
                radius: self.cfg.arena.drone_radius,
            })
            .collect();
        let strength = match source {
            AimSource::Stick => self.cfg.aim.stick_strength,
            AimSource::Pointer => self.cfg.aim.pointer_strength,
        };
        let previous = self.aim.and_then(|a| a.target);
        aim_assist::solve(
            self.player.pos,
            raw,
            &candidates,
            previous,
            strength,
            &self.cfg.aim,
        )
    }

    /// Twin-stick convention: tilting the aim stick fires; a pointer needs
    /// the fire button. The cooldown carries its remainder across steps so
    /// the fire rate is exact at any step size.
    fn fire(&mut self, dt: f32, input: &InputFrame, events: &mut Vec<Event>) {
        self.fire_cooldown -= dt;
        let firing = match (self.aim, input.aim_source) {
            (Some(_), AimSource::Stick) => true,
            (Some(_), AimSource::Pointer) => input.fire,
            (None, _) => false,
        };
        if !firing {
            self.fire_cooldown = self.fire_cooldown.max(0.0);
            return;
        }
        let dir = self.aim.map_or(self.player.facing, |a| a.dir);
        let c = &self.cfg.combat;
        while self.fire_cooldown <= 0.0 {
            // Bullets fired "late" inside this step are advanced by the
            // time they've already existed, so streams stay evenly spaced.
            let age = -self.fire_cooldown;
            let muzzle = self.player.pos + dir * self.cfg.arena.player_radius;
            self.bullets.push(Bullet {
                pos: muzzle + dir * (c.bullet_speed * age),
                vel: dir * c.bullet_speed,
                life: c.bullet_lifetime - age,
            });
            self.fire_cooldown += c.fire_interval;
            events.push(Event::Shot { dir });
        }
    }

    fn update_drones(&mut self, dt: f32, events: &mut Vec<Event>) {
        let a = self.cfg.arena.clone();
        for i in 0..self.drones.len() {
            if !self.drones[i].alive {
                self.drones[i].respawn_timer -= dt;
                if self.drones[i].respawn_timer <= 0.0 {
                    let id = self.drones[i].id;
                    self.drones[i] = self.spawn_drone(id);
                    events.push(Event::DroneRespawned { drone: id });
                }
                continue;
            }

            self.drones[i].turn_timer -= dt;
            if self.drones[i].turn_timer <= 0.0 {
                self.drones[i].heading =
                    Vec2::from_angle(self.rng.range(0.0, std::f32::consts::TAU));
                self.drones[i].turn_timer = self.rng.range(a.drone_turn_min, a.drone_turn_max);
            }

            let d = &mut self.drones[i];
            d.pos += d.heading * (a.drone_speed * dt);
            // Bounce off the walls by reflecting the heading.
            let r = a.drone_radius;
            if d.pos.x < r || d.pos.x > a.width - r {
                d.heading.x = -d.heading.x;
                d.pos.x = d.pos.x.clamp(r, a.width - r);
            }
            if d.pos.y < r || d.pos.y > a.height - r {
                d.heading.y = -d.heading.y;
                d.pos.y = d.pos.y.clamp(r, a.height - r);
            }
        }
    }

    fn update_bullets(&mut self, dt: f32, events: &mut Vec<Event>) {
        let (w, h) = (self.cfg.arena.width, self.cfg.arena.height);
        let hit_dist = self.cfg.arena.drone_radius + self.cfg.combat.bullet_radius;
        let respawn = self.cfg.arena.drone_respawn_delay;
        let drones = &mut self.drones;
        let score = &mut self.score;

        self.bullets.retain_mut(|b| {
            b.pos += b.vel * dt;
            b.life -= dt;
            if b.life <= 0.0 || b.pos.x < 0.0 || b.pos.y < 0.0 || b.pos.x > w || b.pos.y > h {
                return false;
            }
            let Some(d) = drones
                .iter_mut()
                .find(|d| d.alive && d.pos.distance(b.pos) <= hit_dist)
            else {
                return true;
            };
            d.health -= 1;
            events.push(Event::DroneHit { drone: d.id });
            if d.health == 0 {
                d.alive = false;
                d.respawn_timer = respawn;
                *score += 1;
                events.push(Event::DroneDestroyed { drone: d.id });
            }
            false
        });
    }

    fn resolve_contacts(&mut self, events: &mut Vec<Event>) {
        let reach = self.cfg.arena.player_radius + self.cfg.arena.drone_radius;
        for i in 0..self.drones.len() {
            let d = &self.drones[i];
            if !d.alive || d.pos.distance(self.player.pos) > reach {
                continue;
            }
            if self.player.dash.is_invulnerable(&self.cfg.dash) {
                if !self.dodged.contains(&d.id) {
                    self.dodged.push(d.id);
                    events.push(Event::Dodged { drone: d.id });
                }
            } else if self.player.hurt_timer <= 0.0 {
                let away = (self.player.pos - d.pos).normalized_or_zero();
                self.player.health -= 1;
                self.player.hurt_timer = self.cfg.combat.hurt_invulnerability;
                self.player.vel = away * self.cfg.movement.max_speed;
                events.push(Event::PlayerHit {
                    health_left: self.player.health,
                });
                if self.player.health == 0 {
                    events.push(Event::PlayerDefeated);
                    return;
                }
            }
        }
    }

    /// A fresh drone at a random spot clear of the player. Takes the best of
    /// a few samples rather than looping until one fits, so a tiny arena
    /// can never hang the game.
    fn spawn_drone(&mut self, id: u32) -> Drone {
        let a = &self.cfg.arena;
        let r = a.drone_radius;
        let mut best = Vec2::new(r, r);
        let mut best_dist = -1.0;
        for _ in 0..16 {
            let p = Vec2::new(
                self.rng.range(r, a.width - r),
                self.rng.range(r, a.height - r),
            );
            let dist = p.distance(self.player.pos);
            if dist > best_dist {
                best = p;
                best_dist = dist;
            }
            if dist >= a.drone_spawn_clearance {
                break;
            }
        }
        Drone {
            id,
            pos: best,
            heading: Vec2::from_angle(self.rng.range(0.0, std::f32::consts::TAU)),
            health: a.drone_health,
            alive: true,
            turn_timer: self.rng.range(a.drone_turn_min, a.drone_turn_max),
            respawn_timer: 0.0,
        }
    }
}
