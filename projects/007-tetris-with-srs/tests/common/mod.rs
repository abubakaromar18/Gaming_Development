//! Shared helpers for the integration tests.
#![allow(dead_code)]

use tetris_srs::core::{Board, Config, FixedSequence, Game, Input, PieceKind};

pub const DT: f64 = 1.0 / 60.0;

pub fn fixed_game(pieces: &[PieceKind], board: Board) -> Game {
    let seq = FixedSequence::new(pieces.to_vec()).unwrap();
    Game::with_board(Config::default(), Box::new(seq), board).unwrap()
}

pub fn tap(game: &mut Game, input: Input) {
    game.update(DT, &input);
    // Release everything so the next tap is a fresh press.
    game.update(0.0, &Input::default());
}

pub fn hard_drop() -> Input {
    Input {
        hard_drop: true,
        ..Input::default()
    }
}

pub fn cw() -> Input {
    Input {
        rotate_cw: true,
        ..Input::default()
    }
}

pub fn ccw() -> Input {
    Input {
        rotate_ccw: true,
        ..Input::default()
    }
}

pub fn left() -> Input {
    Input {
        left: true,
        ..Input::default()
    }
}

pub fn right() -> Input {
    Input {
        right: true,
        ..Input::default()
    }
}
