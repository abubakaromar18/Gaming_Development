package geom_test

import (
	"math"
	"testing"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
)

func TestOverlapsExcludesTouchingEdges(t *testing.T) {
	a := geom.Rect{X: 0, Y: 0, W: 10, H: 10}
	if a.Overlaps(geom.Rect{X: 10, Y: 0, W: 5, H: 5}) {
		t.Error("edge contact counted as overlap")
	}
	if !a.Overlaps(geom.Rect{X: 9.5, Y: 9.5, W: 5, H: 5}) {
		t.Error("corner overlap missed")
	}
}

func TestRectFromPoints(t *testing.T) {
	got := geom.RectFromPoints([]geom.Vec{{X: 3, Y: -1}, {X: -2, Y: 4}, {X: 0, Y: 0}})
	if got != (geom.Rect{X: -2, Y: -1, W: 5, H: 5}) {
		t.Errorf("got %+v", got)
	}
	if geom.RectFromPoints(nil) != (geom.Rect{}) {
		t.Error("empty input")
	}
}

func TestRotationIsClockwiseOnAYDownScreen(t *testing.T) {
	p := geom.Rotation(90).Apply(geom.Vec{X: 1})
	if math.Abs(p.X) > 1e-12 || math.Abs(p.Y-1) > 1e-12 {
		t.Errorf("right rotated 90° = %+v, want down (0,1)", p)
	}
	m := geom.Mat2{A: 0, B: 1, C: 1, D: 0}
	if m.Mul(m) != geom.Identity {
		t.Error("a reflection applied twice must be the identity")
	}
}
