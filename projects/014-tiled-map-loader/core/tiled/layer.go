package tiled

import "github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"

// Layer is one node of the map's layer tree: *TileLayer, *ObjectGroup,
// *ImageLayer or *GroupLayer. Use a type switch to handle each kind.
type Layer interface {
	Info() *LayerInfo
}

// LayerInfo holds the attributes every layer kind shares. Values are the
// layer's own; see Map.Flatten for values combined with parent groups.
type LayerInfo struct {
	ID         int
	Name       string
	Class      string
	Visible    bool
	Opacity    float64
	Offset     geom.Vec // render offset in pixels
	Parallax   geom.Vec // scroll factor relative to the camera; 1 = normal
	Tint       Color
	HasTint    bool
	Properties Properties
}

// Info returns the shared layer attributes.
func (l *LayerInfo) Info() *LayerInfo { return l }

// TileLayer is a dense grid of GIDs. For infinite maps the chunks Tiled
// exports are stitched into one grid whose top-left tile is (StartX, StartY),
// which may be negative.
type TileLayer struct {
	LayerInfo
	StartX, StartY int
	Width, Height  int
	Tiles          []GID // row-major, Width*Height entries
}

// At returns the GID at map tile coordinates (x, y), or 0 outside the layer.
func (l *TileLayer) At(x, y int) GID {
	lx, ly := x-l.StartX, y-l.StartY
	if lx < 0 || ly < 0 || lx >= l.Width || ly >= l.Height {
		return 0
	}
	return l.Tiles[ly*l.Width+lx]
}

// ObjectGroup is a layer of free-placed objects.
type ObjectGroup struct {
	LayerInfo
	DrawOrder string // "topdown" (sort by y) or "index" (file order)
	Color     Color  // editor display color
	Objects   []Object
}

// ImageLayer is a single image, typically a background or parallax plate.
type ImageLayer struct {
	LayerInfo
	Image   string // path relative to the fs.FS root
	RepeatX bool
	RepeatY bool
}

// GroupLayer nests other layers. Its visibility, opacity, offset and parallax
// combine with those of its children.
type GroupLayer struct {
	LayerInfo
	Layers []Layer
}
