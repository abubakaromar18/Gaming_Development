// Package world turns a loaded tiled.Map into gameplay structures: a
// collision index built from tile metadata and object layers, a tile
// animation clock, the map's entities, and a small top-down mover. Nothing
// here renders; time arrives as an explicit dt.
package world

import (
	"errors"
	"image"
	"math"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/tiled"
)

// Config names the map conventions that carry collision meaning. They are
// data, not code: designers mark tiles and layers in Tiled, and the game only
// agrees on the names.
type Config struct {
	// SolidProperty is a bool tile property that makes the whole tile solid
	// when the tile has no collision shapes of its own.
	SolidProperty string
	// CollidesProperty is a bool tile-layer property; false excludes the
	// layer from collision (for purely decorative overlays).
	CollidesProperty string
	// CollisionLayer is the name or class of object layers whose shapes are
	// collision geometry rather than entities.
	CollisionLayer string
	// SolidBorders surrounds the map with walls so bodies cannot leave it.
	SolidBorders bool
}

// DefaultConfig returns the conventions used by the bundled demo map.
func DefaultConfig() Config {
	return Config{SolidProperty: "solid", CollidesProperty: "collides", CollisionLayer: "collision", SolidBorders: true}
}

// ErrNoArea is returned for a map without any tile area (an empty infinite map).
var ErrNoArea = errors.New("world: map has no tile area")

// Collider is a static set of axis-aligned boxes indexed by a uniform grid
// whose cells match the map's tiles. A query only inspects the boxes
// registered in the cells it touches, so cost depends on the size of the
// query, not the size of the map.
//
// Collider is immutable after construction and safe for concurrent reads.
type Collider struct {
	tileW, tileH float64
	bounds       image.Rectangle // tile coordinates covered by the grid
	boxes        []geom.Rect
	cells        [][]int32 // box indices per cell, row-major over bounds
	minExtent    float64   // thinnest box or tile side; bounds the sub-step size
}

// NewCollider builds collision geometry from three sources:
//  1. tiles with collision shapes (drawn in Tiled's tile collision editor),
//     transformed by the cell's flip flags;
//  2. tiles whose SolidProperty is true, merged into large rectangles;
//  3. shapes in object layers named or classed CollisionLayer.
func NewCollider(m *tiled.Map, cfg Config) (*Collider, error) {
	bounds := m.TileBounds()
	if bounds.Empty() {
		return nil, ErrNoArea
	}
	c := &Collider{tileW: float64(m.TileWidth), tileH: float64(m.TileHeight), bounds: bounds}
	w, h := bounds.Dx(), bounds.Dy()
	solid := make([]bool, w*h)

	for _, r := range m.Flatten() {
		switch l := r.Layer.(type) {
		case *tiled.TileLayer:
			if l.Properties.Bool(cfg.CollidesProperty, true) {
				c.addTileLayer(m, l, cfg, solid)
			}
		case *tiled.ObjectGroup:
			if isCollisionLayer(l, cfg) {
				c.addObjects(l.Objects)
			}
		}
	}
	for _, cell := range MergeCells(solid, w, h) {
		cell = cell.Add(bounds.Min)
		c.boxes = append(c.boxes, geom.Rect{
			X: float64(cell.Min.X) * c.tileW, Y: float64(cell.Min.Y) * c.tileH,
			W: float64(cell.Dx()) * c.tileW, H: float64(cell.Dy()) * c.tileH,
		})
	}
	if cfg.SolidBorders {
		c.addBorders()
	}
	c.buildIndex()
	return c, nil
}

func isCollisionLayer(og *tiled.ObjectGroup, cfg Config) bool {
	return cfg.CollisionLayer != "" && (og.Name == cfg.CollisionLayer || og.Class == cfg.CollisionLayer)
}

