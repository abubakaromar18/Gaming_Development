// Package rng is a seedable, snapshot-able PRNG for deterministic simulation.
//
// Why not math/rand? Two reasons. First, the replay format depends on the
// exact sequence of numbers, so the algorithm must be pinned in this
// repository, not in whatever Go version builds it. Second, a replay player
// that seeks needs to snapshot the generator's state together with the rest
// of the world; Rand is a plain value type, so copying a World copies its RNG.
package rng

import "github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/fixed"

// Rand is xoshiro256** (Blackman & Vigna, 2018): fast, 256 bits of state, and
// statistically excellent for games. The zero value is invalid; use New.
type Rand struct {
	s [4]uint64
}

// New seeds a generator. Any 64-bit seed is fine, including 0: the seed is
// expanded through SplitMix64, as the xoshiro authors recommend, so similar
// seeds still produce unrelated streams and the state is never all zeros.
func New(seed uint64) Rand {
	var r Rand
	for i := range r.s {
		seed += 0x9e3779b97f4a7c15
		z := seed
		z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
		z = (z ^ (z >> 27)) * 0x94d049bb133111eb
		r.s[i] = z ^ (z >> 31)
	}
	return r
}

// FromState rebuilds a generator from a raw state, e.g. to check the
// reference test vectors. It returns false for the all-zero state, which
// xoshiro can never leave.
func FromState(s [4]uint64) (Rand, bool) {
	if s == [4]uint64{} {
		return Rand{}, false
	}
	return Rand{s: s}, true
}

// State exposes the raw state for hashing.
func (r *Rand) State() [4]uint64 { return r.s }

func rotl(x uint64, k uint) uint64 { return (x << k) | (x >> (64 - k)) }

// Uint64 returns the next 64 random bits.
func (r *Rand) Uint64() uint64 {
	result := rotl(r.s[1]*5, 7) * 9
	t := r.s[1] << 17
	r.s[2] ^= r.s[0]
	r.s[3] ^= r.s[1]
	r.s[1] ^= r.s[2]
	r.s[0] ^= r.s[3]
	r.s[2] ^= t
	r.s[3] = rotl(r.s[3], 45)
	return result
}

// Uint32 returns 32 random bits, taken from the high half (the strongest bits).
func (r *Rand) Uint32() uint32 { return uint32(r.Uint64() >> 32) }

// Intn returns a uniform value in [0, n). It uses rejection sampling so that
// no value is more likely than another (plain modulo is biased whenever n
// doesn't divide 2^64). It panics if n <= 0, a programming error.
func (r *Rand) Intn(n int) int {
	if n <= 0 {
		panic("rng: Intn called with n <= 0")
	}
	un := uint64(n)
	limit := -un % un // == 2^64 mod n; values below it are rejected
	for {
		v := r.Uint64()
		if v >= limit {
			return int(v % un)
		}
	}
}

// Range returns a uniform integer in [lo, hi].
func (r *Rand) Range(lo, hi int) int {
	if hi < lo {
		lo, hi = hi, lo
	}
	return lo + r.Intn(hi-lo+1)
}

// FixedRange returns a uniform fixed-point value in [lo, hi).
func (r *Rand) FixedRange(lo, hi fixed.F) fixed.F {
	if hi <= lo {
		return lo
	}
	return lo + fixed.F(r.Intn(int(hi-lo)))
}

// Angle returns a uniformly random heading.
func (r *Rand) Angle() fixed.Angle { return fixed.Angle(r.Uint64() >> 56) }

// Chance reports true with probability num/den.
func (r *Rand) Chance(num, den int) bool { return r.Intn(den) < num }
