package tiled_test

import (
	"errors"
	"math"
	"testing"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/tiled"
)

func TestGIDFlags(t *testing.T) {
	g := tiled.GID(5) | tiled.FlipHorizontal | tiled.FlipDiagonal
	if g.ID() != 5 || g.IsEmpty() || g.Flags() != tiled.FlipHorizontal|tiled.FlipDiagonal {
		t.Errorf("gid %#x: id %d flags %#x", uint32(g), g.ID(), uint32(g.Flags()))
	}
	if !tiled.FlipVertical.IsEmpty() {
		t.Error("a GID with only flag bits must be empty")
	}
}

func TestTransform(t *testing.T) {
	right := geom.Vec{X: 1}
	cases := []struct {
		flags tiled.GID
		want  geom.Vec // where the unit vector "right" ends up
	}{
		{0, geom.Vec{X: 1}},
		{tiled.FlipHorizontal, geom.Vec{X: -1}},
		{tiled.FlipVertical, geom.Vec{X: 1}},
		{tiled.FlipDiagonal, geom.Vec{Y: 1}},
		// Tiled's "rotate 90° clockwise" sets diagonal+horizontal: right -> down.
		{tiled.FlipDiagonal | tiled.FlipHorizontal, geom.Vec{Y: 1}},
		// Rotate 90° counter-clockwise: right -> up.
		{tiled.FlipDiagonal | tiled.FlipVertical, geom.Vec{Y: -1}},
	}
	for _, c := range cases {
		if got := c.flags.Transform().Apply(right); got != c.want {
			t.Errorf("flags %#x: right -> %+v, want %+v", uint32(c.flags), got, c.want)
		}
	}
	// The three flag bits must give the eight distinct orientations of a
	// square (the dihedral group D4), each one a rotation or reflection.
	seen := map[geom.Mat2]bool{}
	for bits := tiled.GID(0); bits < 8; bits++ {
		m := (bits << 29).Transform()
		if det := m.A*m.D - m.B*m.C; math.Abs(det) != 1 {
			t.Errorf("flags %03b: determinant %v", bits, det)
		}
		seen[m] = true
	}
	if len(seen) != 8 {
		t.Errorf("got %d distinct orientations, want 8", len(seen))
	}
}

func TestParseColor(t *testing.T) {
	c, err := tiled.ParseColor("#ff8000")
	if err != nil || c != (tiled.Color{R: 255, G: 128, A: 255}) {
		t.Errorf("#ff8000 = %+v, %v", c, err)
	}
	c, err = tiled.ParseColor("80ff0000")
	if err != nil || c != (tiled.Color{R: 255, A: 128}) {
		t.Errorf("alpha-first = %+v, %v", c, err)
	}
	for _, bad := range []string{"", "#12345", "#gg0000", "#123456789"} {
		if _, err := tiled.ParseColor(bad); !errors.Is(err, tiled.ErrInvalid) {
			t.Errorf("%q: err = %v", bad, err)
		}
	}
	// RGBA must be premultiplied for image/color compatibility.
	r, _, _, a := tiled.Color{R: 255, A: 128}.RGBA()
	if a != 128*0x101 || r != a {
		t.Errorf("premultiplied = r %d a %d", r, a)
	}
}
