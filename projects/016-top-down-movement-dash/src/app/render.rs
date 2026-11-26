//! Drawing and purely cosmetic feedback. Reads the world, never changes it.
//!
//! Everything is drawn from primitives: circles, lines and text.

use std::collections::VecDeque;

use macroquad::prelude::*;
use topdown_dash::core::{AimSource, DashPhase, Event, Vec2 as V2, World};

/// Height of the HUD strip above the arena.
pub const HUD_HEIGHT: f32 = 64.0;

const BG: Color = Color::new(0.07, 0.08, 0.11, 1.0);
const FLOOR_GRID: Color = Color::new(1.0, 1.0, 1.0, 0.04);
const PLAYER: Color = Color::new(0.35, 0.85, 1.0, 1.0);
const PLAYER_GHOST: Color = Color::new(0.75, 0.95, 1.0, 1.0);
const DRONE: Color = Color::new(1.0, 0.42, 0.35, 1.0);
const TARGETED: Color = Color::new(1.0, 0.85, 0.3, 1.0);
const BULLET: Color = Color::new(1.0, 0.95, 0.6, 1.0);
const TEXT: Color = Color::new(0.85, 0.88, 0.95, 1.0);
const DIM: Color = Color::new(0.55, 0.6, 0.7, 1.0);

const AFTERIMAGE_SECONDS: f32 = 0.18;
const POPUP_SECONDS: f32 = 0.7;
const SPEED_SAMPLES: usize = 180;

struct Popup {
    text: &'static str,
    pos: V2,
    age: f32,
    color: Color,
}

/// Cosmetic state the simulation doesn't care about.
pub struct View {
    afterimages: Vec<(V2, f32)>,
    popups: Vec<Popup>,
    shake: f32,
    speeds: VecDeque<f32>,
}

fn to_screen(p: V2) -> (f32, f32) {
    (p.x, p.y + HUD_HEIGHT)
}

fn with_alpha(c: Color, a: f32) -> Color {
    Color::new(c.r, c.g, c.b, a.clamp(0.0, 1.0))
}

impl View {
    pub fn new() -> View {
        View {
            afterimages: Vec::new(),
            popups: Vec::new(),
            shake: 0.0,
            speeds: VecDeque::with_capacity(SPEED_SAMPLES),
        }
    }

    pub fn player_screen(world: &World) -> V2 {
        let (x, y) = to_screen(world.player().pos);
        V2::new(x, y)
    }

    /// Turns simulation events into feedback.
    pub fn react(&mut self, world: &World, events: &[Event]) {
        for e in events {
            match e {
                Event::Dodged { .. } => self.popups.push(Popup {
                    text: "DODGE",
                    pos: world.player().pos,
                    age: 0.0,
                    color: PLAYER_GHOST,
                }),
                Event::PlayerHit { .. } => self.shake = 0.25,
                Event::DroneDestroyed { drone } => {
                    if let Some(d) = world.drones().iter().find(|d| d.id == *drone) {
                        self.popups.push(Popup {
                            text: "+1",
                            pos: d.pos,
                            age: 0.0,
                            color: TARGETED,
                        });
                    }
                }
                _ => {}
            }
        }
    }

    /// Ages cosmetic effects and records a frame of history.
    pub fn tick(&mut self, world: &World, dt: f32) {
        for a in &mut self.afterimages {
            a.1 += dt;
        }
        self.afterimages.retain(|a| a.1 < AFTERIMAGE_SECONDS);
        if world.player().dash.is_active() {
            self.afterimages.push((world.player().pos, 0.0));
        }
        for p in &mut self.popups {
            p.age += dt;
        }
        self.popups.retain(|p| p.age < POPUP_SECONDS);
        self.shake = (self.shake - dt).max(0.0);

        if self.speeds.len() == SPEED_SAMPLES {
            self.speeds.pop_front();
        }
        self.speeds.push_back(world.player().vel.length());
    }

    pub fn draw(&self, world: &World, aim_source: AimSource) {
        clear_background(BG);
        let offset = if self.shake > 0.0 {
            let k = self.shake * 24.0;
            vec2(
                (world.time() as f32 * 91.0).sin() * k,
                (world.time() as f32 * 67.0).cos() * k,
            )
        } else {
            Vec2::ZERO
        };
        set_camera(&Camera2D {
            target: vec2(
                screen_width() / 2.0 - offset.x,
                screen_height() / 2.0 - offset.y,
            ),
            zoom: vec2(2.0 / screen_width(), 2.0 / screen_height()),
            ..Default::default()
        });

        self.draw_arena(world);
        self.draw_aim(world);
        self.draw_drones(world);
        self.draw_player(world);
        for b in world.bullets() {
            let (x, y) = to_screen(b.pos);
            draw_circle(x, y, world.config().combat.bullet_radius, BULLET);
        }
        for p in &self.popups {
            let (x, y) = to_screen(p.pos);
            let t = p.age / POPUP_SECONDS;
            draw_text(
                p.text,
                x - 20.0,
                y - 24.0 - 30.0 * t,
                22.0,
                with_alpha(p.color, 1.0 - t),
            );
        }

        set_default_camera();
        self.draw_hud(world, aim_source);
    }

    fn draw_arena(&self, world: &World) {
        let a = &world.config().arena;
        let mut x = 0.0;
        while x <= a.width {
            draw_line(x, HUD_HEIGHT, x, HUD_HEIGHT + a.height, 1.0, FLOOR_GRID);
            x += 40.0;
        }
        let mut y = 0.0;
        while y <= a.height {
            draw_line(
                0.0,
                HUD_HEIGHT + y,
                a.width,
                HUD_HEIGHT + y,
                1.0,
                FLOOR_GRID,
            );
            y += 40.0;
        }
        draw_rectangle_lines(0.0, HUD_HEIGHT, a.width, a.height, 2.0, DIM);
    }

