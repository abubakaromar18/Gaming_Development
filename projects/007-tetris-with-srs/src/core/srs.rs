//! Super Rotation System wall kicks.
//!
//! When a rotation's basic position collides, SRS tries up to four
//! alternative offsets ("kicks") in a fixed order and takes the first that
//! fits. The tables below are the official Guideline data in y-up
//! coordinates. J, L, S, T and Z share one table; I has its own because its
//! 4x4 box rotates about a point between cells; O never needs to kick.

use super::board::Board;
use super::piece::{Piece, PieceKind, Rotation};

/// Rotation direction requested by the player.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Spin {
    Cw,
    Ccw,
}

type KickRow = [(i32, i32); 5];

// Indexed by [from_state][0 = clockwise, 1 = counter-clockwise].
const JLSTZ_KICKS: [[KickRow; 2]; 4] = [
    // 0 -> R, 0 -> L
    [
        [(0, 0), (-1, 0), (-1, 1), (0, -2), (-1, -2)],
        [(0, 0), (1, 0), (1, 1), (0, -2), (1, -2)],
    ],
    // R -> 2, R -> 0
    [
        [(0, 0), (1, 0), (1, -1), (0, 2), (1, 2)],
        [(0, 0), (1, 0), (1, -1), (0, 2), (1, 2)],
    ],
    // 2 -> L, 2 -> R
    [
        [(0, 0), (1, 0), (1, 1), (0, -2), (1, -2)],
        [(0, 0), (-1, 0), (-1, 1), (0, -2), (-1, -2)],
    ],
    // L -> 0, L -> 2
    [
        [(0, 0), (-1, 0), (-1, -1), (0, 2), (-1, 2)],
        [(0, 0), (-1, 0), (-1, -1), (0, 2), (-1, 2)],
    ],
];

const I_KICKS: [[KickRow; 2]; 4] = [
    // 0 -> R, 0 -> L
    [
        [(0, 0), (-2, 0), (1, 0), (-2, -1), (1, 2)],
        [(0, 0), (-1, 0), (2, 0), (-1, 2), (2, -1)],
    ],
    // R -> 2, R -> 0
    [
        [(0, 0), (-1, 0), (2, 0), (-1, 2), (2, -1)],
        [(0, 0), (2, 0), (-1, 0), (2, 1), (-1, -2)],
    ],
    // 2 -> L, 2 -> R
    [
        [(0, 0), (2, 0), (-1, 0), (2, 1), (-1, -2)],
        [(0, 0), (1, 0), (-2, 0), (1, -2), (-2, 1)],
    ],
    // L -> 0, L -> 2
    [
        [(0, 0), (1, 0), (-2, 0), (1, -2), (-2, 1)],
        [(0, 0), (-2, 0), (1, 0), (-2, -1), (1, 2)],
    ],
];

const O_KICKS: KickRow = [(0, 0); 5];

/// The ordered kick offsets SRS tries for a rotation out of `from`.
pub fn kicks(kind: PieceKind, from: Rotation, spin: Spin) -> &'static KickRow {
    let dir = match spin {
        Spin::Cw => 0,
        Spin::Ccw => 1,
    };
    match kind {
        PieceKind::O => &O_KICKS,
        PieceKind::I => &I_KICKS[from.index()][dir],
        _ => &JLSTZ_KICKS[from.index()][dir],
    }
}

/// A successful rotation and which kick test produced it (0 = no kick).
/// T-spin detection needs the index: kick 4 upgrades a mini to a full T-spin.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Rotated {
    pub piece: Piece,
    pub kick_index: usize,
}

/// Attempts an SRS rotation. Returns `None` if all five tests collide.
pub fn try_rotate(board: &Board, piece: &Piece, spin: Spin) -> Option<Rotated> {
    let to = match spin {
        Spin::Cw => piece.rotation.cw(),
        Spin::Ccw => piece.rotation.ccw(),
    };
    let turned = Piece {
        rotation: to,
        ..*piece
    };
    kicks(piece.kind, piece.rotation, spin)
        .iter()
        .enumerate()
        .map(|(i, &(dx, dy))| Rotated {
            piece: turned.shifted(dx, dy),
            kick_index: i,
        })
        .find(|r| board.fits(&r.piece))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn opposite_rotations_have_negated_kicks() {
        // Rotating A -> B and then B -> A must undo each kick, so the two
        // table rows are exact negations. This catches transcription errors.
        for kind in [PieceKind::T, PieceKind::I] {
            for i in 0..4 {
                let from = Rotation::from_index(i);
                let there = kicks(kind, from, Spin::Cw);
                let back = kicks(kind, from.cw(), Spin::Ccw);
                for (a, b) in there.iter().zip(back.iter()) {
                    assert_eq!((a.0, a.1), (-b.0, -b.1), "{kind:?} from {from:?}");
                }
            }
        }
    }

    #[test]
    fn free_rotation_uses_no_kick() {
        let board = Board::new();
        let t = Piece::new(PieceKind::T, Rotation::Zero, 4, 5);
        let r = try_rotate(&board, &t, Spin::Cw).unwrap();
        assert_eq!(r.kick_index, 0);
        assert_eq!((r.piece.x, r.piece.y), (4, 5));
        assert_eq!(r.piece.rotation, Rotation::Right);
    }

    #[test]
    fn i_piece_kicks_off_the_right_wall() {
        // Vertical I hugging the right wall (state L, column 1 of its box at
        // x = 9). Rotating to state 0 needs space to the left: kick 2 (-2, 0).
        let board = Board::new();
        let i = Piece::new(PieceKind::I, Rotation::Left, 8, 4);
        assert!(i.cells().iter().all(|c| c.0 == 9));
        let r = try_rotate(&board, &i, Spin::Cw).unwrap();
        assert_eq!(r.kick_index, 2);
        assert!(board.fits(&r.piece));
        assert!(r.piece.cells().iter().all(|c| c.0 <= 9 && c.0 >= 6));
    }

    #[test]
    fn fully_boxed_piece_cannot_rotate() {
        let board = Board::from_ascii(&["###.######", "##...#####", "##########"]).unwrap();
        let t = Piece::new(PieceKind::T, Rotation::Zero, 2, 0);
        assert!(board.fits(&t));
        assert!(try_rotate(&board, &t, Spin::Cw).is_none());
    }
}
