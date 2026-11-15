package rng

import (
	"testing"

	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/fixed"
)

// The reference C implementation, started from state {1, 2, 3, 4}, produces
// these outputs. Matching them proves the generator is xoshiro256** exactly.
func TestMatchesXoshiroReferenceVectors(t *testing.T) {
	r, ok := FromState([4]uint64{1, 2, 3, 4})
	if !ok {
		t.Fatal("valid state rejected")
	}
	want := []uint64{11520, 0, 1509978240, 1215971899390074240}
	for i, w := range want {
		if got := r.Uint64(); got != w {
			t.Fatalf("output %d = %d, want %d", i, got, w)
		}
	}
}

// New expands the seed with SplitMix64; its published first output for seed 0
// is 0xe220a8397b1dcdaf.
func TestSeedingUsesSplitMix64(t *testing.T) {
	r := New(0)
	if got := r.State()[0]; got != 0xe220a8397b1dcdaf {
		t.Fatalf("first state word = %#x, want SplitMix64 reference 0xe220a8397b1dcdaf", got)
	}
}

func TestRejectsAllZeroState(t *testing.T) {
	if _, ok := FromState([4]uint64{}); ok {
		t.Fatal("all-zero state must be rejected: xoshiro would output zeros forever")
	}
}

// Old replays store only a seed. If anyone changes the seeding or the
// algorithm, every saved replay silently desyncs, so the first outputs for a
// known seed are frozen here as a golden value.
func TestSeedSequenceIsFrozen(t *testing.T) {
	r := New(42)
	got := [3]uint64{r.Uint64(), r.Uint64(), r.Uint64()}
	want := [3]uint64{0x15780b2e0c2ec716, 0x6104d9866d113a7e, 0xae17533239e499a1}
	if got != want {
		t.Fatalf("seed 42 produced %#x, want %#x: changing the RNG breaks saved replays", got, want)
	}
}

func TestSameSeedSameStreamAndCopiesAreIndependent(t *testing.T) {
	a, b := New(7), New(7)
	for i := 0; i < 1000; i++ {
		if a.Uint64() != b.Uint64() {
			t.Fatalf("streams diverged at %d", i)
		}
	}
	snapshot := a // value copy = snapshot
	x := a.Uint64()
	if y := snapshot.Uint64(); x != y {
		t.Fatal("restoring a snapshot should replay the same next value")
	}
	one, two := New(1), New(2)
	if one.Uint64() == two.Uint64() {
		t.Fatal("adjacent seeds should not collide")
	}
}

func TestIntnIsInRangeAndRoughlyUniform(t *testing.T) {
	r := New(99)
	counts := make([]int, 6)
	const n = 60000
	for i := 0; i < n; i++ {
		v := r.Intn(6)
		if v < 0 || v >= 6 {
			t.Fatalf("Intn(6) = %d", v)
		}
		counts[v]++
	}
	for face, c := range counts {
		if c < n/6-600 || c > n/6+600 {
			t.Fatalf("face %d came up %d times, expected about %d", face, c, n/6)
		}
	}
}

func TestIntnPanicsOnNonPositive(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	r := New(1)
	r.Intn(0)
}

func TestRangeHelpers(t *testing.T) {
	r := New(5)
	for i := 0; i < 500; i++ {
		if v := r.Range(-3, 3); v < -3 || v > 3 {
			t.Fatalf("Range = %d", v)
		}
		if v := r.Range(3, -3); v < -3 || v > 3 {
			t.Fatalf("reversed Range = %d", v)
		}
		lo, hi := fixed.FromInt(1), fixed.FromInt(2)
		if v := r.FixedRange(lo, hi); v < lo || v >= hi {
			t.Fatalf("FixedRange = %v", v)
		}
	}
	if r.FixedRange(fixed.One, fixed.One) != fixed.One {
		t.Fatal("empty FixedRange should return lo")
	}
	if r.Chance(0, 10) || !r.Chance(10, 10) {
		t.Fatal("Chance edge cases wrong")
	}
}