    fn draw_aim(&self, world: &World) {
        let Some(aim) = world.aim() else { return };
        let cfg = world.config();
        let p = world.player().pos;
        let (px, py) = to_screen(p);
        // The assist cone around the corrected direction.
        if cfg.aim.enabled {
            for side in [-1.0, 1.0] {
                let edge = p + aim.dir.rotate(side * cfg.aim.cone_half_angle) * cfg.aim.range;
                let (ex, ey) = to_screen(edge);
                draw_line(px, py, ex, ey, 1.0, with_alpha(DIM, 0.25));
            }
        }
        let (ax, ay) = to_screen(p + aim.dir * 90.0);
        draw_line(px, py, ax, ay, 2.0, with_alpha(PLAYER, 0.8));
    }

    fn draw_drones(&self, world: &World) {
        let cfg = world.config();
        let target = world.aim().and_then(|a| a.target);
        for d in world.drones().iter().filter(|d| d.alive) {
            let (x, y) = to_screen(d.pos);
            let r = cfg.arena.drone_radius;
            draw_circle(x, y, r, DRONE);
            draw_circle(x, y, r * 0.45, BG);
            // Health pips.
            for i in 0..d.health {
                let angle = i as f32 * std::f32::consts::TAU / cfg.arena.drone_health as f32;
                draw_circle(
                    x + angle.cos() * r * 0.7,
                    y + angle.sin() * r * 0.7,
                    2.0,
                    BG,
                );
            }
            if target == Some(d.id) {
                draw_circle_lines(x, y, r + 6.0, 2.0, TARGETED);
            }
        }
    }

    fn draw_player(&self, world: &World) {
        let cfg = world.config();
        let p = world.player();
        let r = cfg.arena.player_radius;
        for (pos, age) in &self.afterimages {
            let (x, y) = to_screen(*pos);
            draw_circle(
                x,
                y,
                r,
                with_alpha(PLAYER_GHOST, 0.35 * (1.0 - age / AFTERIMAGE_SECONDS)),
            );
        }

        let (x, y) = to_screen(p.pos);
        let body = if p.dash.is_invulnerable(&cfg.dash) {
            with_alpha(PLAYER_GHOST, 0.55)
        } else if p.hurt_timer > 0.0 && (p.hurt_timer * 20.0) as i32 % 2 == 0 {
            with_alpha(PLAYER, 0.3)
        } else {
            PLAYER
        };
        draw_circle(x, y, r, body);
        let (fx, fy) = to_screen(p.pos + p.facing * (r + 6.0));
        draw_line(x, y, fx, fy, 3.0, BG);

        // Dash readiness ring.
        let readiness = p.dash.readiness(&cfg.dash);
        if readiness < 1.0 {
            let segments = 24;
            let lit = (segments as f32 * readiness) as i32;
            for i in 0..lit {
                let a = -std::f32::consts::FRAC_PI_2
                    + i as f32 * std::f32::consts::TAU / segments as f32;
                draw_circle(x + a.cos() * (r + 9.0), y + a.sin() * (r + 9.0), 1.6, DIM);
            }
        }
    }

    fn draw_hud(&self, world: &World, aim_source: AimSource) {
        let cfg = world.config();
        let p = world.player();
        let hearts: String = (0..cfg.combat.max_health)
            .map(|i| if i < p.health { '#' } else { '.' })
            .collect();
        let dash = match p.dash.phase() {
            DashPhase::Ready => "READY",
            DashPhase::Active { .. } => "DASH",
            DashPhase::Cooldown { .. } => "cooldown",
        };
        let source = match aim_source {
            AimSource::Stick => "keys (stick)",
            AimSource::Pointer => "mouse",
        };
        draw_text(
            format!("HP {hearts}   SCORE {}   DASH {dash}", world.score()),
            14.0,
            24.0,
            22.0,
            TEXT,
        );
        draw_text(
            format!(
                "[1] assist: {}   [2] curve: {}   aim: {source}",
                if cfg.aim.enabled { "on" } else { "off" },
                cfg.movement.curve.name()
            ),
            14.0,
            48.0,
            18.0,
            DIM,
        );
        draw_text(
            "WASD move  arrows/mouse aim  LMB fire  Space dash  R restart",
            500.0,
            22.0,
            15.0,
            DIM,
        );

        // Speed history: shows the acceleration curve and the dash spike.
        let (gx, gy, gw, gh) = (700.0, 32.0, 240.0, 26.0);
        draw_rectangle_lines(gx, gy, gw, gh, 1.0, with_alpha(DIM, 0.5));
        let top = cfg.dash.distance / cfg.dash.duration;
        let run_line = gy + gh - gh * cfg.movement.max_speed / top;
        draw_line(gx, run_line, gx + gw, run_line, 1.0, with_alpha(DIM, 0.4));
        let step = gw / SPEED_SAMPLES as f32;
        for (i, pair) in self
            .speeds
            .iter()
            .collect::<Vec<_>>()
            .windows(2)
            .enumerate()
        {
            let y0 = gy + gh - gh * (pair[0] / top).min(1.0);
            let y1 = gy + gh - gh * (pair[1] / top).min(1.0);
            draw_line(
                gx + i as f32 * step,
                y0,
                gx + (i + 1) as f32 * step,
                y1,
                1.5,
                PLAYER,
            );
        }

        if world.is_over() {
            let msg = "DOWN - press R";
            let size = measure_text(msg, None, 48, 1.0);
            draw_text(
                msg,
                (screen_width() - size.width) / 2.0,
                screen_height() / 2.0,
                48.0,
                DRONE,
            );
        }
    }
}
