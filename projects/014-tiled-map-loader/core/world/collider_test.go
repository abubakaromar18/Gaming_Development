package world_test

import (
	"image"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/tiled"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/world"
)

func noBorders() world.Config {
	cfg := world.DefaultConfig()
	cfg.SolidBorders = false
	return cfg
}

func TestMergeCellsSimpleShapes(t *testing.T) {
	full := []bool{true, true, true, true, true, true, true, true, true}
	if got := world.MergeCells(full, 3, 3); len(got) != 1 || got[0] != image.Rect(0, 0, 3, 3) {
		t.Errorf("3x3 block = %v", got)
	}
	ell := []bool{
		true, false,
		true, false,
		true, true,
	}
	if got := world.MergeCells(ell, 2, 3); len(got) != 2 {
		t.Errorf("L shape = %v, want 2 rectangles", got)
	}
	if got := world.MergeCells(make([]bool, 6), 3, 2); len(got) != 0 {
		t.Errorf("empty grid = %v", got)
	}
}

// Property test: on random grids, the rectangles must cover exactly the
// solid cells, each cell exactly once.
func TestMergeCellsCoversExactly(t *testing.T) {
	rng := rand.New(rand.NewPCG(14, 14))
	for trial := 0; trial < 200; trial++ {
		w, h := 1+rng.IntN(20), 1+rng.IntN(20)
		density := rng.Float64()
		solid := make([]bool, w*h)
		count := 0
		for i := range solid {
			solid[i] = rng.Float64() < density
			if solid[i] {
				count++
			}
		}
		covered := make([]int, w*h)
		rects := world.MergeCells(solid, w, h)
		for _, r := range rects {
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					covered[y*w+x]++
				}
			}
		}
		for i := range solid {
			want := 0
			if solid[i] {
				want = 1
			}
			if covered[i] != want {
				t.Fatalf("trial %d: cell %d covered %d times, solid=%t", trial, i, covered[i], solid[i])
			}
		}
		if len(rects) > count {
			t.Fatalf("trial %d: %d rects for %d cells", trial, len(rects), count)
		}
	}
}

func TestSolidTilesMergeIntoFewBoxes(t *testing.T) {
	m := buildMap(t, []string{
		"######",
		"#....#",
		"######",
	})
	c, err := world.NewCollider(m, noBorders())
	if err != nil {
		t.Fatal(err)
	}
	// 14 solid tiles -> top row, bottom row, and two 1-tile side pieces.
	if n := len(c.Boxes()); n != 4 {
		t.Errorf("got %d boxes, want 4: %v", n, c.Boxes())
	}
	if !c.Overlaps(geom.Rect{X: 0, Y: 0, W: 1, H: 1}) || c.Overlaps(geom.Rect{X: 20, Y: 20, W: 10, H: 10}) {
		t.Error("overlap queries disagree with the map")
	}
}

func TestTileCollisionShapesFollowFlipFlags(t *testing.T) {
	m := buildMap(t, []string{"LR"})
	c, err := world.NewCollider(m, noBorders())
	if err != nil {
		t.Fatal(err)
	}
	want := []geom.Rect{
		{X: 0, Y: 0, W: 8, H: tile},  // left half of cell 0
		{X: 24, Y: 0, W: 8, H: tile}, // flipped: right half of cell 1
	}
	got := c.Boxes()
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("boxes = %v, want %v", got, want)
	}
}

func TestObjectCollisionLayerAndOptOut(t *testing.T) {
	m := buildMap(t, []string{"....", "...."},
		objectLayer("walls", "collision", map[string]any{"x": 4, "y": 4, "width": 10, "height": 2}),
		objectLayer("deco", "", map[string]any{"x": 30, "y": 4, "width": 10, "height": 10}),
	)
	c, _ := world.NewCollider(m, noBorders())
	if len(c.Boxes()) != 1 || c.Boxes()[0] != (geom.Rect{X: 4, Y: 4, W: 10, H: 2}) {
		t.Errorf("boxes = %v", c.Boxes())
	}

	optOut := buildMap(t, []string{"##"})
	optOut.Layers[0].Info().Properties = tiled.Properties{"collides": {Name: "collides", Type: tiled.PropBool, Value: false}}
	c, _ = world.NewCollider(optOut, noBorders())
	if len(c.Boxes()) != 0 {
		t.Errorf("layer with collides=false produced %v", c.Boxes())
	}
}

