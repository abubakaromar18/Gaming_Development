package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png" // tileset images
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/tiled"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/world"
)

// The logical screen is small and scaled up, so 16px tiles stay crisp.
const screenW, screenH = 320, 240

var (
	playerFill    = color.NRGBA{250, 226, 120, 255}
	playerEdge    = color.NRGBA{60, 40, 20, 255}
	colliderColor = color.NRGBA{255, 60, 60, 220}
	entityColor   = color.NRGBA{255, 230, 80, 220}
	hudShade      = color.NRGBA{0, 0, 0, 150}
)

type game struct {
	explorer      *world.Explorer
	sheets        map[*tiled.Tileset]*ebiten.Image
	camera        geom.Vec
	showCollision bool
	title         string
}

func newGame(mapPath string) (*game, error) {
	fsys := os.DirFS(filepath.Dir(mapPath))
	m, err := tiled.Load(fsys, filepath.Base(mapPath))
	if err != nil {
		return nil, err
	}
	w, err := world.New(m, world.DefaultConfig())
	if err != nil {
		return nil, err
	}
	ex, err := world.NewExplorer(w, world.DefaultExplorerConfig())
	if err != nil {
		return nil, err
	}
	sheets, err := loadSheets(fsys, m)
	if err != nil {
		return nil, err
	}
	return &game{explorer: ex, sheets: sheets, title: m.Properties.String("title", filepath.Base(mapPath))}, nil
}

// loadSheets decodes each tileset image once and uploads it to the GPU.
// Individual tiles are later cut out with SubImage, which shares the texture.
func loadSheets(fsys fs.FS, m *tiled.Map) (map[*tiled.Tileset]*ebiten.Image, error) {
	sheets := make(map[*tiled.Tileset]*ebiten.Image, len(m.Tilesets))
	for _, ts := range m.Tilesets {
		f, err := fsys.Open(ts.Image)
		if err != nil {
			return nil, fmt.Errorf("tileset %q: %w", ts.Name, err)
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("tileset %q image %s: %w", ts.Name, ts.Image, err)
		}
		sheets[ts] = ebiten.NewImageFromImage(img)
	}
	return sheets, nil
}

func (g *game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.showCollision = !g.showCollision
	}
	var intent geom.Vec
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		intent.X--
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		intent.X++
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		intent.Y--
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		intent.Y++
	}
	// Ebitengine calls Update at a fixed tick rate, so dt is constant.
	g.explorer.Update(intent, 1/float64(ebiten.TPS()))
	g.camera = world.Follow(g.explorer.Player.Box.Center(), geom.Vec{X: screenW, Y: screenH}, g.explorer.World.Collider.PixelBounds())
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	m := g.explorer.World.Map
	if m.HasBackground {
		screen.Fill(m.Background)
	}
	for _, l := range m.Flatten() {
		if !l.Visible {
			continue
		}
		switch layer := l.Layer.(type) {
		case *tiled.TileLayer:
			g.drawTileLayer(screen, layer, l)
		case *tiled.ObjectGroup:
			for i := range layer.Objects {
				if o := &layer.Objects[i]; o.Shape == tiled.ShapeTile && o.Visible {
					g.drawTile(screen, o.GID, geom.Vec{X: o.X, Y: o.Y}, l)
				}
			}
		}
		// Image layers are not used by the demo map and are skipped.
	}

	p := g.toScreen(g.explorer.Player.Box)
	vector.DrawFilledRect(screen, p[0], p[1], p[2], p[3], playerFill, false)
	vector.StrokeRect(screen, p[0], p[1], p[2], p[3], 1, playerEdge, false)
	if g.showCollision {
		for _, b := range g.explorer.World.Collider.Boxes() {
			r := g.toScreen(b)
			vector.StrokeRect(screen, r[0], r[1], r[2], r[3], 1, colliderColor, false)
		}
		for _, e := range g.explorer.World.Entities {
			r := g.toScreen(e.Bounds)
			vector.StrokeRect(screen, r[0], r[1], max(r[2], 2), max(r[3], 2), 1, entityColor, false)
		}
	}
	g.drawHUD(screen)
}

