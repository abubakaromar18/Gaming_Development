package world_test

import (
	"testing"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/world"
)

func explorerMap(t *testing.T) *world.Explorer {
	t.Helper()
	m := buildMap(t, []string{"......", "......"},
		objectLayer("things", "",
			map[string]any{"name": "start", "type": "spawn", "x": 8, "y": 8, "point": true},
			map[string]any{"name": "sign", "type": "trigger", "x": 24, "y": 0, "width": 8, "height": 16,
				"properties": []any{map[string]any{"name": "message", "type": "string", "value": "Hello"}}},
			map[string]any{"name": "chest", "type": "chest", "x": 64, "y": 0, "width": 16, "height": 16,
				"properties": []any{
					map[string]any{"name": "gold", "type": "int", "value": 25},
					map[string]any{"name": "loot", "type": "string", "value": "Lantern"},
				}},
		))
	w, err := world.New(m, world.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	e, err := world.NewExplorer(w, world.DefaultExplorerConfig())
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestExplorerReadsSignsAndLootsChestOnce(t *testing.T) {
	e := explorerMap(t)
	const dt = 1.0 / 60
	right := geom.Vec{X: 1}
	if e.Player.Box.Center() != (geom.Vec{X: 8, Y: 8}) {
		t.Fatalf("player not centered on spawn: %+v", e.Player.Box)
	}

	sawSign := false
	for i := 0; i < 120 && e.Gold == 0; i++ {
		e.Update(right, dt)
		sawSign = sawSign || e.Message == "Hello"
	}
	if !sawSign {
		t.Error("walking past the sign never showed its message")
	}
	if e.Gold != 25 || len(e.Inventory) != 1 || e.Inventory[0] != "Lantern" {
		t.Fatalf("gold %d inventory %v", e.Gold, e.Inventory)
	}
	if e.Message != "Found Lantern and 25 gold!" {
		t.Errorf("message = %q", e.Message)
	}
	// Standing on the open chest must not pay out again, and the reward
	// message must outlive the "empty" hint until it expires.
	for i := 0; i < 60; i++ {
		e.Update(geom.Vec{}, dt)
	}
	if e.Gold != 25 || e.Message != "Found Lantern and 25 gold!" {
		t.Errorf("after 1s: gold %d message %q", e.Gold, e.Message)
	}
	for i := 0; i < 180; i++ {
		e.Update(geom.Vec{}, dt)
	}
	if e.Message != "The chest is empty." {
		t.Errorf("after 4s: message %q", e.Message)
	}
	// Walking away clears the message once its hold time passes.
	for i := 0; i < 60; i++ {
		e.Update(geom.Vec{X: -1}, dt)
	}
	if e.Message != "" {
		t.Errorf("message lingered: %q", e.Message)
	}
}

func TestExplorerAdvancesAnimationClock(t *testing.T) {
	e := explorerMap(t)
	e.Update(geom.Vec{}, 0.5)
	if got := e.World.Animator.Elapsed().Seconds(); got != 0.5 {
		t.Errorf("animation clock = %v s", got)
	}
}
