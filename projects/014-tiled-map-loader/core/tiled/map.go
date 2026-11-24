package tiled

import (
	"image"
	"strings"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
)

// Map is a loaded, validated Tiled map. Every non-empty GID in it is
// guaranteed to resolve to a tile inside one of Tilesets.
type Map struct {
	Width, Height         int // in tiles; for infinite maps see TileBounds
	TileWidth, TileHeight int
	Infinite              bool
	RenderOrder           string
	Class                 string
	Background            Color
	HasBackground         bool
	Properties            Properties
	Tilesets              []*Tileset // sorted by FirstGID
	Layers                []Layer    // top-level layers, bottom to top
}

// ResolveGID finds the tileset and metadata for g. It returns false for an
// empty cell or an ID no tileset owns.
func (m *Map) ResolveGID(g GID) (TileRef, bool) {
	id := g.ID()
	if id == 0 {
		return TileRef{}, false
	}
	// Tilesets are few (usually under ten), so a reverse linear scan beats a
	// binary search in practice and stays obviously correct.
	for i := len(m.Tilesets) - 1; i >= 0; i-- {
		ts := m.Tilesets[i]
		if id >= ts.FirstGID {
			if !ts.Contains(id) {
				return TileRef{}, false
			}
			local := int(id - ts.FirstGID)
			return TileRef{Tileset: ts, LocalID: local, Tile: ts.Tiles[local], Flags: g.Flags()}, true
		}
	}
	return TileRef{}, false
}

// TileBounds returns the rectangle, in tile coordinates, covered by the map.
// For finite maps that is (0,0)-(Width,Height); for infinite maps it is the
// union of every tile layer's stitched extent.
func (m *Map) TileBounds() image.Rectangle {
	if !m.Infinite {
		return image.Rect(0, 0, m.Width, m.Height)
	}
	var b image.Rectangle
	for _, r := range m.Flatten() {
		if tl, ok := r.Layer.(*TileLayer); ok {
			b = b.Union(image.Rect(tl.StartX, tl.StartY, tl.StartX+tl.Width, tl.StartY+tl.Height))
		}
	}
	return b
}

// ResolvedLayer is a leaf layer together with attributes inherited from its
// parent groups.
type ResolvedLayer struct {
	Layer    Layer
	Path     string // slash-separated group names, e.g. "terrain/ground"
	Visible  bool   // false if the layer or any ancestor is hidden
	Opacity  float64
	Offset   geom.Vec
	Parallax geom.Vec
}

// Flatten returns every non-group layer in draw order (bottom first) with
// group attributes folded in: visibility and opacity multiply, offsets add,
// and parallax factors multiply, matching how Tiled renders nested groups.
func (m *Map) Flatten() []ResolvedLayer {
	var out []ResolvedLayer
	root := ResolvedLayer{Visible: true, Opacity: 1, Parallax: geom.Vec{X: 1, Y: 1}}
	flattenInto(&out, m.Layers, root, nil)
	return out
}

func flattenInto(out *[]ResolvedLayer, layers []Layer, parent ResolvedLayer, path []string) {
	for _, l := range layers {
		info := l.Info()
		r := ResolvedLayer{
			Layer:    l,
			Path:     strings.Join(append(append([]string(nil), path...), info.Name), "/"),
			Visible:  parent.Visible && info.Visible,
			Opacity:  parent.Opacity * info.Opacity,
			Offset:   parent.Offset.Add(info.Offset),
			Parallax: geom.Vec{X: parent.Parallax.X * info.Parallax.X, Y: parent.Parallax.Y * info.Parallax.Y},
		}
		if g, ok := l.(*GroupLayer); ok {
			flattenInto(out, g.Layers, r, append(path, info.Name))
			continue
		}
		*out = append(*out, r)
	}
}

// TileLayer returns the first tile layer whose name or group path matches.
func (m *Map) TileLayer(name string) (*TileLayer, bool) {
	for _, r := range m.Flatten() {
		if tl, ok := r.Layer.(*TileLayer); ok && (r.Path == name || tl.Name == name) {
			return tl, true
		}
	}
	return nil, false
}

// ObjectGroup returns the first object layer whose name or group path matches.
func (m *Map) ObjectGroup(name string) (*ObjectGroup, bool) {
	for _, r := range m.Flatten() {
		if og, ok := r.Layer.(*ObjectGroup); ok && (r.Path == name || og.Name == name) {
			return og, true
		}
	}
	return nil, false
}
