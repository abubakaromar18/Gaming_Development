package bot

import (
	"testing"

	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/core"
	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/fixed"
)

func worldWithRockAt(t *testing.T, offset fixed.Vec) *core.World {
	t.Helper()
	w, err := core.NewWorld(core.DefaultConfig(), 1)
	if err != nil {
		t.Fatal(err)
	}
	w.Ship.Heading = fixed.Angle0 // facing right, +x
	w.Asteroids = []core.Asteroid{{Pos: w.Ship.Pos.Add(offset), Size: core.Large}}
	return w
}

func TestFiresWhenAimed(t *testing.T) {
	in := Decide(worldWithRockAt(t, fixed.V(150, 0)))
	if !in.Has(core.InputFire) || in.Has(core.InputLeft) || in.Has(core.InputRight) {
		t.Fatalf("rock dead ahead: got %08b, want fire without turning", in)
	}
}

func TestTurnsTowardTarget(t *testing.T) {
	// y grows downward, so a rock below a right-facing ship is clockwise.
	if in := Decide(worldWithRockAt(t, fixed.V(0, 150))); !in.Has(core.InputRight) || in.Has(core.InputFire) {
		t.Fatalf("rock below: got %08b, want turn right", in)
	}
	if in := Decide(worldWithRockAt(t, fixed.V(0, -150))); !in.Has(core.InputLeft) {
		t.Fatalf("rock above: got %08b, want turn left", in)
	}
}

func TestAimsAcrossTheWrapSeam(t *testing.T) {
	w := worldWithRockAt(t, fixed.Vec{})
	w.Ship.Pos = fixed.V(780, 300)
	w.Asteroids[0].Pos = fixed.V(60, 300) // 80px to the right, across the edge
	if in := Decide(w); !in.Has(core.InputFire) {
		t.Fatalf("should shoot through the seam, got %08b", in)
	}
}

func TestIdleWhenNothingToDo(t *testing.T) {
	w := worldWithRockAt(t, fixed.V(150, 0))
	w.Asteroids = nil
	if in := Decide(w); in != 0 {
		t.Fatalf("empty field: got %08b", in)
	}
	w = worldWithRockAt(t, fixed.V(150, 0))
	w.Ship.Alive = false
	if in := Decide(w); in != 0 {
		t.Fatalf("dead ship: got %08b", in)
	}
}

func TestOutputIsAlwaysValidInput(t *testing.T) {
	w, _ := core.NewWorld(core.DefaultConfig(), 9)
	for i := 0; i < 5000; i++ {
		in := Decide(w)
		if !in.Valid() {
			t.Fatalf("tick %d: invalid input %08b", i, in)
		}
		w.Step(in)
	}
}