func (c *Collider) addTileLayer(m *tiled.Map, l *tiled.TileLayer, cfg Config, solid []bool) {
	w := c.bounds.Dx()
	for y := l.StartY; y < l.StartY+l.Height; y++ {
		for x := l.StartX; x < l.StartX+l.Width; x++ {
			ref, ok := m.ResolveGID(l.At(x, y))
			if !ok || ref.Tile == nil {
				continue
			}
			if len(ref.Tile.Collision) > 0 {
				c.addTileShapes(ref, x, y)
			} else if ref.Tile.Properties.Bool(cfg.SolidProperty, false) {
				solid[(y-c.bounds.Min.Y)*w+(x-c.bounds.Min.X)] = true
			}
		}
	}
}

// addTileShapes places a tile's collision shapes in map space. Tiled anchors
// tiles larger than the map grid at the cell's bottom-left, and a flipped
// tile must flip its collision shapes too or the wall would not match the art.
func (c *Collider) addTileShapes(ref tiled.TileRef, x, y int) {
	ts := ref.Tileset
	tw, th := float64(ts.TileWidth), float64(ts.TileHeight)
	origin := geom.Vec{X: float64(x) * c.tileW, Y: float64(y+1)*c.tileH - th}
	center := geom.Vec{X: tw / 2, Y: th / 2}
	xf := ref.Flags.Transform()
	for i := range ref.Tile.Collision {
		o := &ref.Tile.Collision[i]
		if o.Shape == tiled.ShapePoint {
			continue
		}
		corners := o.Bounds().Corners()
		for j, p := range corners {
			corners[j] = xf.Apply(p.Sub(center)).Add(center).Add(origin)
		}
		if b := geom.RectFromPoints(corners); b.W > 0 && b.H > 0 {
			c.boxes = append(c.boxes, b)
		}
	}
}

// addObjects uses each shape's bounding box. Polygons become their AABB,
// which is conservative; see the README for the trade-off.
func (c *Collider) addObjects(objs []tiled.Object) {
	for i := range objs {
		if b := objs[i].Bounds(); b.W > 0 && b.H > 0 {
			c.boxes = append(c.boxes, b)
		}
	}
}

func (c *Collider) addBorders() {
	px := c.PixelBounds()
	t := math.Max(c.tileW, c.tileH)
	c.boxes = append(c.boxes,
		geom.Rect{X: px.X - t, Y: px.Y - t, W: px.W + 2*t, H: t}, // top
		geom.Rect{X: px.X - t, Y: px.Bottom(), W: px.W + 2*t, H: t},
		geom.Rect{X: px.X - t, Y: px.Y, W: t, H: px.H}, // left
		geom.Rect{X: px.Right(), Y: px.Y, W: t, H: px.H},
	)
}

func (c *Collider) buildIndex() {
	c.cells = make([][]int32, c.bounds.Dx()*c.bounds.Dy())
	c.minExtent = math.Min(c.tileW, c.tileH)
	for i, b := range c.boxes {
		c.minExtent = math.Min(c.minExtent, math.Min(b.W, b.H))
		x0, y0, x1, y1 := c.cellRange(b)
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				cell := y*c.bounds.Dx() + x
				c.cells[cell] = append(c.cells[cell], int32(i))
			}
		}
	}
}

// cellRange returns the inclusive grid cells r touches, clamped to the grid.
// Clamping (rather than discarding) means border walls outside the map are
// filed in the edge cells, where every query that reaches past the edge will
// find them.
func (c *Collider) cellRange(r geom.Rect) (x0, y0, x1, y1 int) {
	clamp := func(v, hi int) int { return max(0, min(v, hi)) }
	w, h := c.bounds.Dx()-1, c.bounds.Dy()-1
	x0 = clamp(int(math.Floor(r.X/c.tileW))-c.bounds.Min.X, w)
	y0 = clamp(int(math.Floor(r.Y/c.tileH))-c.bounds.Min.Y, h)
	x1 = max(x0, clamp(int(math.Ceil(r.Right()/c.tileW))-1-c.bounds.Min.X, w))
	y1 = max(y0, clamp(int(math.Ceil(r.Bottom()/c.tileH))-1-c.bounds.Min.Y, h))
	return x0, y0, x1, y1
}

