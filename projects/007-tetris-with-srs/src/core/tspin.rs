//! T-spin detection with the Guideline three-corner rule.
//!
//! A lock counts as a T-spin when the piece is a T, its last successful
//! action was a rotation, and at least three of the four cells diagonal to
//! its centre are blocked (walls and floor count). It is a *full* T-spin if
//! both corners on the side the T points toward are blocked, otherwise a
//! *mini*, unless the rotation used the fifth kick test, which the Guideline
//! always upgrades to full (this is what makes T-spin triples possible).

use super::board::Board;
use super::piece::{Piece, PieceKind, Rotation};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TSpin {
    None,
    Mini,
    Full,
}

/// Kick index of the fifth SRS test.
const UPGRADE_KICK: usize = 4;

/// Classifies a lock. `last_rotation_kick` is `Some(kick index)` if the
/// piece's last successful action was a rotation, `None` otherwise.
pub fn detect(board: &Board, piece: &Piece, last_rotation_kick: Option<usize>) -> TSpin {
    if piece.kind != PieceKind::T {
        return TSpin::None;
    }
    let Some(kick) = last_rotation_kick else {
        return TSpin::None;
    };

    // Centre of the T is (1, 1) in its 3x3 box.
    let (cx, cy) = (piece.x + 1, piece.y + 1);
    let blocked = |dx: i32, dy: i32| board.is_blocked(cx + dx, cy + dy);

    let corners = [(-1, 1), (1, 1), (1, -1), (-1, -1)];
    let filled = corners.iter().filter(|&&(dx, dy)| blocked(dx, dy)).count();
    if filled < 3 {
        return TSpin::None;
    }

    let front = match piece.rotation {
        Rotation::Zero => [(-1, 1), (1, 1)],
        Rotation::Right => [(1, 1), (1, -1)],
        Rotation::Two => [(1, -1), (-1, -1)],
        Rotation::Left => [(-1, -1), (-1, 1)],
    };
    let front_filled = front.iter().all(|&(dx, dy)| blocked(dx, dy));

    if front_filled || kick == UPGRADE_KICK {
        TSpin::Full
    } else {
        TSpin::Mini
    }
}