func TestBordersKeepBodiesInside(t *testing.T) {
	c, _ := world.NewCollider(buildMap(t, []string{"...", "..."}), world.DefaultConfig())
	box := geom.Rect{X: 2, Y: 2, W: 8, H: 8}
	box, ct := c.MoveAndSlide(box, geom.Vec{X: -500, Y: 500})
	if box.X != 0 || box.Bottom() != 2*tile || !ct.Left || !ct.Down {
		t.Errorf("box = %+v contacts %+v", box, ct)
	}
}

func TestMoveAndSlide(t *testing.T) {
	m := buildMap(t, []string{
		"........",
		"....#...",
		"........",
	})
	c, _ := world.NewCollider(m, noBorders())
	wall := geom.Rect{X: 4 * tile, Y: tile, W: tile, H: tile}

	t.Run("stops flush against a wall", func(t *testing.T) {
		box := geom.Rect{X: 0, Y: tile + 4, W: 8, H: 8}
		got, ct := c.MoveAndSlide(box, geom.Vec{X: 100})
		if got.Right() != wall.X || !ct.Right || ct.Left || ct.Down {
			t.Errorf("box = %+v contacts %+v", got, ct)
		}
	})
	t.Run("slides along a wall when pushed diagonally", func(t *testing.T) {
		box := geom.Rect{X: 4*tile + 2, Y: 0, W: 8, H: 8} // just above the wall
		got, ct := c.MoveAndSlide(box, geom.Vec{X: 10, Y: 10})
		if got.Bottom() != wall.Y || got.X != box.X+10 || !ct.Down || ct.Right {
			t.Errorf("box = %+v contacts %+v", got, ct)
		}
	})
	t.Run("touching is not colliding", func(t *testing.T) {
		box := geom.Rect{X: wall.X - 8, Y: wall.Y, W: 8, H: 8}
		got, ct := c.MoveAndSlide(box, geom.Vec{Y: 4})
		if got.Y != box.Y+4 || ct.Any() {
			t.Errorf("box = %+v contacts %+v", got, ct)
		}
	})
	t.Run("non-finite motion is ignored", func(t *testing.T) {
		box := geom.Rect{W: 8, H: 8}
		if got, _ := c.MoveAndSlide(box, geom.Vec{X: math.NaN()}); got != box {
			t.Errorf("NaN moved box to %+v", got)
		}
	})
}

func TestFastBodiesDoNotTunnel(t *testing.T) {
	// A 1-pixel-thin wall and a body moving 10,000 px in one step: without
	// sub-stepping it would jump straight over the wall.
	m := buildMap(t, []string{"........"},
		objectLayer("collision", "", map[string]any{"x": 60, "y": 0, "width": 1, "height": tile}))
	c, _ := world.NewCollider(m, noBorders())
	got, ct := c.MoveAndSlide(geom.Rect{X: 0, Y: 4, W: 4, H: 4}, geom.Vec{X: 10000})
	if got.Right() != 60 || !ct.Right {
		t.Fatalf("body ended at %+v (contacts %+v), want stopped at x=60", got, ct)
	}
}

func TestInfiniteMapWithNegativeCoordinates(t *testing.T) {
	m := buildMap(t, []string{"#."})
	m.Infinite = true
	ground, _ := m.TileLayer("ground")
	ground.StartX, ground.StartY = -5, -3
	c, err := world.NewCollider(m, noBorders())
	if err != nil {
		t.Fatal(err)
	}
	want := geom.Rect{X: -5 * tile, Y: -3 * tile, W: tile, H: tile}
	if len(c.Boxes()) != 1 || c.Boxes()[0] != want {
		t.Fatalf("boxes = %v, want %v", c.Boxes(), want)
	}
	got, _ := c.MoveAndSlide(geom.Rect{X: -4 * tile, Y: -3 * tile, W: 8, H: 8}, geom.Vec{X: -50})
	if got.X != want.Right() {
		t.Errorf("box = %+v", got)
	}
}
