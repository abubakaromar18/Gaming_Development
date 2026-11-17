//! Injectable randomness.
//!
//! The core never touches a global or OS RNG. Callers pass in anything that
//! implements [`RandomSource`]; the game ships with [`SplitMix64`], which is
//! tiny, fast, and fully reproducible from a 64-bit seed.

pub trait RandomSource {
    fn next_u64(&mut self) -> u64;

    /// Uniform integer in `0..bound` without modulo bias (Lemire's method).
    fn below(&mut self, bound: u32) -> u32 {
        assert!(bound > 0, "below(0) has no valid result");
        let bound = bound as u64;
        loop {
            let m = (self.next_u64() >> 32) * bound;
            let low = m & 0xFFFF_FFFF;
            if low >= (1u64 << 32) % bound {
                return (m >> 32) as u32;
            }
        }
    }
}

/// Sebastiano Vigna's SplitMix64.
#[derive(Debug, Clone)]
pub struct SplitMix64 {
    state: u64,
}

impl SplitMix64 {
    pub fn new(seed: u64) -> SplitMix64 {
        SplitMix64 { state: seed }
    }
}

impl RandomSource for SplitMix64 {
    fn next_u64(&mut self) -> u64 {
        self.state = self.state.wrapping_add(0x9E37_79B9_7F4A_7C15);
        let mut z = self.state;
        z = (z ^ (z >> 30)).wrapping_mul(0xBF58_476D_1CE4_E5B9);
        z = (z ^ (z >> 27)).wrapping_mul(0x94D0_49BB_1331_11EB);
        z ^ (z >> 31)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn matches_reference_output() {
        // First outputs of the reference C implementation seeded with 1234567.
        let mut r = SplitMix64::new(1234567);
        assert_eq!(r.next_u64(), 6457827717110365317);
        assert_eq!(r.next_u64(), 3203168211198807973);
    }

    #[test]
    fn below_stays_in_range() {
        let mut r = SplitMix64::new(9);
        for _ in 0..10_000 {
            assert!(r.below(7) < 7);
        }
    }
}
