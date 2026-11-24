package tiled

import (
	"image"
	"time"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
)

// Tileset is a grid of equally sized tiles cut from one image, plus optional
// per-tile metadata.
type Tileset struct {
	FirstGID   uint32
	Source     string // path of the external .tsj file, empty when embedded
	Name       string
	Class      string
	TileWidth  int
	TileHeight int
	Margin     int // pixels around the whole image
	Spacing    int // pixels between neighbouring tiles
	TileCount  int
	Columns    int
	// Image is the tileset image path relative to the root of the fs.FS the
	// map was loaded from, ready to be opened with the same filesystem.
	Image       string
	ImageWidth  int
	ImageHeight int
	TileOffset  geom.Vec // drawing offset applied to every tile
	Properties  Properties
	// Tiles holds metadata for the tiles that have any, keyed by local ID.
	// Most tiles have none, so this is sparse.
	Tiles map[int]*Tile
}

// Tile is the metadata Tiled stores for an individual tile.
type Tile struct {
	ID          int
	Class       string
	Probability float64 // terrain-brush weight; 1 when unset
	Properties  Properties
	// Collision holds the shapes drawn in Tiled's collision editor, in
	// pixels relative to the tile's top-left corner.
	Collision []Object
	Animation []Frame
}

// Frame is one step of a tile animation.
type Frame struct {
	TileID   int // local ID inside the same tileset
	Duration time.Duration
}

// Contains reports whether the global ID id belongs to this tileset.
func (ts *Tileset) Contains(id uint32) bool {
	return id >= ts.FirstGID && id < ts.FirstGID+uint32(ts.TileCount)
}

// SourceRect returns the pixel rectangle of local tile id inside Image,
// honoring margin and spacing.
func (ts *Tileset) SourceRect(id int) image.Rectangle {
	col, row := id%ts.Columns, id/ts.Columns
	x := ts.Margin + col*(ts.TileWidth+ts.Spacing)
	y := ts.Margin + row*(ts.TileHeight+ts.Spacing)
	return image.Rect(x, y, x+ts.TileWidth, y+ts.TileHeight)
}
