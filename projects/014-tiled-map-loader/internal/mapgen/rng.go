package mapgen

// splitMix64 is a tiny, well-mixed PRNG. It is implemented here rather than
// taken from math/rand so that a given seed produces the same map on every
// Go version: the committed demo map is checked against regeneration.
type splitMix64 struct{ state uint64 }

func (r *splitMix64) next() uint64 {
	r.state += 0x9e3779b97f4a7c15
	z := r.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// float returns a uniform value in [0, 1).
func (r *splitMix64) float() float64 { return float64(r.next()>>11) / (1 << 53) }

// intn returns a value in [0, n). The modulo bias is irrelevant at map scale.
func (r *splitMix64) intn(n int) int { return int(r.next() % uint64(n)) }
