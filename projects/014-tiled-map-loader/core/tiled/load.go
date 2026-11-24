package tiled

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
)

// DefaultMaxLayerTiles is the default cap on width*height of a tile layer.
const DefaultMaxLayerTiles = 1 << 22

// Options configures LoadWithOptions.
type Options struct {
	// MaxLayerTiles caps the cell count of any one tile layer (or stitched
	// infinite layer), so a corrupt or hostile file cannot make the loader
	// allocate gigabytes. Zero means DefaultMaxLayerTiles.
	MaxLayerTiles int
}

// Load reads the .tmj map at name from fsys, along with any external
// tilesets it references, and validates the result.
//
// Paths inside the map are resolved relative to the map's directory. Because
// all reads go through fsys, a map can never reach files outside it: fs.FS
// rejects paths that climb above its root.
func Load(fsys fs.FS, name string) (*Map, error) {
	return LoadWithOptions(fsys, name, Options{})
}

// LoadWithOptions is Load with explicit limits.
func LoadWithOptions(fsys fs.FS, name string, opts Options) (*Map, error) {
	if opts.MaxLayerTiles <= 0 {
		opts.MaxLayerTiles = DefaultMaxLayerTiles
	}
	l := &loader{fsys: fsys, opts: opts, dir: path.Dir(name), external: map[string]*rawTileset{}}
	m, err := l.loadMap(name)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", name, err)
	}
	return m, nil
}

type loader struct {
	fsys     fs.FS
	opts     Options
	dir      string                 // directory of the map file
	external map[string]*rawTileset // external tilesets parsed so far, by path
}

func (l *loader) readJSON(name string, v any) error {
	switch strings.ToLower(path.Ext(name)) {
	case ".tmx", ".tsx", ".tx", ".xml":
		return fmt.Errorf("%w: XML file %s (export as JSON: .tmj/.tsj)", ErrUnsupported, name)
	}
	data, err := fs.ReadFile(l.fsys, name)
	if err != nil {
		return err
	}
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '<' {
		return fmt.Errorf("%w: %s is XML (export as JSON: .tmj/.tsj)", ErrUnsupported, name)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrInvalid, name, err)
	}
	return nil
}

func (l *loader) loadMap(name string) (*Map, error) {
	var rm rawMap
	if err := l.readJSON(name, &rm); err != nil {
		return nil, err
	}
	if rm.Type != "" && rm.Type != "map" {
		return nil, fmt.Errorf("%w: type is %q, not \"map\"", ErrInvalid, rm.Type)
	}
	switch rm.Orientation {
	case "orthogonal":
	case "":
		return nil, fmt.Errorf("%w: missing orientation", ErrInvalid)
	default:
		return nil, fmt.Errorf("%w: %s orientation", ErrUnsupported, rm.Orientation)
	}
	if rm.TileWidth <= 0 || rm.TileHeight <= 0 {
		return nil, fmt.Errorf("%w: tile size %dx%d", ErrInvalid, rm.TileWidth, rm.TileHeight)
	}
	if !rm.Infinite && (rm.Width <= 0 || rm.Height <= 0) {
		return nil, fmt.Errorf("%w: map size %dx%d", ErrInvalid, rm.Width, rm.Height)
	}

	m := &Map{
		Width: rm.Width, Height: rm.Height,
		TileWidth: rm.TileWidth, TileHeight: rm.TileHeight,
		Infinite:    rm.Infinite,
		RenderOrder: rm.RenderOrder,
		Class:       rm.Class,
	}
	if m.RenderOrder == "" {
		m.RenderOrder = "right-down"
	}
	var err error
	if m.Background, m.HasBackground, err = parseOptionalColor(rm.BackgroundColor); err != nil {
		return nil, fmt.Errorf("backgroundcolor: %w", err)
	}
	if m.Properties, err = parseProperties(rm.Properties); err != nil {
		return nil, err
	}
	for i, rts := range rm.Tilesets {
		ts, err := l.tileset(rts)
		if err != nil {
			return nil, fmt.Errorf("tileset #%d %q: %w", i, firstNonEmpty(rts.Name, rts.Source), err)
		}
		m.Tilesets = append(m.Tilesets, ts)
	}
	if err := checkGIDRanges(m.Tilesets); err != nil {
		return nil, err
	}
	if m.Layers, err = l.layers(rm.Layers, m); err != nil {
		return nil, err
	}
	return m, nil
}

