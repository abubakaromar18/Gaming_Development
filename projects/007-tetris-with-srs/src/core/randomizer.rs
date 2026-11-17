//! Piece randomizers (Strategy pattern).
//!
//! Modern Tetris deals pieces from a shuffled "bag" of all seven, so the
//! longest possible wait for any piece is 12 and the longest run of S/Z
//! "snake" pieces is 4. That removes the droughts and floods of a uniformly
//! random generator while keeping the order unpredictable.

use super::piece::PieceKind;
use super::rng::RandomSource;

/// Anything that can deal an endless stream of pieces.
pub trait Randomizer {
    fn next_piece(&mut self) -> PieceKind;
}

/// The Guideline 7-bag: shuffle all seven, deal them, repeat.
pub struct SevenBag {
    rng: Box<dyn RandomSource>,
    bag: Vec<PieceKind>,
}

impl SevenBag {
    pub fn new(rng: Box<dyn RandomSource>) -> SevenBag {
        SevenBag {
            rng,
            bag: Vec::with_capacity(7),
        }
    }

    fn refill(&mut self) {
        self.bag.extend_from_slice(&PieceKind::ALL);
        // Fisher-Yates. Pieces are popped from the back, so the order of the
        // shuffled vector is the dealing order reversed; either is uniform.
        for i in (1..self.bag.len()).rev() {
            let j = self.rng.below(i as u32 + 1) as usize;
            self.bag.swap(i, j);
        }
    }
}

impl Randomizer for SevenBag {
    fn next_piece(&mut self) -> PieceKind {
        if self.bag.is_empty() {
            self.refill();
        }
        self.bag.pop().expect("bag was just refilled")
    }
}

/// Deals a fixed, looping sequence. Useful for tutorials, puzzle modes and
/// tests that need an exact piece order.
pub struct FixedSequence {
    pieces: Vec<PieceKind>,
    next: usize,
}

impl FixedSequence {
    pub fn new(pieces: Vec<PieceKind>) -> Result<FixedSequence, String> {
        if pieces.is_empty() {
            return Err("a fixed sequence needs at least one piece".into());
        }
        Ok(FixedSequence { pieces, next: 0 })
    }
}

impl Randomizer for FixedSequence {
    fn next_piece(&mut self) -> PieceKind {
        let p = self.pieces[self.next];
        self.next = (self.next + 1) % self.pieces.len();
        p
    }
}
