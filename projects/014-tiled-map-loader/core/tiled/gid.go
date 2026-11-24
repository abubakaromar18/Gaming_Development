package tiled

import "github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"

// GID is a global tile ID as stored in layer data and tile objects.
//
// Tiled numbers the tiles of all tilesets in one global space: each tileset
// owns the range [FirstGID, FirstGID+TileCount). Zero means "no tile". The
// top four bits are not part of the ID; they are flags that flip or rotate
// the tile, which lets one tile image appear in eight orientations without
// duplicating art.
type GID uint32

// Flag bits packed into the high end of a GID.
const (
	FlipHorizontal GID = 0x80000000
	FlipVertical   GID = 0x40000000
	FlipDiagonal   GID = 0x20000000 // swap x and y (anti-diagonal flip)
	RotateHex120   GID = 0x10000000 // only meaningful on hexagonal maps
	flagMask           = FlipHorizontal | FlipVertical | FlipDiagonal | RotateHex120
)

// ID returns the global tile ID with every flag bit cleared.
func (g GID) ID() uint32 { return uint32(g &^ flagMask) }

// IsEmpty reports whether the cell holds no tile.
func (g GID) IsEmpty() bool { return g.ID() == 0 }

// Flags returns only the flag bits of g.
func (g GID) Flags() GID { return g & flagMask }

// Transform returns the 2x2 matrix that maps a point relative to the tile's
// center into the flipped/rotated orientation. Tiled applies the diagonal
// flip first, then horizontal, then vertical, so a 90° clockwise rotation is
// stored as Diagonal|Horizontal.
func (g GID) Transform() geom.Mat2 {
	m := geom.Identity
	if g&FlipDiagonal != 0 {
		m = geom.Mat2{A: 0, B: 1, C: 1, D: 0}.Mul(m)
	}
	if g&FlipHorizontal != 0 {
		m = geom.Mat2{A: -1, B: 0, C: 0, D: 1}.Mul(m)
	}
	if g&FlipVertical != 0 {
		m = geom.Mat2{A: 1, B: 0, C: 0, D: -1}.Mul(m)
	}
	return m
}

// TileRef is a GID resolved against the map's tilesets.
type TileRef struct {
	Tileset *Tileset
	LocalID int   // index inside Tileset
	Tile    *Tile // per-tile metadata, or nil when the tile has none
	Flags   GID
}
