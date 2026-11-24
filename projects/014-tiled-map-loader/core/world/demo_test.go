package world_test

import (
	"math"
	"os"
	"testing"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/tiled"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/world"
)

// These tests run against the committed demo map, the same file the viewer
// ships with, so a broken map fails CI instead of failing at runtime.

func loadDemo(tb testing.TB) *world.World {
	tb.Helper()
	m, err := tiled.Load(os.DirFS("../../maps"), "demo.tmj")
	if err != nil {
		tb.Fatal(err)
	}
	w, err := world.New(m, world.DefaultConfig())
	if err != nil {
		tb.Fatal(err)
	}
	return w
}

const playerSize = 10

func TestDemoMapContent(t *testing.T) {
	w := loadDemo(t)
	if got := w.Map.Properties.String("title", ""); got != "Pondside Ruins" {
		t.Errorf("title = %q", got)
	}
	chests := w.OfClass("chest")
	if len(chests) != 1 || chests[0].Properties.String("loot", "") == "" || chests[0].Properties.Int("gold", 0) <= 0 {
		t.Fatalf("chest = %+v", chests)
	}
	if len(w.OfClass("trigger")) < 3 {
		t.Error("expected signpost, bridge and gate triggers")
	}
	spawn, err := w.Spawn("spawn")
	if err != nil {
		t.Fatal(err)
	}
	if w.Collider.Overlaps(box(spawn)) {
		t.Error("player spawns inside a wall")
	}
	// Merging must have shrunk hundreds of solid tiles to far fewer boxes.
	if n := len(w.Collider.Boxes()); n > 150 {
		t.Errorf("%d collision boxes; greedy merge is not working", n)
	}
}

func box(center geom.Vec) geom.Rect {
	return geom.Rect{X: center.X - playerSize/2, Y: center.Y - playerSize/2, W: playerSize, H: playerSize}
}

// TestDemoChestIsReachable walks the map with a breadth-first search over
// tile centers, asking the collider whether a player-sized box fits, and
// then replays the found route through the real mover.
func TestDemoChestIsReachable(t *testing.T) {
	w := loadDemo(t)
	start, _ := w.Spawn("spawn")
	goal := w.OfClass("chest")[0].Bounds.Center()
	tw, th := float64(w.Map.TileWidth), float64(w.Map.TileHeight)
	cell := func(p geom.Vec) [2]int { return [2]int{int(p.X / tw), int(p.Y / th)} }
	center := func(c [2]int) geom.Vec { return geom.Vec{X: (float64(c[0]) + 0.5) * tw, Y: (float64(c[1]) + 0.5) * th} }

	from := map[[2]int][2]int{cell(start): cell(start)}
	queue := [][2]int{cell(start)}
	reached := func() bool { _, ok := from[cell(goal)]; return ok }
	for len(queue) > 0 && !reached() {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			next := [2]int{cur[0] + d[0], cur[1] + d[1]}
			if _, seen := from[next]; seen {
				continue
			}
			// The chest is not solid, so the player may stand on its cell.
			if w.Collider.Overlaps(box(center(next))) {
				continue
			}
			from[next] = cur
			queue = append(queue, next)
		}
	}
	if !reached() {
		t.Fatal("chest is not reachable from the spawn point")
	}

	var route []geom.Vec
	for c := cell(goal); c != cell(start); c = from[c] {
		route = append([]geom.Vec{center(c)}, route...)
	}
	actor := world.Actor{Box: box(center(cell(start))), Speed: 90}
	const dt = 1.0 / 60
	for _, waypoint := range route {
		for steps := 0; actor.Box.Center().Sub(waypoint).Len() > 0.5; steps++ {
			if steps > 120 {
				t.Fatalf("actor stuck at %+v heading to %+v", actor.Box.Center(), waypoint)
			}
			d := waypoint.Sub(actor.Box.Center())
			intent := d.Scale(1 / math.Max(d.Len(), actor.Speed*dt))
			actor.Step(intent, dt, w.Collider)
		}
	}
	if len(w.Touching(actor.Box, "chest")) != 1 {
		t.Errorf("actor finished at %+v without touching the chest", actor.Box)
	}
}

func BenchmarkLoadDemo(b *testing.B) {
	fsys := os.DirFS("../../maps")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := tiled.Load(fsys, "demo.tmj"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildWorld(b *testing.B) {
	m, err := tiled.Load(os.DirFS("../../maps"), "demo.tmj")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := world.New(m, world.DefaultConfig()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMoveAndSlide(b *testing.B) {
	w := loadDemo(b)
	start, _ := w.Spawn("spawn")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// A long diagonal sweep that hits and slides along several walls.
		w.Collider.MoveAndSlide(box(start), geom.Vec{X: 400, Y: 300})
	}
}
