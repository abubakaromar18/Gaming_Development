// Package fixed implements 16.16 fixed-point arithmetic and integer trigonometry.
//
// A replay only works if the simulation produces bit-identical results every
// time it runs, on every machine. Floating point can't promise that: Go is
// allowed to fuse x*y+z into one FMA instruction on arm64, ppc64 and s390x,
// which rounds differently from the separate multiply and add used on amd64.
// Integer math has no such freedom, so the whole simulation runs on F values
// and converts to float only at the very edge, when the renderer draws.
package fixed

import "strconv"

// F is a signed 16.16 fixed-point number: the top 16 bits hold the integer
// part and the bottom 16 bits hold the fraction. The range is roughly
// ±32768 with a resolution of 1/65536, plenty for a playfield in pixels.
type F int32

// Shift is the number of fractional bits.
const Shift = 16

// One is the fixed-point value 1.0.
const One F = 1 << Shift

// FromInt converts an integer to fixed point.
func FromInt(i int) F { return F(i << Shift) }

// FromRatio returns num/den in fixed point, for writing constants like 0.99
// as FromRatio(99, 100) without going through a float.
func FromRatio(num, den int) F { return F((int64(num) << Shift) / int64(den)) }

// Mul multiplies two fixed-point numbers using a 64-bit intermediate so the
// product doesn't overflow before it is shifted back down.
func (a F) Mul(b F) F { return F((int64(a) * int64(b)) >> Shift) }

// Div divides a by b. Dividing by zero panics, as integer division does.
func (a F) Div(b F) F { return F((int64(a) << Shift) / int64(b)) }

// Int truncates toward negative infinity.
func (a F) Int() int { return int(a >> Shift) }

// Float64 converts to float64. Only presentation code should call this.
func (a F) Float64() float64 { return float64(a) / float64(One) }

// Float32 converts to float32. Only presentation code should call this.
func (a F) Float32() float32 { return float32(a) / float32(One) }

// Abs returns |a|.
func (a F) Abs() F {
	if a < 0 {
		return -a
	}
	return a
}

// String formats the value with four decimals, for debugging and test output.
func (a F) String() string { return strconv.FormatFloat(a.Float64(), 'f', 4, 64) }

// Wrap folds v into [0, size), which is how objects leave one edge of the
// Asteroids playfield and re-enter on the opposite side.
func Wrap(v, size F) F {
	v %= size
	if v < 0 {
		v += size
	}
	return v
}

// WrapDelta returns the shortest signed distance from a to b on a ring of the
// given size. Collision and targeting use it so an asteroid touching the
// right edge can hit a ship touching the left edge.
func WrapDelta(a, b, size F) F {
	d := Wrap(b-a, size)
	if d > size/2 {
		d -= size
	}
	return d
}

// Sqrt returns the square root of a non-negative fixed-point value using
// integer Newton iteration. Negative input returns 0.
func Sqrt(a F) F {
	if a <= 0 {
		return 0
	}
	// sqrt(a / 2^16) * 2^16 == sqrt(a * 2^16), so take the integer square
	// root of the raw value scaled up by another 16 bits.
	return F(isqrt(uint64(a) << Shift))
}

// isqrt is floor(sqrt(n)) by Newton's method on integers.
func isqrt(n uint64) uint64 {
	if n < 2 {
		return n
	}
	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}
