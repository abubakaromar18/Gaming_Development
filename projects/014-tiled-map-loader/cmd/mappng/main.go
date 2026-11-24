// Command mappng rasterizes a Tiled map to a PNG with only the standard
// library. It exercises the same rendering rules as the Ebitengine viewer
// (tileset source rectangles, flip flags, layer opacity and offsets, tile
// objects, animation frames) without needing a GPU or window.
//
//	go run ./cmd/mappng -scale 2 -collision maps/demo.tmj preview.png
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/tiled"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/world"
)

func main() {
	scale := flag.Int("scale", 1, "integer upscale factor (nearest neighbour)")
	at := flag.Duration("time", 0, "animation clock to render, e.g. 450ms")
	overlay := flag.Bool("collision", false, "outline collision boxes and entities")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: mappng [flags] <map.tmj> <out.png>")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 2 || *scale < 1 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(flag.Arg(0), flag.Arg(1), *scale, *at, *overlay); err != nil {
		fmt.Fprintln(os.Stderr, "mappng:", err)
		os.Exit(1)
	}
}

func run(in, out string, scale int, at time.Duration, overlay bool) error {
	fsys := os.DirFS(filepath.Dir(in))
	m, err := tiled.Load(fsys, filepath.Base(in))
	if err != nil {
		return err
	}
	wd, err := world.New(m, world.DefaultConfig())
	if err != nil {
		return err
	}
	wd.Animator.Advance(at)
	sheets, err := loadSheets(fsys, m)
	if err != nil {
		return err
	}

	px := wd.Collider.PixelBounds()
	origin := geom.Vec{X: px.X, Y: px.Y}
	canvas := image.NewNRGBA(image.Rect(0, 0, int(px.W), int(px.H)))
	if m.HasBackground {
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(m.Background), image.Point{}, draw.Src)
	}
	r := renderer{m: m, w: wd, sheets: sheets, dst: canvas, origin: origin}
	for _, l := range m.Flatten() {
		if l.Visible {
			r.layer(l)
		}
	}
	if overlay {
		for _, b := range wd.Collider.Boxes() {
			r.outline(b, color.NRGBA{255, 60, 60, 255})
		}
		for _, e := range wd.Entities {
			r.outline(e.Bounds, color.NRGBA{255, 230, 80, 255})
		}
	}

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	if err := png.Encode(f, upscale(canvas, scale)); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func loadSheets(fsys fs.FS, m *tiled.Map) (map[*tiled.Tileset]image.Image, error) {
	sheets := map[*tiled.Tileset]image.Image{}
	for _, ts := range m.Tilesets {
		f, err := fsys.Open(ts.Image)
		if err != nil {
			return nil, fmt.Errorf("tileset %q: %w", ts.Name, err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("tileset %q image: %w", ts.Name, err)
		}
		sheets[ts] = img
	}
	return sheets, nil
}

type renderer struct {
	m      *tiled.Map
	w      *world.World
	sheets map[*tiled.Tileset]image.Image
	dst    *image.NRGBA
	origin geom.Vec
}

func (r *renderer) layer(l tiled.ResolvedLayer) {
	switch layer := l.Layer.(type) {
	case *tiled.TileLayer:
		for y := layer.StartY; y < layer.StartY+layer.Height; y++ {
			for x := layer.StartX; x < layer.StartX+layer.Width; x++ {
				// Tiles taller than the grid grow upward from the cell's
				// bottom-left corner, as in Tiled.
				pos := geom.Vec{X: float64(x * r.m.TileWidth), Y: float64((y + 1) * r.m.TileHeight)}
				r.tile(layer.At(x, y), pos.Add(l.Offset), l.Opacity)
			}
		}
	case *tiled.ObjectGroup:
		for i := range layer.Objects {
			o := &layer.Objects[i]
			if o.Shape == tiled.ShapeTile && o.Visible {
				r.tile(o.GID, geom.Vec{X: o.X, Y: o.Y}.Add(l.Offset), l.Opacity)
			}
		}
	}
}

// tile draws gid with its bottom-left corner at pos (map pixels).
func (r *renderer) tile(gid tiled.GID, pos geom.Vec, opacity float64) {
	ref, ok := r.m.ResolveGID(gid)
	if !ok {
		return
	}
	ts := ref.Tileset
	src := ts.SourceRect(r.w.Animator.Frame(ts, ref.LocalID))
	sheet := r.sheets[ts]
	tw, th := ts.TileWidth, ts.TileHeight
	// Flip matrices are orthogonal, so the inverse is the transpose: map
	// each destination pixel back to the source pixel it shows.
	f := ref.Flags.Transform()
	inv := geom.Mat2{A: f.A, B: f.C, C: f.B, D: f.D}
	center := geom.Vec{X: float64(tw) / 2, Y: float64(th) / 2}
	flipped := image.NewNRGBA(image.Rect(0, 0, tw, th))
	for y := 0; y < th; y++ {
		for x := 0; x < tw; x++ {
			s := inv.Apply(geom.Vec{X: float64(x) + 0.5, Y: float64(y) + 0.5}.Sub(center)).Add(center)
			sx, sy := int(math.Floor(s.X)), int(math.Floor(s.Y))
			flipped.Set(x, y, sheet.At(src.Min.X+sx, src.Min.Y+sy))
		}
	}
	at := pos.Add(ts.TileOffset).Sub(r.origin)
	dr := image.Rect(int(at.X), int(at.Y)-th, int(at.X)+tw, int(at.Y))
	mask := image.NewUniform(color.Alpha{A: uint8(math.Round(255 * opacity))})
	draw.DrawMask(r.dst, dr, flipped, image.Point{}, mask, image.Point{}, draw.Over)
}

func (r *renderer) outline(b geom.Rect, c color.NRGBA) {
	b = b.Translate(r.origin.Scale(-1))
	x0, y0 := int(math.Floor(b.X)), int(math.Floor(b.Y))
	x1, y1 := int(math.Ceil(b.Right()))-1, int(math.Ceil(b.Bottom()))-1
	for x := x0; x <= x1; x++ {
		r.dst.SetNRGBA(x, y0, c)
		r.dst.SetNRGBA(x, y1, c)
	}
	for y := y0; y <= y1; y++ {
		r.dst.SetNRGBA(x0, y, c)
		r.dst.SetNRGBA(x1, y, c)
	}
}

func upscale(src *image.NRGBA, k int) image.Image {
	if k == 1 {
		return src
	}
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx()*k, b.Dy()*k))
	for y := 0; y < dst.Bounds().Dy(); y++ {
		for x := 0; x < dst.Bounds().Dx(); x++ {
			dst.SetNRGBA(x, y, src.NRGBAAt(x/k, y/k))
		}
	}
	return dst
}
