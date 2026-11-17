//! Guideline scoring: line clears, T-spins, back-to-back and combos.

use super::tspin::TSpin;

/// What a single lock produced, as reported to the player.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ClearReport {
    pub lines: u32,
    pub tspin: TSpin,
    /// This clear continued a back-to-back chain (and got the 1.5x bonus).
    pub back_to_back: bool,
    /// Consecutive line-clearing locks minus one; 0 for the first clear.
    pub combo: u32,
    /// Total points awarded for the lock, combo included.
    pub points: u64,
}

/// Base points before the level multiplier.
pub fn base_points(lines: u32, tspin: TSpin) -> u64 {
    match (tspin, lines) {
        (TSpin::None, 0) => 0,
        (TSpin::None, 1) => 100,
        (TSpin::None, 2) => 300,
        (TSpin::None, 3) => 500,
        (TSpin::None, _) => 800,
        (TSpin::Mini, 0) => 100,
        (TSpin::Mini, 1) => 200,
        (TSpin::Mini, _) => 400,
        (TSpin::Full, 0) => 400,
        (TSpin::Full, 1) => 800,
        (TSpin::Full, 2) => 1200,
        (TSpin::Full, _) => 1600,
    }
}

/// "Difficult" clears keep a back-to-back chain alive: Tetrises and any
/// T-spin that clears lines. An ordinary single/double/triple breaks it;
/// locks that clear nothing leave it untouched.
fn is_difficult(lines: u32, tspin: TSpin) -> bool {
    lines >= 4 || (lines > 0 && tspin != TSpin::None)
}

#[derive(Debug, Clone, Default)]
pub struct ScoreKeeper {
    pub score: u64,
    pub lines: u32,
    b2b_active: bool,
    /// `None` until the first clear; reset to `None` by a non-clearing lock.
    combo: Option<u32>,
}

impl ScoreKeeper {
    pub fn new() -> ScoreKeeper {
        ScoreKeeper::default()
    }

    pub fn add_drop(&mut self, cells: u32, per_cell: u64) {
        self.score += cells as u64 * per_cell;
    }

    /// Records a lock and returns the report, or `None` if it scored nothing.
    pub fn record_lock(&mut self, lines: u32, tspin: TSpin, level: u32) -> Option<ClearReport> {
        let level = level.max(1) as u64;
        if lines == 0 {
            self.combo = None;
            let pts = base_points(0, tspin) * level;
            self.score += pts;
            return (pts > 0).then_some(ClearReport {
                lines,
                tspin,
                back_to_back: false,
                combo: 0,
                points: pts,
            });
        }

        let difficult = is_difficult(lines, tspin);
        let back_to_back = difficult && self.b2b_active;
        self.b2b_active = difficult;

        let combo = self.combo.map_or(0, |c| c + 1);
        self.combo = Some(combo);

        let mut pts = base_points(lines, tspin) * level;
        if back_to_back {
            pts = pts * 3 / 2;
        }
        pts += 50 * combo as u64 * level;

        self.score += pts;
        self.lines += lines;
        Some(ClearReport {
            lines,
            tspin,
            back_to_back,
            combo,
            points: pts,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tetris_then_tetris_is_back_to_back() {
        let mut s = ScoreKeeper::new();
        let a = s.record_lock(4, TSpin::None, 1).unwrap();
        assert!(!a.back_to_back);
        let b = s.record_lock(4, TSpin::None, 1).unwrap();
        assert!(b.back_to_back);
        assert_eq!(b.combo, 1);
        assert_eq!(b.points, 1200 + 50);
    }

    #[test]
    fn single_breaks_back_to_back_but_empty_lock_does_not() {
        let mut s = ScoreKeeper::new();
        s.record_lock(2, TSpin::Full, 1);
        s.record_lock(0, TSpin::None, 1);
        assert!(s.record_lock(4, TSpin::None, 1).unwrap().back_to_back);
        s.record_lock(1, TSpin::None, 1);
        assert!(!s.record_lock(4, TSpin::None, 1).unwrap().back_to_back);
    }

    #[test]
    fn combo_resets_on_non_clearing_lock() {
        let mut s = ScoreKeeper::new();
        s.record_lock(1, TSpin::None, 1);
        assert_eq!(s.record_lock(1, TSpin::None, 1).unwrap().combo, 1);
        assert_eq!(s.record_lock(1, TSpin::None, 1).unwrap().combo, 2);
        assert!(s.record_lock(0, TSpin::None, 1).is_none());
        assert_eq!(s.record_lock(1, TSpin::None, 1).unwrap().combo, 0);
    }

    #[test]
    fn level_multiplies_points() {
        let mut s = ScoreKeeper::new();
        assert_eq!(s.record_lock(3, TSpin::None, 5).unwrap().points, 2500);
        assert_eq!(s.lines, 3);
    }
}
