//! Draws the game with procedural shapes only: no textures or fonts beyond
//! macroquad's built-in one. Reads `Game` state and never mutates it.

use macroquad::prelude::*;
use tetris_srs::core::board::{VISIBLE_HEIGHT, WIDTH};
use tetris_srs::core::{ClearReport, Game, Piece, PieceKind, Rotation, Status, TSpin, TopOut};

const CELL: f32 = 32.0;
const BOARD_X: f32 = 200.0;
const BOARD_Y: f32 = 48.0;
const SIDE_CELL: f32 = 20.0;

const BACKGROUND: Color = Color::new(0.06, 0.07, 0.10, 1.0);
const WELL: Color = Color::new(0.10, 0.11, 0.15, 1.0);
const GRID: Color = Color::new(0.16, 0.17, 0.22, 1.0);
const TEXT: Color = Color::new(0.88, 0.90, 0.95, 1.0);
const DIM: Color = Color::new(0.55, 0.58, 0.66, 1.0);

/// Guideline colours.
fn piece_color(kind: PieceKind) -> Color {
    match kind {
        PieceKind::I => Color::from_rgba(0, 210, 230, 255),
        PieceKind::O => Color::from_rgba(240, 210, 0, 255),
        PieceKind::T => Color::from_rgba(170, 60, 220, 255),
        PieceKind::S => Color::from_rgba(60, 200, 80, 255),
        PieceKind::Z => Color::from_rgba(230, 60, 60, 255),
        PieceKind::J => Color::from_rgba(50, 100, 230, 255),
        PieceKind::L => Color::from_rgba(240, 140, 20, 255),
    }
}

fn shade(c: Color, k: f32) -> Color {
    Color::new(c.r * k, c.g * k, c.b * k, c.a)
}

/// A bevelled block: base fill, lighter top-left edge, darker bottom-right.
fn draw_block(x: f32, y: f32, size: f32, color: Color) {
    let bevel = (size * 0.12).max(2.0);
    draw_rectangle(x, y, size, size, shade(color, 0.75));
    draw_rectangle(x, y, size - bevel, size - bevel, shade(color, 1.15));
    draw_rectangle(
        x + bevel,
        y + bevel,
        size - 2.0 * bevel,
        size - 2.0 * bevel,
        color,
    );
}

fn cell_origin(x: i32, y: i32) -> (f32, f32) {
    (
        BOARD_X + x as f32 * CELL,
        BOARD_Y + (VISIBLE_HEIGHT - 1 - y) as f32 * CELL,
    )
}

fn draw_piece_on_board(piece: &Piece, color: Color) {
    for (x, y) in piece.cells() {
        if y < VISIBLE_HEIGHT {
            let (sx, sy) = cell_origin(x, y);
            draw_block(sx, sy, CELL, color);
        }
    }
}

fn draw_ghost(piece: &Piece) {
    let color = piece_color(piece.kind);
    for (x, y) in piece.cells() {
        if y < VISIBLE_HEIGHT {
            let (sx, sy) = cell_origin(x, y);
            draw_rectangle_lines(
                sx + 2.0,
                sy + 2.0,
                CELL - 4.0,
                CELL - 4.0,
                2.0,
                shade(color, 0.8),
            );
        }
    }
}

/// Draws a piece centred in a side-panel box at (cx, cy).
fn draw_mini_piece(kind: PieceKind, cx: f32, cy: f32, color: Color) {
    let cells = kind.cells(Rotation::Zero);
    let (min_x, max_x) = cells.iter().fold((i32::MAX, i32::MIN), |(lo, hi), c| {
        (lo.min(c.0), hi.max(c.0))
    });
    let (min_y, max_y) = cells.iter().fold((i32::MAX, i32::MIN), |(lo, hi), c| {
        (lo.min(c.1), hi.max(c.1))
    });
    let w = (max_x - min_x + 1) as f32 * SIDE_CELL;
    let h = (max_y - min_y + 1) as f32 * SIDE_CELL;
    for (x, y) in cells {
        let sx = cx - w / 2.0 + (x - min_x) as f32 * SIDE_CELL;
        let sy = cy - h / 2.0 + (max_y - y) as f32 * SIDE_CELL;
        draw_block(sx, sy, SIDE_CELL, color);
    }
}

fn label(text: &str, x: f32, y: f32, size: f32, color: Color) {
    draw_text(text, x, y, size, color);
}

fn centred(text: &str, cx: f32, y: f32, size: f32, color: Color) {
    let dims = measure_text(text, None, size as u16, 1.0);
    draw_text(text, cx - dims.width / 2.0, y, size, color);
}

/// Player-facing name of a clear, e.g. "B2B T-SPIN DOUBLE".
pub fn describe(report: &ClearReport) -> String {
    let lines = match report.lines {
        0 => "",
        1 => "SINGLE",
        2 => "DOUBLE",
        3 => "TRIPLE",
        _ => "TETRIS",
    };
    let spin = match report.tspin {
        TSpin::None => "",
        TSpin::Mini => "T-SPIN MINI",
        TSpin::Full => "T-SPIN",
    };
    let mut parts: Vec<String> = Vec::new();
    if report.back_to_back {
        parts.push("B2B".into());
    }
    if !spin.is_empty() {
        parts.push(spin.into());
    }
    if !lines.is_empty() {
        parts.push(lines.into());
    }
    let mut text = parts.join(" ");
    if report.combo > 0 {
        text.push_str(&format!("  COMBO x{}", report.combo));
    }
    text
}