// checkGIDRanges sorts tilesets by FirstGID and rejects overlapping ranges,
// which would make GID resolution ambiguous.
func checkGIDRanges(sets []*Tileset) error {
	sort.Slice(sets, func(i, j int) bool { return sets[i].FirstGID < sets[j].FirstGID })
	for i := 1; i < len(sets); i++ {
		prev, cur := sets[i-1], sets[i]
		if prev.FirstGID+uint32(prev.TileCount) > cur.FirstGID {
			return fmt.Errorf("%w: tilesets %q and %q have overlapping GID ranges", ErrInvalid, prev.Name, cur.Name)
		}
	}
	return nil
}

func (l *loader) tileset(rts rawTileset) (*Tileset, error) {
	if rts.FirstGID == 0 {
		return nil, fmt.Errorf("%w: firstgid must be at least 1", ErrInvalid)
	}
	base := l.dir
	if rts.Source != "" {
		src := path.Join(l.dir, rts.Source)
		ext, ok := l.external[src]
		if !ok {
			ext = &rawTileset{}
			if err := l.readJSON(src, ext); err != nil {
				return nil, err
			}
			l.external[src] = ext
		}
		firstGID := rts.FirstGID
		rts = *ext
		rts.FirstGID, rts.Source = firstGID, src
		base = path.Dir(src)
	}
	if rts.TileWidth <= 0 || rts.TileHeight <= 0 || rts.TileCount <= 0 {
		return nil, fmt.Errorf("%w: tile size %dx%d, count %d", ErrInvalid, rts.TileWidth, rts.TileHeight, rts.TileCount)
	}
	if rts.Image == "" {
		return nil, fmt.Errorf("%w: image-collection tilesets", ErrUnsupported)
	}
	if rts.Columns <= 0 {
		return nil, fmt.Errorf("%w: columns must be positive", ErrInvalid)
	}

	ts := &Tileset{
		FirstGID: rts.FirstGID, Source: rts.Source,
		Name: rts.Name, Class: rts.Class,
		TileWidth: rts.TileWidth, TileHeight: rts.TileHeight,
		Margin: rts.Margin, Spacing: rts.Spacing,
		TileCount: rts.TileCount, Columns: rts.Columns,
		Image:      path.Join(base, rts.Image),
		ImageWidth: rts.ImageWidth, ImageHeight: rts.ImageHeight,
		Tiles: make(map[int]*Tile, len(rts.Tiles)),
	}
	if rts.TileOffset != nil {
		ts.TileOffset = *rts.TileOffset
	}
	// Every tile must lie fully inside the image, or renderers would sample
	// garbage (or panic) when cutting it out.
	if last := ts.SourceRect(ts.TileCount - 1); ts.ImageWidth > 0 && !last.In(image.Rect(0, 0, ts.ImageWidth, ts.ImageHeight)) {
		return nil, fmt.Errorf("%w: %d tiles do not fit in a %dx%d image", ErrInvalid, ts.TileCount, ts.ImageWidth, ts.ImageHeight)
	}
	var err error
	if ts.Properties, err = parseProperties(rts.Properties); err != nil {
		return nil, err
	}
	for _, rt := range rts.Tiles {
		tile, err := parseTile(rt, ts.TileCount)
		if err != nil {
			return nil, fmt.Errorf("tile %d: %w", rt.ID, err)
		}
		ts.Tiles[tile.ID] = tile
	}
	return ts, nil
}

