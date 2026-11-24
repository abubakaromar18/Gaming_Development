package world

import (
	"time"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/tiled"
)

// Animator advances every tile animation in a map from a single clock.
//
// Tiled animations are per tile type, not per cell: every water tile on the
// map shows the same frame at the same moment. One shared clock reproduces
// that, costs nothing per cell, and makes the current frame a pure function
// of elapsed time, which keeps it deterministic and testable.
type Animator struct {
	clock time.Duration
	anims map[animKey]animation
}

type animKey struct {
	ts *tiled.Tileset
	id int
}

type animation struct {
	frames []tiled.Frame
	total  time.Duration
}

// NewAnimator collects the animations of every tileset in m.
func NewAnimator(m *tiled.Map) *Animator {
	a := &Animator{anims: map[animKey]animation{}}
	for _, ts := range m.Tilesets {
		for id, tile := range ts.Tiles {
			var total time.Duration
			for _, f := range tile.Animation {
				total += f.Duration
			}
			if total > 0 {
				a.anims[animKey{ts, id}] = animation{frames: tile.Animation, total: total}
			}
		}
	}
	return a
}

// Advance moves the clock forward by dt. Negative dt is ignored so a
// misbehaving caller cannot run animations backwards.
func (a *Animator) Advance(dt time.Duration) {
	if dt > 0 {
		a.clock += dt
	}
}

// Elapsed returns the total time advanced so far.
func (a *Animator) Elapsed() time.Duration { return a.clock }

// Frame returns the local tile ID to draw for tile id of ts right now: the
// current animation frame for animated tiles, id itself otherwise.
func (a *Animator) Frame(ts *tiled.Tileset, id int) int {
	an, ok := a.anims[animKey{ts, id}]
	if !ok {
		return id
	}
	t := a.clock % an.total
	for _, f := range an.frames {
		if t < f.Duration {
			return f.TileID
		}
		t -= f.Duration
	}
	return an.frames[len(an.frames)-1].TileID
}

// Animated reports whether tile id of ts has an animation.
func (a *Animator) Animated(ts *tiled.Tileset, id int) bool {
	_, ok := a.anims[animKey{ts, id}]
	return ok
}
