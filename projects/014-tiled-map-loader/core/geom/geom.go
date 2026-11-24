// Package geom holds the tiny amount of 2D math shared by the loader and the
// world: vectors, axis-aligned rectangles, and 2x2 linear transforms.
// Coordinates follow Tiled and most 2D engines: x grows right, y grows down.
package geom

import "math"

// Vec is a 2D point or displacement in pixels.
type Vec struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Add returns v + o.
func (v Vec) Add(o Vec) Vec { return Vec{v.X + o.X, v.Y + o.Y} }

// Sub returns v - o.
func (v Vec) Sub(o Vec) Vec { return Vec{v.X - o.X, v.Y - o.Y} }

// Scale returns v * s.
func (v Vec) Scale(s float64) Vec { return Vec{v.X * s, v.Y * s} }

// Len returns the Euclidean length of v.
func (v Vec) Len() float64 { return math.Hypot(v.X, v.Y) }

// Rect is an axis-aligned rectangle with its top-left corner at (X, Y).
type Rect struct {
	X, Y, W, H float64
}

// RectFromPoints returns the smallest Rect containing every point.
// It returns the zero Rect for an empty slice.
func RectFromPoints(pts []Vec) Rect {
	if len(pts) == 0 {
		return Rect{}
	}
	minX, minY := pts[0].X, pts[0].Y
	maxX, maxY := minX, minY
	for _, p := range pts[1:] {
		minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
		minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
	}
	return Rect{minX, minY, maxX - minX, maxY - minY}
}

// Right returns the x coordinate of the right edge.
func (r Rect) Right() float64 { return r.X + r.W }

// Bottom returns the y coordinate of the bottom edge.
func (r Rect) Bottom() float64 { return r.Y + r.H }

// Center returns the midpoint of r.
func (r Rect) Center() Vec { return Vec{r.X + r.W/2, r.Y + r.H/2} }

// Translate returns r moved by d.
func (r Rect) Translate(d Vec) Rect { return Rect{r.X + d.X, r.Y + d.Y, r.W, r.H} }

// Overlaps reports whether r and o share interior area. Rectangles that only
// touch along an edge do not overlap; this is what lets a body slide along a
// wall it is resting against without being reported as stuck.
func (r Rect) Overlaps(o Rect) bool {
	return r.X < o.Right() && o.X < r.Right() && r.Y < o.Bottom() && o.Y < r.Bottom()
}

// Contains reports whether p lies inside r (left/top inclusive).
func (r Rect) Contains(p Vec) bool {
	return p.X >= r.X && p.X < r.Right() && p.Y >= r.Y && p.Y < r.Bottom()
}

// Corners returns the four corners of r in clockwise order from top-left.
func (r Rect) Corners() []Vec {
	return []Vec{{r.X, r.Y}, {r.Right(), r.Y}, {r.Right(), r.Bottom()}, {r.X, r.Bottom()}}
}

// Mat2 is a 2x2 linear transform [A B; C D] applied as (A*x + B*y, C*x + D*y).
type Mat2 struct {
	A, B, C, D float64
}

// Identity is the transform that leaves points unchanged.
var Identity = Mat2{1, 0, 0, 1}

// Mul returns m*o, the transform that applies o first and then m.
func (m Mat2) Mul(o Mat2) Mat2 {
	return Mat2{
		A: m.A*o.A + m.B*o.C, B: m.A*o.B + m.B*o.D,
		C: m.C*o.A + m.D*o.C, D: m.C*o.B + m.D*o.D,
	}
}

// Apply transforms v.
func (m Mat2) Apply(v Vec) Vec { return Vec{m.A*v.X + m.B*v.Y, m.C*v.X + m.D*v.Y} }

// Rotation returns a clockwise rotation (on a y-down screen) by deg degrees.
func Rotation(deg float64) Mat2 {
	s, c := math.Sincos(deg * math.Pi / 180)
	return Mat2{c, -s, s, c}
}