func parseTile(rt rawTile, tileCount int) (*Tile, error) {
	if rt.ID < 0 || rt.ID >= tileCount {
		return nil, fmt.Errorf("%w: id outside 0..%d", ErrInvalid, tileCount-1)
	}
	tile := &Tile{ID: rt.ID, Class: firstNonEmpty(rt.Class, rt.Type), Probability: floatOr(rt.Probability, 1)}
	var err error
	if tile.Properties, err = parseProperties(rt.Properties); err != nil {
		return nil, err
	}
	if rt.ObjectGroup != nil {
		for _, ro := range rt.ObjectGroup.Objects {
			o, err := parseObject(ro, nil)
			if err != nil {
				return nil, fmt.Errorf("collision object %d: %w", ro.ID, err)
			}
			tile.Collision = append(tile.Collision, o)
		}
	}
	for _, f := range rt.Animation {
		if f.TileID < 0 || f.TileID >= tileCount || f.Duration < 0 {
			return nil, fmt.Errorf("%w: animation frame (tile %d, %d ms)", ErrInvalid, f.TileID, f.Duration)
		}
		tile.Animation = append(tile.Animation, Frame{TileID: f.TileID, Duration: time.Duration(f.Duration) * time.Millisecond})
	}
	return tile, nil
}

// parseObject converts a raw object. m resolves tile-object GIDs; it is nil
// inside a tileset's collision editor, where tile objects are not allowed.
func parseObject(ro rawObject, m *Map) (Object, error) {
	if ro.Template != "" {
		return Object{}, fmt.Errorf("%w: object templates (detach %s in Tiled before exporting)", ErrUnsupported, ro.Template)
	}
	o := Object{
		ID: ro.ID, Name: ro.Name, Class: firstNonEmpty(ro.Class, ro.Type),
		X: ro.X, Y: ro.Y, Width: ro.Width, Height: ro.Height, Rotation: ro.Rotation,
		Visible: boolOr(ro.Visible, true),
	}
	switch {
	case ro.GID != 0:
		o.Shape, o.GID = ShapeTile, GID(ro.GID)
		if m == nil {
			return Object{}, fmt.Errorf("%w: tile object not allowed here", ErrInvalid)
		}
		if _, ok := m.ResolveGID(o.GID); !ok {
			return Object{}, fmt.Errorf("%w: gid %d matches no tileset", ErrInvalid, o.GID.ID())
		}
	case ro.Point:
		o.Shape = ShapePoint
	case ro.Ellipse:
		o.Shape = ShapeEllipse
	case ro.Polygon != nil:
		o.Shape, o.Points = ShapePolygon, ro.Polygon
	case ro.Polyline != nil:
		o.Shape, o.Points = ShapePolyline, ro.Polyline
	case ro.Text != nil:
		o.Shape, o.Text = ShapeText, ro.Text.Text
	default:
		o.Shape = ShapeRectangle
	}
	var err error
	o.Properties, err = parseProperties(ro.Properties)
	return o, err
}

func (l *loader) layers(raws []rawLayer, m *Map) ([]Layer, error) {
	out := make([]Layer, 0, len(raws))
	for _, rl := range raws {
		layer, err := l.layer(rl, m)
		if err != nil {
			return nil, fmt.Errorf("layer %q: %w", rl.Name, err)
		}
		out = append(out, layer)
	}
	return out, nil
}

func (l *loader) layer(rl rawLayer, m *Map) (Layer, error) {
	info := LayerInfo{
		ID: rl.ID, Name: rl.Name, Class: rl.Class,
		Visible:  boolOr(rl.Visible, true),
		Opacity:  floatOr(rl.Opacity, 1),
		Offset:   geom.Vec{X: rl.OffsetX, Y: rl.OffsetY},
		Parallax: geom.Vec{X: floatOr(rl.ParallaxX, 1), Y: floatOr(rl.ParallaxY, 1)},
	}
	var err error
	if info.Tint, info.HasTint, err = parseOptionalColor(rl.TintColor); err != nil {
		return nil, fmt.Errorf("tintcolor: %w", err)
	}
	if info.Properties, err = parseProperties(rl.Properties); err != nil {
		return nil, err
	}

	switch rl.Type {
	case "tilelayer":
		return l.tileLayer(rl, info, m)
	case "objectgroup":
		og := &ObjectGroup{LayerInfo: info, DrawOrder: rl.DrawOrder}
		if og.DrawOrder == "" {
			og.DrawOrder = "topdown"
		}
		if og.Color, _, err = parseOptionalColor(rl.Color); err != nil {
			return nil, fmt.Errorf("color: %w", err)
		}
		for _, ro := range rl.Objects {
			o, err := parseObject(ro, m)
			if err != nil {
				return nil, fmt.Errorf("object %d %q: %w", ro.ID, ro.Name, err)
			}
			og.Objects = append(og.Objects, o)
		}
		return og, nil
	case "imagelayer":
		img := &ImageLayer{LayerInfo: info, RepeatX: rl.RepeatX, RepeatY: rl.RepeatY}
		if rl.Image != "" {
			img.Image = path.Join(l.dir, rl.Image)
		}
		return img, nil
	case "group":
		children, err := l.layers(rl.Layers, m)
		if err != nil {
			return nil, err
		}
		return &GroupLayer{LayerInfo: info, Layers: children}, nil
	default:
		return nil, fmt.Errorf("%w: unknown layer type %q", ErrInvalid, rl.Type)
	}
}

