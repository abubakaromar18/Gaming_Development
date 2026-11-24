// Command mapascii loads a Tiled map and prints it as text: one glyph per
// cell taken from each tile's "glyph" property, with entities drawn on top,
// followed by a summary of what the loader and world builder found. It is a
// headless second renderer for the same core, useful in a terminal or CI.
//
//	go run ./cmd/mapascii maps/demo.tmj
package main

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/tiled"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/world"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: mapascii <map.tmj>")
		os.Exit(2)
	}
	if err := run(os.Stdout, os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "mapascii:", err)
		var code = 1
		if errors.Is(err, tiled.ErrUnsupported) {
			code = 3
		}
		os.Exit(code)
	}
}

func run(w io.Writer, file string) error {
	m, err := tiled.Load(os.DirFS(filepath.Dir(file)), filepath.Base(file))
	if err != nil {
		return err
	}
	wd, err := world.New(m, world.DefaultConfig())
	if err != nil {
		return err
	}
	b := m.TileBounds()
	grid := make([][]rune, b.Dy())
	for y := range grid {
		grid[y] = []rune(strings.Repeat(" ", b.Dx()))
	}
	// Later layers overwrite earlier ones, the same painter's order a
	// graphical renderer uses.
	for _, r := range m.Flatten() {
		tl, ok := r.Layer.(*tiled.TileLayer)
		if !ok || !r.Visible {
			continue
		}
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if g := glyphOf(m, tl.At(x, y)); g != 0 {
					grid[y-b.Min.Y][x-b.Min.X] = g
				}
			}
		}
	}
	for _, e := range wd.Entities {
		var g rune
		switch {
		case e.Class == "spawn":
			g = '@'
		case e.Shape == tiled.ShapeTile:
			g = glyphOf(m, e.GID)
		}
		p := e.Position()
		x, y := int(math.Floor(p.X/float64(m.TileWidth)))-b.Min.X, int(math.Floor(p.Y/float64(m.TileHeight)))-b.Min.Y
		if g != 0 && y >= 0 && y < len(grid) && x >= 0 && x < len(grid[y]) {
			grid[y][x] = g
		}
	}
	for _, row := range grid {
		fmt.Fprintln(w, string(row))
	}

	fmt.Fprintf(w, "\n%q: %dx%d tiles of %dx%d px\n", m.Properties.String("title", "untitled"), b.Dx(), b.Dy(), m.TileWidth, m.TileHeight)
	for _, ts := range m.Tilesets {
		where := "embedded"
		if ts.Source != "" {
			where = ts.Source
		}
		fmt.Fprintf(w, "tileset %-8s gids %d-%d (%s)\n", ts.Name, ts.FirstGID, ts.FirstGID+uint32(ts.TileCount)-1, where)
	}
	for _, r := range m.Flatten() {
		fmt.Fprintf(w, "layer   %-16s %-12s visible=%t\n", r.Path, kind(r.Layer), r.Visible)
	}
	fmt.Fprintf(w, "collision boxes: %d · entities: %d\n", len(wd.Collider.Boxes()), len(wd.Entities))
	return nil
}

func glyphOf(m *tiled.Map, g tiled.GID) rune {
	ref, ok := m.ResolveGID(g)
	if !ok || ref.Tile == nil {
		return 0
	}
	for _, r := range ref.Tile.Properties.String("glyph", "") {
		return r
	}
	return 0
}

func kind(l tiled.Layer) string {
	switch l.(type) {
	case *tiled.TileLayer:
		return "tiles"
	case *tiled.ObjectGroup:
		return "objects"
	case *tiled.ImageLayer:
		return "image"
	}
	return "group"
}
