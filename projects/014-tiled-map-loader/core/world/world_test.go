package world_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/world"
)

func TestAnimatorFollowsFrameDurations(t *testing.T) {
	m := buildMap(t, []string{"w"})
	a := world.NewAnimator(m)
	ts := m.Tilesets[0]
	steps := []struct {
		advance time.Duration
		want    int
	}{
		{0, 2},
		{99 * time.Millisecond, 2},
		{1 * time.Millisecond, 3},   // 100ms: second frame
		{199 * time.Millisecond, 3}, // 299ms
		{1 * time.Millisecond, 2},   // 300ms: loops
		{-time.Hour, 2},             // negative dt ignored
		{3*time.Second + 150*time.Millisecond, 3},
	}
	for i, s := range steps {
		a.Advance(s.advance)
		if got := a.Frame(ts, 2); got != s.want {
			t.Errorf("step %d (t=%v): frame %d, want %d", i, a.Elapsed(), got, s.want)
		}
	}
	if !a.Animated(ts, 2) || a.Animated(ts, 0) || a.Frame(ts, 0) != 0 {
		t.Error("non-animated tiles must map to themselves")
	}
}

func TestEntitiesSpawnAndTriggers(t *testing.T) {
	m := buildMap(t, []string{"....", "...."},
		objectLayer("actors", "",
			map[string]any{"name": "start", "type": "spawn", "x": 8, "y": 8, "point": true},
			map[string]any{"name": "door", "type": "trigger", "x": 32, "y": 0, "width": 16, "height": 16,
				"properties": []any{map[string]any{"name": "to", "type": "string", "value": "cave"}}},
		),
		objectLayer("collision", "", map[string]any{"x": 0, "y": 0, "width": 4, "height": 4, "type": "trigger"}),
	)
	w, err := world.New(m, world.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Entities) != 2 {
		t.Fatalf("collision-layer objects leaked into entities: %d", len(w.Entities))
	}
	if p, err := w.Spawn("spawn"); err != nil || p != (geom.Vec{X: 8, Y: 8}) {
		t.Errorf("spawn = %+v, %v", p, err)
	}
	if _, err := w.Spawn("boss"); err == nil || !strings.Contains(err.Error(), `"boss"`) {
		t.Errorf("missing spawn error = %v", err)
	}
	hits := w.Touching(geom.Rect{X: 40, Y: 4, W: 4, H: 4}, "trigger")
	if len(hits) != 1 || hits[0].Properties.String("to", "") != "cave" || hits[0].Position() != (geom.Vec{X: 40, Y: 8}) {
		t.Errorf("touching = %+v", hits)
	}
	if len(w.Touching(geom.Rect{X: 0, Y: 0, W: 10, H: 10}, "spawn")) != 1 {
		t.Error("point entity inside box not reported")
	}
	if len(w.OfClass("trigger")) != 1 {
		t.Error("OfClass")
	}
}

func TestActorNormalizesDiagonalSpeed(t *testing.T) {
	c, _ := world.NewCollider(buildMap(t, make20x20()), world.DefaultConfig())
	a := world.Actor{Box: geom.Rect{X: 100, Y: 100, W: 8, H: 8}, Speed: 60}
	a.Step(geom.Vec{X: 1, Y: 1}, 0.5, c)
	moved := geom.Vec{X: a.Box.X - 100, Y: a.Box.Y - 100}.Len()
	if math.Abs(moved-30) > 1e-9 {
		t.Errorf("diagonal step moved %v px, want 30", moved)
	}
}

func make20x20() []string {
	rows := make([]string, 20)
	for i := range rows {
		rows[i] = strings.Repeat(".", 20)
	}
	return rows
}

func TestFollowClampsToBounds(t *testing.T) {
	bounds := geom.Rect{W: 640, H: 480}
	view := geom.Vec{X: 320, Y: 240}
	cases := []struct{ target, want geom.Vec }{
		{geom.Vec{X: 320, Y: 240}, geom.Vec{X: 160, Y: 120}}, // centered
		{geom.Vec{X: 10, Y: 10}, geom.Vec{X: 0, Y: 0}},       // top-left corner
		{geom.Vec{X: 630, Y: 470}, geom.Vec{X: 320, Y: 240}}, // bottom-right corner
	}
	for _, c := range cases {
		if got := world.Follow(c.target, view, bounds); got != c.want {
			t.Errorf("Follow(%+v) = %+v, want %+v", c.target, got, c.want)
		}
	}
	// A map narrower than the view is centered, not pinned to one side.
	if got := world.Follow(geom.Vec{}, view, geom.Rect{W: 100, H: 480}); got.X != -110 {
		t.Errorf("small map x = %v, want -110", got.X)
	}
}

func TestEmptyInfiniteMapIsRejected(t *testing.T) {
	m := buildMap(t, []string{"."})
	m.Infinite = true
	ground, _ := m.TileLayer("ground")
	ground.Width, ground.Height, ground.Tiles = 0, 0, nil
	if _, err := world.New(m, world.DefaultConfig()); err != world.ErrNoArea {
		t.Fatalf("err = %v, want ErrNoArea", err)
	}
}
