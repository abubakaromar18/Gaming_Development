package world

import (
	"fmt"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/tiled"
)

// Entity is a gameplay object read from an object layer: a spawn point, a
// chest, a trigger zone. Games map Class to behavior and read tuning values
// from Properties, so designers can add content without code changes.
type Entity struct {
	ID         int
	Name       string
	Class      string
	Layer      string // group path of the source layer
	Shape      tiled.Shape
	GID        tiled.GID // tile objects only
	Bounds     geom.Rect
	Properties tiled.Properties
}

// Position returns the entity's anchor: the point itself for point objects,
// the center of the bounds for everything else.
func (e Entity) Position() geom.Vec {
	if e.Shape == tiled.ShapePoint {
		return geom.Vec{X: e.Bounds.X, Y: e.Bounds.Y}
	}
	return e.Bounds.Center()
}

// World bundles everything gameplay needs from a map.
type World struct {
	Map      *tiled.Map
	Collider *Collider
	Animator *Animator
	Entities []Entity
}

// New builds a World from a loaded map.
func New(m *tiled.Map, cfg Config) (*World, error) {
	col, err := NewCollider(m, cfg)
	if err != nil {
		return nil, err
	}
	w := &World{Map: m, Collider: col, Animator: NewAnimator(m)}
	for _, r := range m.Flatten() {
		og, ok := r.Layer.(*tiled.ObjectGroup)
		if !ok || isCollisionLayer(og, cfg) {
			continue
		}
		for i := range og.Objects {
			o := &og.Objects[i]
			w.Entities = append(w.Entities, Entity{
				ID: o.ID, Name: o.Name, Class: o.Class, Layer: r.Path,
				Shape: o.Shape, GID: o.GID, Bounds: o.Bounds(), Properties: o.Properties,
			})
		}
	}
	return w, nil
}

// OfClass returns every entity with the given class, in map order.
func (w *World) OfClass(class string) []Entity {
	var out []Entity
	for _, e := range w.Entities {
		if e.Class == class {
			out = append(out, e)
		}
	}
	return out
}

// Spawn returns the position of the first entity of class, or an error
// naming what the map is missing, which is far easier to debug than a player
// silently appearing at (0, 0).
func (w *World) Spawn(class string) (geom.Vec, error) {
	for _, e := range w.Entities {
		if e.Class == class {
			return e.Position(), nil
		}
	}
	return geom.Vec{}, fmt.Errorf("world: map has no object of class %q", class)
}

// Touching returns the entities of class whose area overlaps box. Point
// entities count when box contains them.
func (w *World) Touching(box geom.Rect, class string) []Entity {
	var out []Entity
	for _, e := range w.Entities {
		if e.Class != class {
			continue
		}
		if (e.Shape == tiled.ShapePoint && box.Contains(e.Position())) || e.Bounds.Overlaps(box) {
			out = append(out, e)
		}
	}
	return out
}

// Actor is a top-down body that walks through the world's collision.
type Actor struct {
	Box   geom.Rect
	Speed float64 // pixels per second
}

// Step moves the actor along intent (each axis in [-1, 1]) for dt seconds.
// Diagonal intent is normalized so moving diagonally is not ~41% faster.
func (a *Actor) Step(intent geom.Vec, dt float64, c *Collider) Contacts {
	if l := intent.Len(); l > 1 {
		intent = intent.Scale(1 / l)
	}
	var ct Contacts
	a.Box, ct = c.MoveAndSlide(a.Box, intent.Scale(a.Speed*dt))
	return ct
}

// Follow returns the top-left corner of a view of size view that centers on
// target but never shows space outside bounds. A map smaller than the view is
// centered instead.
func Follow(target, view geom.Vec, bounds geom.Rect) geom.Vec {
	axis := func(t, v, lo, size float64) float64 {
		if size <= v {
			return lo - (v-size)/2
		}
		return min(max(t-v/2, lo), lo+size-v)
	}
	return geom.Vec{
		X: axis(target.X, view.X, bounds.X, bounds.W),
		Y: axis(target.Y, view.Y, bounds.Y, bounds.H),
	}
}
