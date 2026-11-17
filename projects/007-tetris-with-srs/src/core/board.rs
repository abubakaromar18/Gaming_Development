//! The playfield ("matrix").
//!
//! Ten columns wide and forty rows tall: rows 0..20 are visible, rows 20..40
//! are the hidden buffer zone pieces spawn into and can be kicked up through.

use super::piece::{Piece, PieceKind};

pub const WIDTH: i32 = 10;
pub const HEIGHT: i32 = 40;
pub const VISIBLE_HEIGHT: i32 = 20;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Board {
    /// Row-major, row 0 at the bottom.
    cells: Vec<Option<PieceKind>>,
}

impl Default for Board {
    fn default() -> Self {
        Board::new()
    }
}

impl Board {
    pub fn new() -> Board {
        Board {
            cells: vec![None; (WIDTH * HEIGHT) as usize],
        }
    }

    /// Builds a board from ASCII art, top row first. `#` (or any piece
    /// letter) is a filled cell, `.` is empty. Rows are bottom-aligned, so a
    /// three-line picture fills rows 2, 1, 0. Used by tests and puzzle setups.
    pub fn from_ascii(rows: &[&str]) -> Result<Board, String> {
        if rows.len() > HEIGHT as usize {
            return Err(format!("{} rows exceed matrix height {HEIGHT}", rows.len()));
        }
        let mut board = Board::new();
        for (i, row) in rows.iter().enumerate() {
            if row.chars().count() != WIDTH as usize {
                return Err(format!("row {i} is not {WIDTH} characters wide: {row:?}"));
            }
            let y = (rows.len() - 1 - i) as i32;
            for (x, ch) in row.chars().enumerate() {
                let cell = match ch {
                    '.' => None,
                    '#' => Some(PieceKind::O),
                    'I' => Some(PieceKind::I),
                    'O' => Some(PieceKind::O),
                    'T' => Some(PieceKind::T),
                    'S' => Some(PieceKind::S),
                    'Z' => Some(PieceKind::Z),
                    'J' => Some(PieceKind::J),
                    'L' => Some(PieceKind::L),
                    other => return Err(format!("unknown cell {other:?} in row {i}")),
                };
                board.set(x as i32, y, cell);
            }
        }
        Ok(board)
    }

    fn index(x: i32, y: i32) -> Option<usize> {
        if (0..WIDTH).contains(&x) && (0..HEIGHT).contains(&y) {
            Some((y * WIDTH + x) as usize)
        } else {
            None
        }
    }

    /// Contents of a cell; `None` both for empty and out-of-range cells.
    pub fn get(&self, x: i32, y: i32) -> Option<PieceKind> {
        Board::index(x, y).and_then(|i| self.cells[i])
    }

    /// True if the cell is filled or outside the walls/floor. The space
    /// above the buffer counts as open so kicks never fail on the ceiling.
    pub fn is_blocked(&self, x: i32, y: i32) -> bool {
        if !(0..WIDTH).contains(&x) || y < 0 {
            return true;
        }
        self.get(x, y).is_some()
    }

    fn set(&mut self, x: i32, y: i32, cell: Option<PieceKind>) {
        if let Some(i) = Board::index(x, y) {
            self.cells[i] = cell;
        }
    }

    /// Whether `piece` can occupy its current position.
    pub fn fits(&self, piece: &Piece) -> bool {
        piece
            .cells()
            .iter()
            .all(|&(x, y)| y < HEIGHT && !self.is_blocked(x, y))
    }

    /// Writes the piece into the matrix. The caller must have checked `fits`.
    pub fn lock(&mut self, piece: &Piece) {
        debug_assert!(self.fits(piece), "locking an overlapping piece");
        for (x, y) in piece.cells() {
            self.set(x, y, Some(piece.kind));
        }
    }

    fn row_full(&self, y: i32) -> bool {
        (0..WIDTH).all(|x| self.get(x, y).is_some())
    }

    /// Removes every full row, drops the rows above, and returns how many
    /// were cleared.
    pub fn clear_full_rows(&mut self) -> u32 {
        let mut write = 0;
        let mut cleared = 0;
        for read in 0..HEIGHT {
            if self.row_full(read) {
                cleared += 1;
                continue;
            }
            if write != read {
                for x in 0..WIDTH {
                    let cell = self.get(x, read);
                    self.set(x, write, cell);
                }
            }
            write += 1;
        }
        for y in write..HEIGHT {
            for x in 0..WIDTH {
                self.set(x, y, None);
            }
        }
        cleared
    }

    pub fn is_empty(&self) -> bool {
        self.cells.iter().all(Option::is_none)
    }

    /// Number of filled cells; handy for invariants in tests.
    pub fn filled_count(&self) -> usize {
        self.cells.iter().filter(|c| c.is_some()).count()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::core::piece::Rotation;

    #[test]
    fn walls_and_floor_block() {
        let b = Board::new();
        assert!(b.is_blocked(-1, 0));
        assert!(b.is_blocked(WIDTH, 0));
        assert!(b.is_blocked(0, -1));
        assert!(!b.is_blocked(0, 0));
    }

    #[test]
    fn clearing_drops_rows_above() {
        let mut b =
            Board::from_ascii(&["#.........", "##########", ".#........", "##########"]).unwrap();
        assert_eq!(b.clear_full_rows(), 2);
        assert!(b.get(1, 0).is_some());
        assert!(b.get(0, 1).is_some());
        assert_eq!(b.filled_count(), 2);
    }

    #[test]
    fn piece_cannot_fit_through_floor() {
        let b = Board::new();
        let p = Piece::new(PieceKind::T, Rotation::Zero, 0, -2);
        assert!(!b.fits(&p));
    }

    #[test]
    fn rejects_malformed_ascii() {
        assert!(Board::from_ascii(&["###"]).is_err());
        assert!(Board::from_ascii(&["#########?"]).is_err());
    }
}
