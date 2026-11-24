package mapgen

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// The tileset images are drawn pixel by pixel at generation time, so the
// project ships no third-party art. Each painter draws one 16x16 tile at
// (ox, oy) of the sheet.

const tilePx = 16

var (
	grassBase  = color.NRGBA{88, 160, 72, 255}
	grassDark  = color.NRGBA{70, 135, 60, 255}
	grassLight = color.NRGBA{112, 186, 92, 255}
	dirtBase   = color.NRGBA{150, 110, 70, 255}
	dirtDark   = color.NRGBA{128, 92, 58, 255}
	dirtLight  = color.NRGBA{172, 132, 90, 255}
	mortar     = color.NRGBA{70, 70, 82, 255}
	brick      = color.NRGBA{128, 128, 140, 255}
	brickHi    = color.NRGBA{160, 160, 172, 255}
	waterBase  = color.NRGBA{50, 100, 190, 255}
	waterHi    = color.NRGBA{120, 170, 232, 255}
	wood       = color.NRGBA{140, 100, 60, 255}
	woodDark   = color.NRGBA{96, 66, 38, 255}
	leaf       = color.NRGBA{40, 118, 52, 255}
	leafHi     = color.NRGBA{72, 152, 72, 255}
	stone      = color.NRGBA{120, 120, 126, 255}
	stoneHi    = color.NRGBA{162, 162, 168, 255}
	gold       = color.NRGBA{222, 182, 60, 255}
	petalPink  = color.NRGBA{232, 92, 124, 255}
	petalCream = color.NRGBA{244, 224, 132, 255}
)

type sheet struct{ img *image.NRGBA }

func newSheet(w, h int) *sheet { return &sheet{image.NewNRGBA(image.Rect(0, 0, w, h))} }

func (s *sheet) fill(ox, oy, x0, y0, x1, y1 int, c color.NRGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			s.img.SetNRGBA(ox+x, oy+y, c)
		}
	}
}

func (s *sheet) ellipse(ox, oy int, cx, cy, rx, ry float64, c color.NRGBA) {
	for y := 0; y < tilePx; y++ {
		for x := 0; x < tilePx; x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if dx*dx+dy*dy <= 1 {
				s.img.SetNRGBA(ox+x, oy+y, c)
			}
		}
	}
}

func (s *sheet) speckle(ox, oy int, r *splitMix64, n int, colors ...color.NRGBA) {
	for i := 0; i < n; i++ {
		s.img.SetNRGBA(ox+r.intn(tilePx), oy+r.intn(tilePx), colors[r.intn(len(colors))])
	}
}

func (s *sheet) png() ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, s.img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// paintTerrain draws the terrain sheet: 8 columns x 2 rows of 16px tiles.
func paintTerrain(r *splitMix64) *sheet {
	s := newSheet(terrainColumns*tilePx, 2*tilePx)
	at := func(id int) (int, int) { return (id % terrainColumns) * tilePx, (id / terrainColumns) * tilePx }

	ox, oy := at(tileGrass)
	s.fill(ox, oy, 0, 0, tilePx, tilePx, grassBase)
	s.speckle(ox, oy, r, 22, grassDark, grassLight)

	ox, oy = at(tileTallGrass)
	s.fill(ox, oy, 0, 0, tilePx, tilePx, grassBase)
	for i := 0; i < 7; i++ {
		x, y := 1+r.intn(tilePx-2), 3+r.intn(tilePx-4)
		s.fill(ox, oy, x, y-3, x+1, y, grassDark)
		s.img.SetNRGBA(ox+x, oy+y-3, grassLight)
	}

	ox, oy = at(tileDirt)
	s.fill(ox, oy, 0, 0, tilePx, tilePx, dirtBase)
	s.speckle(ox, oy, r, 26, dirtDark, dirtLight)

	ox, oy = at(tileWall)
	s.fill(ox, oy, 0, 0, tilePx, tilePx, mortar)
	for row := 0; row < 4; row++ {
		shift := (row % 2) * 4
		for col := -1; col < 2; col++ {
			x0 := col*8 + shift + 1
			s.fill(ox, oy, max(x0, 0), row*4+1, min(x0+7, tilePx), row*4+4, brick)
			s.fill(ox, oy, max(x0, 0), row*4+1, min(x0+7, tilePx), row*4+2, brickHi)
		}
	}

	for f := 0; f < waterFrames; f++ {
		ox, oy = at(tileWater + f)
		s.fill(ox, oy, 0, 0, tilePx, tilePx, waterBase)
		for _, y := range []int{3, 8, 13} {
			for x := 0; x < tilePx; x++ {
				if (x+f*3+y)%8 < 3 {
					s.img.SetNRGBA(ox+x, oy+y, waterHi)
				}
			}
		}
	}

	ox, oy = at(tileBridge)
	s.fill(ox, oy, 0, 0, tilePx, tilePx, wood)
	for x := 3; x < tilePx; x += 4 {
		s.fill(ox, oy, x, 0, x+1, tilePx, woodDark)
	}
	s.fill(ox, oy, 0, 0, tilePx, 2, woodDark)
	s.fill(ox, oy, 0, tilePx-2, tilePx, tilePx, woodDark)

	// Decorations are drawn on a transparent background so the ground layer
	// shows through; their collision shapes in the tileset match this art.
	ox, oy = at(tileTree)
	s.fill(ox, oy, 6, 9, 10, 16, woodDark)
	s.ellipse(ox, oy, 8, 7, 6.5, 6, leaf)
	s.ellipse(ox, oy, 6.5, 5, 3, 2.5, leafHi)

	ox, oy = at(tileFlower)
	for _, p := range [][2]int{{4, 6}, {11, 4}, {8, 11}} {
		s.fill(ox, oy, p[0], p[1]+1, p[0]+1, p[1]+4, grassDark)
		s.fill(ox, oy, p[0]-1, p[1]-1, p[0]+2, p[1]+2, petalPink)
		s.img.SetNRGBA(ox+p[0], oy+p[1], petalCream)
	}

	// The rock sits left of center on purpose: flipped copies on the map then
	// prove that collision shapes are flipped along with the art.
	ox, oy = at(tileRock)
	s.ellipse(ox, oy, rockCX, rockCY, rockRX, rockRY, stone)
	s.ellipse(ox, oy, 5, 9, 2.5, 1.8, stoneHi)
	return s
}

// paintMarkers draws the embedded marker tileset: a chest and a signpost.
func paintMarkers() *sheet {
	s := newSheet(2*tilePx, tilePx)
	s.fill(0, 0, 2, 6, 14, 15, wood)
	s.fill(0, 0, 2, 6, 14, 8, woodDark)
	s.fill(0, 0, 7, 6, 9, 15, gold)
	s.fill(0, 0, 6, 9, 10, 11, gold)

	ox := tilePx
	s.fill(ox, 0, 7, 8, 9, 16, woodDark)
	s.fill(ox, 0, 2, 2, 14, 9, wood)
	s.fill(ox, 0, 2, 2, 14, 3, woodDark)
	for _, y := range []int{4, 6} {
		s.fill(ox, 0, 4, y, 12, y+1, woodDark)
	}
	return s
}

// ellipseBounds keeps the drawn rock and its collision ellipse in sync.
func ellipseBounds(cx, cy, rx, ry float64) (x, y, w, h float64) {
	return math.Round(cx - rx), math.Round(cy - ry), math.Round(2 * rx), math.Round(2 * ry)
}
