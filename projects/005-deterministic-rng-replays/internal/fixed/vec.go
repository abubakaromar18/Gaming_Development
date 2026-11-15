package fixed

// Vec is a 2D fixed-point vector.
type Vec struct{ X, Y F }

// V builds a vector from whole numbers.
func V(x, y int) Vec { return Vec{FromInt(x), FromInt(y)} }

// Add returns a+b.
func (a Vec) Add(b Vec) Vec { return Vec{a.X + b.X, a.Y + b.Y} }

// Sub returns a-b.
func (a Vec) Sub(b Vec) Vec { return Vec{a.X - b.X, a.Y - b.Y} }

// Scale multiplies both components by s.
func (a Vec) Scale(s F) Vec { return Vec{a.X.Mul(s), a.Y.Mul(s)} }

// LenSq returns the squared length as a raw 64-bit value in 32.32 format.
// Squaring a playfield-sized distance overflows F's ±32768 range, so callers
// compare squared lengths against RawSq of a threshold instead.
func (a Vec) LenSq() int64 { return int64(a.X)*int64(a.X) + int64(a.Y)*int64(a.Y) }

// Len returns the length of the vector.
func (a Vec) Len() F { return F(isqrt(uint64(a.LenSq()))) }

// RawSq squares a scalar in the same 32.32 units LenSq uses.
func RawSq(f F) int64 { return int64(f) * int64(f) }

// FromAngle returns a unit vector pointing along a. With y growing downward on
// screen, increasing the angle turns clockwise.
func FromAngle(a Angle) Vec { return Vec{Cos(a), Sin(a)} }

// WrapVec folds a position into the [0,w)×[0,h) torus.
func WrapVec(p Vec, w, h F) Vec { return Vec{Wrap(p.X, w), Wrap(p.Y, h)} }

// WrapDeltaVec is the shortest vector from a to b on the torus.
func WrapDeltaVec(a, b Vec, w, h F) Vec {
	return Vec{WrapDelta(a.X, b.X, w), WrapDelta(a.Y, b.Y, h)}
}
