//! Injectable randomness.
//!
//! The core never touches a global or OS RNG. The world owns any
//! [`RandomSource`] it is given; the game ships with [`SplitMix64`], which is
//! tiny, fast, and fully reproducible from a 64-bit seed. Only the drones'
//! wandering and respawn points are random: player movement is deterministic.

pub trait RandomSource {
    fn next_u64(&mut self) -> u64;

    /// Uniform float in `[0, 1)` built from the top 24 bits, which is all
    /// the precision an `f32` mantissa can hold.
    fn next_f32(&mut self) -> f32 {
        (self.next_u64() >> 40) as f32 / (1u64 << 24) as f32
    }

    /// Uniform float in `[lo, hi)`.
    fn range(&mut self, lo: f32, hi: f32) -> f32 {
        lo + (hi - lo) * self.next_f32()
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
    fn floats_stay_in_range() {
        let mut r = SplitMix64::new(9);
        for _ in 0..10_000 {
            let f = r.next_f32();
            assert!((0.0..1.0).contains(&f));
            let g = r.range(-3.0, 5.0);
            assert!((-3.0..5.0).contains(&g));
        }
    }
}