// PixelBounds returns the map area in pixels.
func (c *Collider) PixelBounds() geom.Rect {
	return geom.Rect{
		X: float64(c.bounds.Min.X) * c.tileW, Y: float64(c.bounds.Min.Y) * c.tileH,
		W: float64(c.bounds.Dx()) * c.tileW, H: float64(c.bounds.Dy()) * c.tileH,
	}
}

// Boxes returns every collision box. The slice is shared; do not modify it.
func (c *Collider) Boxes() []geom.Rect { return c.boxes }

// Query calls fn once for each box overlapping r.
func (c *Collider) Query(r geom.Rect, fn func(geom.Rect)) {
	for _, i := range c.candidates(r, nil) {
		if b := c.boxes[i]; b.Overlaps(r) {
			fn(b)
		}
	}
}

// Overlaps reports whether r overlaps any box.
func (c *Collider) Overlaps(r geom.Rect) bool {
	for _, i := range c.candidates(r, nil) {
		if c.boxes[i].Overlaps(r) {
			return true
		}
	}
	return false
}

// candidates appends the unique box indices filed in the cells r touches.
// A box spanning several cells appears in each, so duplicates are dropped;
// the lists are short, so a linear check beats a set.
func (c *Collider) candidates(r geom.Rect, dst []int32) []int32 {
	x0, y0, x1, y1 := c.cellRange(r)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
		next:
			for _, i := range c.cells[y*c.bounds.Dx()+x] {
				for _, seen := range dst {
					if seen == i {
						continue next
					}
				}
				dst = append(dst, i)
			}
		}
	}
	return dst
}

// Contacts reports which sides of a moving box hit something.
type Contacts struct {
	Left, Right, Up, Down bool
}

// Any reports whether any side made contact.
func (ct Contacts) Any() bool { return ct.Left || ct.Right || ct.Up || ct.Down }

// MoveAndSlide moves box by delta and stops it at the first box in its way,
// one axis at a time. Resolving x then y means a diagonal push into a wall
// keeps the component along the wall, which is what makes movement "slide"
// instead of sticking.
//
// The motion is split into sub-steps no longer than half the thinnest
// collider (or half the moving box), so a fast body cannot tunnel through a
// thin wall between two steps.
func (c *Collider) MoveAndSlide(box geom.Rect, delta geom.Vec) (geom.Rect, Contacts) {
	var ct Contacts
	if math.IsNaN(delta.X) || math.IsNaN(delta.Y) || math.IsInf(delta.X, 0) || math.IsInf(delta.Y, 0) {
		return box, ct
	}
	step := c.minExtent
	if body := math.Min(box.W, box.H); body > 0 {
		step = math.Min(step, body)
	}
	step /= 2
	n := int(math.Ceil(math.Max(math.Abs(delta.X), math.Abs(delta.Y)) / step))
	if n < 1 {
		n = 1
	}
	// Positions are computed from the start point rather than accumulated,
	// so an unobstructed move lands exactly on box+delta with no drift.
	start := geom.Vec{X: box.X, Y: box.Y}
	movingX, movingY := delta.X != 0, delta.Y != 0
	var scratch []int32
	for i := 1; i <= n && (movingX || movingY); i++ {
		f := float64(i) / float64(n)
		if movingX {
			box.X = start.X + delta.X*f
			scratch = c.candidates(box, scratch[:0])
			for _, j := range scratch {
				b := c.boxes[j]
				if !b.Overlaps(box) {
					continue
				}
				if delta.X > 0 {
					box.X, ct.Right = math.Min(box.X, b.X-box.W), true
				} else {
					box.X, ct.Left = math.Max(box.X, b.Right()), true
				}
			}
			movingX = !(ct.Left || ct.Right)
		}
		if movingY {
			box.Y = start.Y + delta.Y*f
			scratch = c.candidates(box, scratch[:0])
			for _, j := range scratch {
				b := c.boxes[j]
				if !b.Overlaps(box) {
					continue
				}
				if delta.Y > 0 {
					box.Y, ct.Down = math.Min(box.Y, b.Y-box.H), true
				} else {
					box.Y, ct.Up = math.Max(box.Y, b.Bottom()), true
				}
			}
			movingY = !(ct.Up || ct.Down)
		}
	}
	return box, ct
}
