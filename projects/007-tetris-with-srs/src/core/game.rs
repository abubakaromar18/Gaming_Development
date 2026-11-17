//! The game loop's rules: spawning, movement, gravity, lock delay, hold,
//! line clears, scoring and top-out.
//!
//! `Game::update(dt, &input)` is the only way time passes. It never reads a
//! clock or a global RNG, so the same input stream always produces the same
//! game, which is what the determinism tests rely on.

use std::collections::VecDeque;

use super::autorepeat::AutoRepeat;
use super::board::{Board, VISIBLE_HEIGHT, WIDTH};
use super::config::{seconds_per_row, Config, ConfigError};
use super::piece::{Piece, PieceKind, Rotation};
use super::randomizer::Randomizer;
use super::scoring::{ClearReport, ScoreKeeper};
use super::srs::{try_rotate, Spin};
use super::tspin;

/// One frame of player intent. Directions and soft drop are *held* states;
/// the rest are *pressed this frame* edges, so the shell decides what a
/// press is and the core never sees raw key repeat.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Input {
    pub left: bool,
    pub right: bool,
    pub soft_drop: bool,
    pub hard_drop: bool,
    pub rotate_cw: bool,
    pub rotate_ccw: bool,
    pub hold: bool,
}

/// Why the game ended.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TopOut {
    /// A new piece spawned overlapping the stack.
    BlockOut,
    /// A piece locked entirely above the visible matrix.
    LockOut,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Status {
    Playing,
    GameOver(TopOut),
}

/// Things that happened during an update, for the presentation layer
/// (sounds, popups, screen shake) to react to.
#[derive(Debug, Clone, PartialEq)]
pub enum Event {
    Locked(Piece),
    Scored(ClearReport),
    Held(PieceKind),
    LevelUp(u32),
    GameOver(TopOut),
}

const HARD_DROP_POINTS_PER_CELL: u64 = 2;
const SOFT_DROP_POINTS_PER_CELL: u64 = 1;

pub struct Game {
    config: Config,
    board: Board,
    randomizer: Box<dyn Randomizer>,
    queue: VecDeque<PieceKind>,
    active: Piece,
    hold: Option<PieceKind>,
    hold_used: bool,
    shift: AutoRepeat,
    /// Fractional rows of gravity banked toward the next one-row fall.
    fall_accum: f64,
    lock_timer: f64,
    lock_resets: u32,
    /// Lowest row the active piece has reached; reaching a new low refills
    /// its lock resets so pieces can't stall forever but aren't punished
    /// for working their way down.
    lowest_y: i32,
    /// `Some(kick index)` while the piece's last successful action was a
    /// rotation; any translation clears it. Feeds T-spin detection.
    last_rotation_kick: Option<usize>,
    scores: ScoreKeeper,
    status: Status,
    events: Vec<Event>,
    pieces_locked: u64,
}

impl Game {
    /// Starts a game on an empty matrix.
    pub fn new(config: Config, randomizer: Box<dyn Randomizer>) -> Result<Game, ConfigError> {
        Game::with_board(config, randomizer, Board::new())
    }

    /// Starts a game on a prepared matrix (puzzles, garbage drills, tests).
    pub fn with_board(
        config: Config,
        randomizer: Box<dyn Randomizer>,
        board: Board,
    ) -> Result<Game, ConfigError> {
        config.validate()?;
        let shift = AutoRepeat::new(config.das, config.arr);
        let placeholder = Piece::new(PieceKind::O, Rotation::Zero, 0, 0);
        let mut game = Game {
            config,
            board,
            randomizer,
            queue: VecDeque::new(),
            active: placeholder,
            hold: None,
            hold_used: false,
            shift,
            fall_accum: 0.0,
            lock_timer: 0.0,
            lock_resets: 0,
            lowest_y: 0,
            last_rotation_kick: None,
            scores: ScoreKeeper::new(),
            status: Status::Playing,
            events: Vec::new(),
            pieces_locked: 0,
        };
        game.fill_queue();
        let first = game.take_next();
        game.spawn(first);
        Ok(game)
    }

