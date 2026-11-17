//! Delayed Auto Shift (DAS) and Auto Repeat Rate (ARR).
//!
//! Tapping left moves one column. Holding it moves one column at once, waits
//! `das` seconds, then repeats every `arr` seconds (or slides straight to the
//! wall when `arr` is 0). If both directions are held, the one pressed most
//! recently wins, so players can "reverse" without releasing the first key.

use super::board::WIDTH;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Dir {
    Left,
    Right,
}

impl Dir {
    fn sign(self) -> i32 {
        match self {
            Dir::Left => -1,
            Dir::Right => 1,
        }
    }
}

#[derive(Debug, Clone)]
pub struct AutoRepeat {
    das: f64,
    arr: f64,
    held_left: bool,
    held_right: bool,
    active: Option<Dir>,
    /// Time the active direction has been held.
    held_for: f64,
    /// Time banked toward the next repeat once DAS has charged.
    repeat_accum: f64,
}

impl AutoRepeat {
    pub fn new(das: f64, arr: f64) -> AutoRepeat {
        AutoRepeat {
            das,
            arr,
            held_left: false,
            held_right: false,
            active: None,
            held_for: 0.0,
            repeat_accum: 0.0,
        }
    }

    /// Advances by `dt` with the current held state and returns the signed
    /// number of columns to shift this frame. With `arr == 0` the count is
    /// `WIDTH`, which the caller clamps by moving until blocked.
    pub fn update(&mut self, left: bool, right: bool, dt: f64) -> i32 {
        let newly_left = left && !self.held_left;
        let newly_right = right && !self.held_right;
        self.held_left = left;
        self.held_right = right;

        let pressed = if newly_right {
            Some(Dir::Right)
        } else if newly_left {
            Some(Dir::Left)
        } else {
            None
        };
        if let Some(dir) = pressed {
            self.active = Some(dir);
            self.held_for = 0.0;
            self.repeat_accum = 0.0;
            return dir.sign();
        }

        // Fall back to the other key if the active one was released. The
        // fallback direction starts with a fresh DAS charge.
        let next = match self.active {
            Some(Dir::Left) if left => Some(Dir::Left),
            Some(Dir::Right) if right => Some(Dir::Right),
            _ if left => Some(Dir::Left),
            _ if right => Some(Dir::Right),
            _ => None,
        };
        if next != self.active {
            self.active = next;
            self.held_for = 0.0;
            self.repeat_accum = 0.0;
            return 0;
        }
        let Some(dir) = self.active else {
            return 0;
        };

        let before = self.held_for;
        self.held_for += dt;
        if self.held_for < self.das {
            return 0;
        }
        // Only the part of this frame after DAS charged counts toward repeats.
        self.repeat_accum += self.held_for - before.max(self.das);
        if self.arr <= 0.0 {
            return dir.sign() * WIDTH;
        }
        // Moving on the exact frame DAS charges feels responsive and matches
        // most modern clients.
        let mut steps = if before < self.das { 1 } else { 0 };
        while self.repeat_accum >= self.arr {
            self.repeat_accum -= self.arr;
            steps += 1;
        }
        dir.sign() * steps.min(WIDTH)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const DT: f64 = 1.0 / 60.0;

    #[test]
    fn tap_moves_exactly_once() {
        let mut a = AutoRepeat::new(0.167, 0.033);
        assert_eq!(a.update(true, false, DT), -1);
        assert_eq!(a.update(false, false, DT), 0);
    }

    #[test]
    fn hold_waits_for_das_then_repeats() {
        // Powers of two keep the float sums exact: 64 frames per second,
        // DAS of 6 frames, one repeat every 3 frames.
        let dt = 1.0 / 64.0;
        let mut a = AutoRepeat::new(6.0 * dt, 3.0 * dt);
        let mut total = a.update(false, true, dt);
        assert_eq!(total, 1);
        let mut first_repeat_frame = None;
        for frame in 1..=64 {
            let s = a.update(false, true, dt);
            if s != 0 && first_repeat_frame.is_none() {
                first_repeat_frame = Some(frame);
            }
            total += s;
        }
        // Initial tap + DAS-charge move + 58 frames / 3 = 19 repeats.
        assert_eq!(first_repeat_frame, Some(6));
        assert_eq!(total, 21);
    }

    #[test]
    fn zero_arr_slides_to_the_wall() {
        let mut a = AutoRepeat::new(0.05, 0.0);
        a.update(true, false, DT);
        a.update(true, false, DT);
        a.update(true, false, DT);
        assert_eq!(a.update(true, false, DT), -WIDTH);
    }

    #[test]
    fn most_recent_direction_wins() {
        let mut a = AutoRepeat::new(0.1, 0.05);
        assert_eq!(a.update(true, false, DT), -1);
        assert_eq!(a.update(true, true, DT), 1);
        // Releasing right falls back to the still-held left, with fresh DAS.
        assert_eq!(a.update(true, false, DT), 0);
    }
}