func (l *loader) cellCount(w, h int) (int, error) {
	if w <= 0 || h <= 0 {
		return 0, fmt.Errorf("%w: layer size %dx%d", ErrInvalid, w, h)
	}
	if w > l.opts.MaxLayerTiles/h {
		return 0, fmt.Errorf("%w: layer size %dx%d exceeds the %d-tile limit", ErrInvalid, w, h, l.opts.MaxLayerTiles)
	}
	return w * h, nil
}

func (l *loader) tileLayer(rl rawLayer, info LayerInfo, m *Map) (*TileLayer, error) {
	tl := &TileLayer{LayerInfo: info}
	if m.Infinite {
		if err := l.stitchChunks(tl, rl); err != nil {
			return nil, err
		}
	} else {
		n, err := l.cellCount(rl.Width, rl.Height)
		if err != nil {
			return nil, err
		}
		if tl.Tiles, err = decodeTileData(rl.Data, rl.Encoding, rl.Compression, n); err != nil {
			return nil, err
		}
		tl.Width, tl.Height = rl.Width, rl.Height
	}
	for i, g := range tl.Tiles {
		if g.IsEmpty() {
			continue
		}
		if _, ok := m.ResolveGID(g); !ok {
			x, y := tl.StartX+i%tl.Width, tl.StartY+i/tl.Width
			return nil, fmt.Errorf("%w: tile (%d,%d) has gid %d, which no tileset owns", ErrInvalid, x, y, g.ID())
		}
	}
	return tl, nil
}

// stitchChunks assembles an infinite layer's chunks into one dense grid
// covering their bounding box. Games usually stream chunks; for a loader
// whose output feeds collision and rendering, one grid is simpler and the
// size limit keeps it bounded.
func (l *loader) stitchChunks(tl *TileLayer, rl rawLayer) error {
	if len(rl.Chunks) == 0 {
		return nil
	}
	var bounds image.Rectangle
	for _, ch := range rl.Chunks {
		bounds = bounds.Union(image.Rect(ch.X, ch.Y, ch.X+ch.Width, ch.Y+ch.Height))
	}
	n, err := l.cellCount(bounds.Dx(), bounds.Dy())
	if err != nil {
		return err
	}
	tl.StartX, tl.StartY, tl.Width, tl.Height = bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy()
	tl.Tiles = make([]GID, n)
	for _, ch := range rl.Chunks {
		cn, err := l.cellCount(ch.Width, ch.Height)
		if err != nil {
			return fmt.Errorf("chunk (%d,%d): %w", ch.X, ch.Y, err)
		}
		gids, err := decodeTileData(ch.Data, rl.Encoding, rl.Compression, cn)
		if err != nil {
			return fmt.Errorf("chunk (%d,%d): %w", ch.X, ch.Y, err)
		}
		for row := 0; row < ch.Height; row++ {
			dst := (ch.Y-tl.StartY+row)*tl.Width + (ch.X - tl.StartX)
			copy(tl.Tiles[dst:dst+ch.Width], gids[row*ch.Width:(row+1)*ch.Width])
		}
	}
	return nil
}