/// A short-lived message shown under the board after a scoring lock.
pub struct Popup {
    pub text: String,
    pub points: u64,
    pub remaining: f32,
}

pub struct View<'a> {
    pub game: &'a Game,
    pub popup: Option<&'a Popup>,
    pub paused: bool,
    pub seed: u64,
}

pub fn draw(view: &View) {
    let game = view.game;
    clear_background(BACKGROUND);

    let board_w = WIDTH as f32 * CELL;
    let board_h = VISIBLE_HEIGHT as f32 * CELL;
    draw_rectangle(BOARD_X, BOARD_Y, board_w, board_h, WELL);
    for x in 0..=WIDTH {
        let sx = BOARD_X + x as f32 * CELL;
        draw_line(sx, BOARD_Y, sx, BOARD_Y + board_h, 1.0, GRID);
    }
    for y in 0..=VISIBLE_HEIGHT {
        let sy = BOARD_Y + y as f32 * CELL;
        draw_line(BOARD_X, sy, BOARD_X + board_w, sy, 1.0, GRID);
    }

    let board = game.board();
    for y in 0..VISIBLE_HEIGHT {
        for x in 0..WIDTH {
            if let Some(kind) = board.get(x, y) {
                let (sx, sy) = cell_origin(x, y);
                draw_block(sx, sy, CELL, shade(piece_color(kind), 0.85));
            }
        }
    }

    if let (Some(ghost), Some(active)) = (game.ghost(), game.active()) {
        draw_ghost(&ghost);
        // Darken the piece as its lock delay runs out.
        let k = 1.0 - 0.45 * game.lock_progress() as f32;
        draw_piece_on_board(active, shade(piece_color(active.kind), k));
    }
    draw_rectangle_lines(
        BOARD_X - 3.0,
        BOARD_Y - 3.0,
        board_w + 6.0,
        board_h + 6.0,
        3.0,
        DIM,
    );

    // Hold panel.
    let left_cx = BOARD_X / 2.0;
    label("HOLD", left_cx - 28.0, BOARD_Y + 18.0, 28.0, TEXT);
    draw_rectangle_lines(left_cx - 60.0, BOARD_Y + 28.0, 120.0, 80.0, 2.0, GRID);
    if let Some(kind) = game.held() {
        let color = if game.can_hold() {
            piece_color(kind)
        } else {
            shade(piece_color(kind), 0.4)
        };
        draw_mini_piece(kind, left_cx, BOARD_Y + 68.0, color);
    }

    // Stats.
    let stats = [
        ("SCORE", game.score().to_string()),
        ("LEVEL", game.level().to_string()),
        ("LINES", game.lines().to_string()),
    ];
    for (i, (name, value)) in stats.iter().enumerate() {
        let y = BOARD_Y + 180.0 + i as f32 * 70.0;
        label(name, 24.0, y, 22.0, DIM);
        label(value, 24.0, y + 30.0, 32.0, TEXT);
    }
    label(
        &format!("seed {}", view.seed),
        24.0,
        BOARD_Y + board_h - 4.0,
        18.0,
        DIM,
    );

    // Next queue.
    let right_x = BOARD_X + board_w;
    let right_cx = right_x + 100.0;
    label("NEXT", right_cx - 28.0, BOARD_Y + 18.0, 28.0, TEXT);
    for (i, kind) in game.preview().enumerate() {
        draw_mini_piece(
            kind,
            right_cx,
            BOARD_Y + 68.0 + i as f32 * 76.0,
            piece_color(kind),
        );
    }

    // Controls help.
    let help = [
        "<- -> move",
        "Down soft drop",
        "Space hard drop",
        "Up/X rotate cw",
        "Z/Ctrl rotate ccw",
        "C/Shift hold",
        "P pause  R restart",
    ];
    for (i, line) in help.iter().enumerate() {
        label(
            line,
            right_x + 20.0,
            BOARD_Y + 470.0 + i as f32 * 22.0,
            18.0,
            DIM,
        );
    }

    let mid_x = BOARD_X + board_w / 2.0;
    if let Some(p) = view.popup {
        let alpha = p.remaining.clamp(0.0, 1.0);
        let gold = Color::new(1.0, 0.85, 0.35, alpha);
        centred(&p.text, mid_x, BOARD_Y + board_h + 32.0, 28.0, gold);
        centred(
            &format!("+{}", p.points),
            mid_x,
            BOARD_Y + board_h + 56.0,
            22.0,
            Color::new(TEXT.r, TEXT.g, TEXT.b, alpha),
        );
    }

    let overlay = |title: &str, subtitle: &str| {
        draw_rectangle(
            BOARD_X,
            BOARD_Y,
            board_w,
            board_h,
            Color::new(0.0, 0.0, 0.0, 0.65),
        );
        centred(title, mid_x, BOARD_Y + board_h / 2.0 - 10.0, 44.0, TEXT);
        centred(subtitle, mid_x, BOARD_Y + board_h / 2.0 + 26.0, 22.0, DIM);
    };
    match game.status() {
        Status::GameOver(reason) => {
            let why = match reason {
                TopOut::BlockOut => "block out",
                TopOut::LockOut => "lock out",
            };
            overlay("GAME OVER", &format!("{why} - press R"));
        }
        Status::Playing if view.paused => overlay("PAUSED", "press P to resume"),
        Status::Playing => {}
    }
}
