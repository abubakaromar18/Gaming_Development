//! Tetromino shapes and the four SRS rotation states.
//!
//! Coordinates are **y-up** with row 0 at the bottom of the matrix, which is
//! the convention the SRS kick tables are published in. A piece's `(x, y)` is
//! the bottom-left corner of its bounding box: 4x4 for I, 2x2 for O, 3x3 for
//! the rest. SRS rotates every piece about the centre of that box ("true
//! rotation"), so the other three states are derived from spawn state by a
//! quarter-turn of the box instead of being typed in by hand.

/// The seven one-sided tetrominoes.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub enum PieceKind {
    I,
    O,
    T,
    S,
    Z,
    J,
    L,
}

impl PieceKind {
    /// All kinds in a fixed order; the 7-bag shuffles a copy of this.
    pub const ALL: [PieceKind; 7] = [
        PieceKind::I,
        PieceKind::O,
        PieceKind::T,
        PieceKind::S,
        PieceKind::Z,
        PieceKind::J,
        PieceKind::L,
    ];

    /// Side length of the rotation bounding box.
    pub const fn box_size(self) -> i32 {
        match self {
            PieceKind::I => 4,
            PieceKind::O => 2,
            _ => 3,
        }
    }

    /// Cells of the spawn state (state 0), relative to the box's bottom-left.
    /// Every piece spawns flat side down, pointing up, as in the Guideline.
    const fn spawn_cells(self) -> [(i32, i32); 4] {
        match self {
            // ....
            // ####   row 2 of the 4x4 box
            PieceKind::I => [(0, 2), (1, 2), (2, 2), (3, 2)],
            PieceKind::O => [(0, 0), (1, 0), (0, 1), (1, 1)],
            // .#.
            // ###
            PieceKind::T => [(0, 1), (1, 1), (2, 1), (1, 2)],
            // .##
            // ##.
            PieceKind::S => [(0, 1), (1, 1), (1, 2), (2, 2)],
            // ##.
            // .##
            PieceKind::Z => [(1, 1), (2, 1), (0, 2), (1, 2)],
            // #..
            // ###
            PieceKind::J => [(0, 1), (1, 1), (2, 1), (0, 2)],
            // ..#
            // ###
            PieceKind::L => [(0, 1), (1, 1), (2, 1), (2, 2)],
        }
    }

    /// Cells for `rotation`, relative to the bounding box.
    pub fn cells(self, rotation: Rotation) -> [(i32, i32); 4] {
        let n = self.box_size();
        let mut cells = self.spawn_cells();
        for _ in 0..rotation.index() {
            for c in cells.iter_mut() {
                // Clockwise quarter-turn inside an n x n box, y-up.
                *c = (c.1, n - 1 - c.0);
            }
        }
        cells
    }
}

/// SRS rotation states, named the way the kick tables name them.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Rotation {
    /// Spawn state.
    Zero,
    /// One clockwise turn from spawn ("R").
    Right,
    /// Two turns ("2").
    Two,
    /// One counter-clockwise turn from spawn ("L").
    Left,
}

impl Rotation {
    pub const fn index(self) -> usize {
        match self {
            Rotation::Zero => 0,
            Rotation::Right => 1,
            Rotation::Two => 2,
            Rotation::Left => 3,
        }
    }

    pub const fn from_index(i: usize) -> Rotation {
        match i % 4 {
            0 => Rotation::Zero,
            1 => Rotation::Right,
            2 => Rotation::Two,
            _ => Rotation::Left,
        }
    }

    pub const fn cw(self) -> Rotation {
        Rotation::from_index(self.index() + 1)
    }

    pub const fn ccw(self) -> Rotation {
        Rotation::from_index(self.index() + 3)
    }
}

/// A tetromino placed on (or above) the matrix.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Piece {
    pub kind: PieceKind,
    pub rotation: Rotation,
    /// Bottom-left of the bounding box.
    pub x: i32,
    pub y: i32,
}

impl Piece {
    pub fn new(kind: PieceKind, rotation: Rotation, x: i32, y: i32) -> Piece {
        Piece {
            kind,
            rotation,
            x,
            y,
        }
    }

    /// Absolute matrix coordinates of the four minoes.
    pub fn cells(&self) -> [(i32, i32); 4] {
        let mut cells = self.kind.cells(self.rotation);
        for c in cells.iter_mut() {
            c.0 += self.x;
            c.1 += self.y;
        }
        cells
    }

    pub fn shifted(&self, dx: i32, dy: i32) -> Piece {
        Piece {
            x: self.x + dx,
            y: self.y + dy,
            ..*self
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn sorted(mut c: [(i32, i32); 4]) -> [(i32, i32); 4] {
        c.sort();
        c
    }

    #[test]
    fn four_turns_return_to_spawn() {
        for kind in PieceKind::ALL {
            let mut r = Rotation::Zero;
            for _ in 0..4 {
                r = r.cw();
            }
            assert_eq!(r, Rotation::Zero);
            assert_eq!(kind.cells(r), kind.cells(Rotation::Zero));
        }
    }

    #[test]
    fn t_right_state_points_right() {
        assert_eq!(
            sorted(PieceKind::T.cells(Rotation::Right)),
            sorted([(1, 0), (1, 1), (1, 2), (2, 1)])
        );
    }

    #[test]
    fn i_states_occupy_the_srs_columns_and_rows() {
        // R is column 2, 2 is row 1, L is column 1 of the 4x4 box.
        assert!(PieceKind::I.cells(Rotation::Right).iter().all(|c| c.0 == 2));
        assert!(PieceKind::I.cells(Rotation::Two).iter().all(|c| c.1 == 1));
        assert!(PieceKind::I.cells(Rotation::Left).iter().all(|c| c.0 == 1));
    }

    #[test]
    fn o_piece_is_rotation_invariant() {
        let spawn = sorted(PieceKind::O.cells(Rotation::Zero));
        for i in 0..4 {
            assert_eq!(sorted(PieceKind::O.cells(Rotation::from_index(i))), spawn);
        }
    }
}