// drawTileLayer draws only the tiles inside the camera view. Culling keeps
// the cost proportional to the screen, not the map.
func (g *game) drawTileLayer(screen *ebiten.Image, l *tiled.TileLayer, r tiled.ResolvedLayer) {
	m := g.explorer.World.Map
	tw, th := float64(m.TileWidth), float64(m.TileHeight)
	view := g.camera.Sub(r.Offset)
	view.X *= r.Parallax.X
	view.Y *= r.Parallax.Y
	// One extra tile on each side covers partial tiles; tiles taller than
	// the grid extend upward, hence the extra rows below the view.
	x0, x1 := int(math.Floor(view.X/tw))-1, int(math.Ceil((view.X+screenW)/tw))+1
	y0, y1 := int(math.Floor(view.Y/th))-1, int(math.Ceil((view.Y+screenH)/th))+2
	x0, y0 = max(x0, l.StartX), max(y0, l.StartY)
	x1, y1 = min(x1, l.StartX+l.Width), min(y1, l.StartY+l.Height)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			g.drawTile(screen, l.At(x, y), geom.Vec{X: float64(x) * tw, Y: float64(y+1) * th}, r)
		}
	}
}

// drawTile draws gid with its bottom-left corner at pos (map pixels),
// applying animation, flip flags, layer offset, parallax and opacity.
func (g *game) drawTile(screen *ebiten.Image, gid tiled.GID, pos geom.Vec, r tiled.ResolvedLayer) {
	ref, ok := g.explorer.World.Map.ResolveGID(gid)
	if !ok {
		return
	}
	ts := ref.Tileset
	src := ts.SourceRect(g.explorer.World.Animator.Frame(ts, ref.LocalID))
	tile := g.sheets[ts].SubImage(src).(*ebiten.Image)
	tw, th := float64(ts.TileWidth), float64(ts.TileHeight)

	var op ebiten.DrawImageOptions
	if f := ref.Flags.Transform(); f != geom.Identity {
		// Flip around the tile's center with the core's 2x2 matrix.
		var flip ebiten.GeoM
		flip.SetElement(0, 0, f.A)
		flip.SetElement(0, 1, f.B)
		flip.SetElement(1, 0, f.C)
		flip.SetElement(1, 1, f.D)
		op.GeoM.Translate(-tw/2, -th/2)
		op.GeoM.Concat(flip)
		op.GeoM.Translate(tw/2, th/2)
	}
	at := pos.Add(r.Offset).Add(ts.TileOffset)
	at.X -= g.camera.X * r.Parallax.X
	at.Y -= g.camera.Y * r.Parallax.Y
	op.GeoM.Translate(math.Round(at.X), math.Round(at.Y-th))
	op.ColorScale.ScaleAlpha(float32(r.Opacity))
	screen.DrawImage(tile, &op)
}

func (g *game) toScreen(b geom.Rect) [4]float32 {
	return [4]float32{
		float32(math.Round(b.X - g.camera.X)), float32(math.Round(b.Y - g.camera.Y)),
		float32(b.W), float32(b.H),
	}
}

func (g *game) drawHUD(screen *ebiten.Image) {
	ex := g.explorer
	status := fmt.Sprintf("%s  gold %d", g.title, ex.Gold)
	if len(ex.Inventory) > 0 {
		status += "  [" + strings.Join(ex.Inventory, ", ") + "]"
	}
	vector.DrawFilledRect(screen, 0, 0, screenW, 16, hudShade, false)
	ebitenutil.DebugPrintAt(screen, status, 4, 0)
	if ex.Message != "" {
		vector.DrawFilledRect(screen, 0, screenH-18, screenW, 18, hudShade, false)
		ebitenutil.DebugPrintAt(screen, ex.Message, 4, screenH-17)
	} else {
		ebitenutil.DebugPrintAt(screen, "WASD/arrows move  C collision  Esc quit", 4, screenH-17)
	}
}

func (g *game) Layout(int, int) (int, int) { return screenW, screenH }