    // ----- queries used by the renderer and tests -------------------------

    pub fn board(&self) -> &Board {
        &self.board
    }

    pub fn active(&self) -> Option<&Piece> {
        matches!(self.status, Status::Playing).then_some(&self.active)
    }

    /// Where the active piece would land if hard-dropped now.
    pub fn ghost(&self) -> Option<Piece> {
        self.active().map(|p| p.shifted(0, -self.drop_distance()))
    }

    pub fn held(&self) -> Option<PieceKind> {
        self.hold
    }

    pub fn can_hold(&self) -> bool {
        !self.hold_used
    }

    pub fn preview(&self) -> impl Iterator<Item = PieceKind> + '_ {
        self.queue.iter().copied().take(self.config.preview_count)
    }

    pub fn score(&self) -> u64 {
        self.scores.score
    }

    pub fn lines(&self) -> u32 {
        self.scores.lines
    }

    pub fn level(&self) -> u32 {
        self.config.start_level + self.scores.lines / self.config.lines_per_level
    }

    pub fn status(&self) -> Status {
        self.status
    }

    pub fn pieces_locked(&self) -> u64 {
        self.pieces_locked
    }

    /// 0..1 fraction of lock delay used, for a "about to lock" indicator.
    pub fn lock_progress(&self) -> f64 {
        if self.config.lock_delay == 0.0 {
            return if self.is_grounded() { 1.0 } else { 0.0 };
        }
        (self.lock_timer / self.config.lock_delay).min(1.0)
    }

    /// Returns and clears the events produced since the last call.
    pub fn drain_events(&mut self) -> Vec<Event> {
        std::mem::take(&mut self.events)
    }

    // ----- the update step -----------------------------------------------

    /// Advances the game by `dt` seconds with this frame's input.
    pub fn update(&mut self, dt: f64, input: &Input) {
        if self.status != Status::Playing {
            return;
        }
        let dt = dt.max(0.0);

        if input.hold && self.try_hold() {
            return;
        }
        if input.rotate_cw {
            self.rotate(Spin::Cw);
        }
        if input.rotate_ccw {
            self.rotate(Spin::Ccw);
        }

        let columns = self.shift.update(input.left, input.right, dt);
        let dir = columns.signum();
        for _ in 0..columns.abs() {
            if !self.translate(dir, 0) {
                break;
            }
        }

        if input.hard_drop {
            let distance = self.drop_distance();
            if distance > 0 {
                self.active = self.active.shifted(0, -distance);
                self.last_rotation_kick = None;
            }
            self.scores
                .add_drop(distance as u32, HARD_DROP_POINTS_PER_CELL);
            self.lock_active();
            return;
        }

        self.apply_gravity(dt, input.soft_drop);

        if self.is_grounded() {
            self.lock_timer += dt;
            if self.lock_timer >= self.config.lock_delay {
                self.lock_active();
            }
        }
    }

    fn apply_gravity(&mut self, dt: f64, soft_drop: bool) {
        let mut rows_per_second = 1.0 / seconds_per_row(self.level());
        if soft_drop {
            rows_per_second *= self.config.soft_drop_factor;
        }
        self.fall_accum += dt * rows_per_second;
        while self.fall_accum >= 1.0 {
            self.fall_accum -= 1.0;
            if !self.translate(0, -1) {
                // Resting on the stack: don't bank gravity, or the piece
                // would teleport down the moment it slides off a ledge.
                self.fall_accum = 0.0;
                break;
            }
            if soft_drop {
                self.scores.add_drop(1, SOFT_DROP_POINTS_PER_CELL);
            }
        }
    }

    // ----- movement ------------------------------------------------------

    fn is_grounded(&self) -> bool {
        !self.board.fits(&self.active.shifted(0, -1))
    }

    fn drop_distance(&self) -> i32 {
        let mut d = 0;
        while self.board.fits(&self.active.shifted(0, -(d + 1))) {
            d += 1;
        }
        d
    }

    fn translate(&mut self, dx: i32, dy: i32) -> bool {
        let moved = self.active.shifted(dx, dy);
        if !self.board.fits(&moved) {
            return false;
        }
        let was_grounded = self.is_grounded();
        self.active = moved;
        self.last_rotation_kick = None;
        self.after_successful_action(was_grounded);
        true
    }

    fn rotate(&mut self, spin: Spin) {
        let Some(rotated) = try_rotate(&self.board, &self.active, spin) else {
            return;
        };
        let was_grounded = self.is_grounded();
        self.active = rotated.piece;
        self.last_rotation_kick = Some(rotated.kick_index);
        self.after_successful_action(was_grounded);
    }

    /// Lock-delay bookkeeping shared by moves and rotations.
    fn after_successful_action(&mut self, was_grounded: bool) {
        if self.active.y < self.lowest_y {
            self.lowest_y = self.active.y;
            self.lock_resets = 0;
            self.lock_timer = 0.0;
        } else if was_grounded && self.lock_resets < self.config.max_lock_resets {
            self.lock_resets += 1;
            self.lock_timer = 0.0;
        }
    }

    // ----- piece lifecycle ----------------------------------------------

    fn fill_queue(&mut self) {
        let want = self.config.preview_count.max(1);
        while self.queue.len() < want {
            self.queue.push_back(self.randomizer.next_piece());
        }
    }

    fn take_next(&mut self) -> PieceKind {
        let next = self
            .queue
            .pop_front()
            .unwrap_or_else(|| self.randomizer.next_piece());
        self.fill_queue();
        next
    }

    /// Guideline spawn: centred (rounding left for 3-wide pieces), with the
    /// piece's lowest row just above the visible matrix, then dropped one
    /// row at once if there is room.
    fn spawn(&mut self, kind: PieceKind) {
        let min_y = kind
            .cells(Rotation::Zero)
            .iter()
            .map(|c| c.1)
            .min()
            .unwrap_or(0);
        let x = (WIDTH - kind.box_size()) / 2;
        let piece = Piece::new(kind, Rotation::Zero, x, VISIBLE_HEIGHT - min_y);
        if !self.board.fits(&piece) {
            self.active = piece;
            self.end(TopOut::BlockOut);
            return;
        }
        self.active = piece;
        let lowered = piece.shifted(0, -1);
        if self.board.fits(&lowered) {
            self.active = lowered;
        }
        self.lowest_y = self.active.y;
        self.lock_timer = 0.0;
        self.lock_resets = 0;
        self.fall_accum = 0.0;
        self.last_rotation_kick = None;
    }

    /// Hold swaps the active piece with the held one (or the next piece the
    /// first time). Allowed once per piece so it can't be used to stall.
    fn try_hold(&mut self) -> bool {
        if self.hold_used {
            return false;
        }
        self.hold_used = true;
        let current = self.active.kind;
        let next = match self.hold.replace(current) {
            Some(kind) => kind,
            None => self.take_next(),
        };
        self.events.push(Event::Held(current));
        self.spawn(next);
        true
    }

    fn lock_active(&mut self) {
        let piece = self.active;
        let kind = tspin::detect(&self.board, &piece, self.last_rotation_kick);
        let above_matrix = piece.cells().iter().all(|c| c.1 >= VISIBLE_HEIGHT);

        self.board.lock(&piece);
        self.pieces_locked += 1;
        self.events.push(Event::Locked(piece));

        let level_before = self.level();
        let lines = self.board.clear_full_rows();
        if let Some(report) = self.scores.record_lock(lines, kind, level_before) {
            self.events.push(Event::Scored(report));
        }
        if self.level() > level_before {
            self.events.push(Event::LevelUp(self.level()));
        }

        if above_matrix {
            self.end(TopOut::LockOut);
            return;
        }
        self.hold_used = false;
        let next = self.take_next();
        self.spawn(next);
    }

    fn end(&mut self, reason: TopOut) {
        self.status = Status::GameOver(reason);
        self.events.push(Event::GameOver(reason));
    }
}
