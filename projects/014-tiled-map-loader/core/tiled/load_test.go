package tiled_test

import (
	"errors"
	"io/fs"
	"math"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/tiled"
)

func load(t *testing.T) *tiled.Map {
	t.Helper()
	m, err := tiled.Load(fsWith(t, fixture(t)), "maps/level.tmj")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return m
}

func TestLoadMapHeader(t *testing.T) {
	m := load(t)
	if m.Width != 4 || m.Height != 3 || m.TileWidth != 16 || m.TileHeight != 16 {
		t.Fatalf("size = %dx%d @ %dx%d", m.Width, m.Height, m.TileWidth, m.TileHeight)
	}
	if !m.HasBackground || m.Background != (tiled.Color{R: 0x33, G: 0x66, B: 0x99, A: 0xff}) {
		t.Errorf("background = %+v (has %t)", m.Background, m.HasBackground)
	}
	if m.RenderOrder != "right-down" {
		t.Errorf("render order = %q", m.RenderOrder)
	}
}

func TestTypedProperties(t *testing.T) {
	p := load(t).Properties
	if got := p.String("title", ""); got != "Fixture" {
		t.Errorf("title = %q", got)
	}
	if got := p.Float("gravity", 0); got != 9.5 {
		t.Errorf("gravity = %v", got)
	}
	if got := p.Float("lives", 0); got != 3 {
		t.Errorf("int widened to float = %v", got)
	}
	if got := p.Int("lives", 0); got != 3 {
		t.Errorf("lives = %v", got)
	}
	if !p.Bool("hard", false) {
		t.Error("hard = false")
	}
	if c, ok := p.Color("tint"); !ok || c != (tiled.Color{R: 255, A: 128}) {
		t.Errorf("tint = %+v, %t", c, ok)
	}
	if got := p.String("music", ""); got != "theme.ogg" {
		t.Errorf("file property = %q", got)
	}
	if id, ok := p.ObjectRef("boss"); !ok || id != 7 {
		t.Errorf("boss = %d, %t", id, ok)
	}
	stats, _ := p.Lookup("stats")
	if stats.CustomType != "Stats" || stats.Value.(map[string]any)["hp"] != 10.0 {
		t.Errorf("class property = %+v", stats)
	}
	// Wrong type or missing name falls back rather than guessing.
	if got := p.Int("title", -1); got != -1 {
		t.Errorf("string read as int = %d", got)
	}
	if got := p.String("nope", "def"); got != "def" {
		t.Errorf("missing = %q", got)
	}
}

func TestTilesetsAreSortedAndResolvedRelativeToTheirFile(t *testing.T) {
	m := load(t)
	if len(m.Tilesets) != 2 || m.Tilesets[0].Name != "terrain" || m.Tilesets[1].Name != "items" {
		t.Fatalf("tilesets not sorted by firstgid: %v, %v", m.Tilesets[0].Name, m.Tilesets[1].Name)
	}
	terrain, items := m.Tilesets[0], m.Tilesets[1]
	if terrain.Source != "maps/sets/terrain.tsj" || terrain.Image != "maps/img/terrain.png" {
		t.Errorf("external paths: source %q image %q", terrain.Source, terrain.Image)
	}
	if items.Source != "" || items.Image != "maps/items.png" {
		t.Errorf("embedded paths: source %q image %q", items.Source, items.Image)
	}
	if got := terrain.Tiles[1].Class; got != "wall" {
		t.Errorf("legacy \"type\" not read as class: %q", got)
	}
	if got := terrain.Tiles[2].Class; got != "water" {
		t.Errorf("class = %q", got)
	}
	if got := terrain.Tiles[3].Probability; got != 0.25 {
		t.Errorf("probability = %v", got)
	}
	if got := terrain.Tiles[1].Probability; got != 1 {
		t.Errorf("default probability = %v", got)
	}
}

func TestSourceRectHonorsMarginAndSpacing(t *testing.T) {
	ts := load(t).Tilesets[0]
	if got := ts.SourceRect(0); got.Min.X != 1 || got.Min.Y != 1 || got.Dx() != 16 {
		t.Errorf("tile 0 = %v", got)
	}
	if got := ts.SourceRect(3); got.Min.X != 19 || got.Min.Y != 19 || got.Max.X != 35 {
		t.Errorf("tile 3 = %v", got)
	}
}

func TestTileAnimationAndCollisionMetadata(t *testing.T) {
	ts := load(t).Tilesets[0]
	anim := ts.Tiles[2].Animation
	if len(anim) != 2 || anim[0].Duration != 100*time.Millisecond || anim[1].TileID != 3 {
		t.Errorf("animation = %+v", anim)
	}
	col := ts.Tiles[3].Collision
	if len(col) != 1 || col[0].Bounds() != (geom.Rect{X: 2, Y: 4, W: 8, H: 12}) {
		t.Errorf("collision = %+v", col)
	}
}

func TestResolveGID(t *testing.T) {
	m := load(t)
	ref, ok := m.ResolveGID(2 | tiled.FlipHorizontal)
	if !ok || ref.Tileset.Name != "terrain" || ref.LocalID != 1 || ref.Tile.Class != "wall" || ref.Flags != tiled.FlipHorizontal {
		t.Errorf("gid 2|H = %+v, %t", ref, ok)
	}
	ref, ok = m.ResolveGID(11)
	if !ok || ref.Tileset.Name != "items" || ref.LocalID != 1 || ref.Tile != nil {
		t.Errorf("gid 11 = %+v, %t", ref, ok)
	}
	for _, g := range []tiled.GID{0, 5, 9, 12, tiled.FlipVertical} {
		if _, ok := m.ResolveGID(g); ok {
			t.Errorf("gid %#x resolved, want none", uint32(g))
		}
	}
}

func TestFlattenCombinesGroupAttributes(t *testing.T) {
	flat := load(t).Flatten()
	if len(flat) != 3 {
		t.Fatalf("got %d leaf layers", len(flat))
	}
	ground, decor, things := flat[0], flat[1], flat[2]
	if ground.Path != "world/ground" || decor.Path != "world/decor" || things.Path != "things" {
		t.Errorf("paths = %q %q %q", ground.Path, decor.Path, things.Path)
	}
	if ground.Opacity != 0.25 || ground.Offset != (geom.Vec{X: 5, Y: 2}) || ground.Parallax != (geom.Vec{X: 0.5, Y: 1}) {
		t.Errorf("ground resolved = %+v", ground)
	}
	if decor.Visible || !ground.Visible {
		t.Errorf("visibility: ground %t decor %t", ground.Visible, decor.Visible)
	}
}

func TestTileLayerData(t *testing.T) {
	m := load(t)
	ground, ok := m.TileLayer("world/ground")
	if !ok {
		t.Fatal("ground not found by path")
	}
	if g := ground.At(2, 1); g.ID() != 2 || g.Flags() != tiled.FlipHorizontal {
		t.Errorf("At(2,1) = %#x", uint32(g))
	}
	if g := ground.At(4, 0); g != 0 {
		t.Errorf("out of range = %#x", uint32(g))
	}
	decor, ok := m.TileLayer("decor")
	if !ok {
		t.Fatal("decor not found by name")
	}
	for i, want := range decorGIDs {
		if uint32(decor.Tiles[i]) != want {
			t.Fatalf("base64+zlib tile %d = %#x, want %#x", i, uint32(decor.Tiles[i]), want)
		}
	}
}

func TestObjects(t *testing.T) {
	og, ok := load(t).ObjectGroup("things")
	if !ok {
		t.Fatal("things not found")
	}
	if og.DrawOrder != "topdown" {
		t.Errorf("default draw order = %q", og.DrawOrder)
	}
	want := []struct {
		name   string
		class  string
		shape  tiled.Shape
		bounds geom.Rect
	}{
		{"start", "spawn", tiled.ShapePoint, geom.Rect{X: 8, Y: 8}},
		// Tile objects are anchored bottom-left.
		{"coin", "pickup", tiled.ShapeTile, geom.Rect{X: 16, Y: 16, W: 16, H: 16}},
		// Rotated 90° clockwise around the top-left corner.
		{"zone", "trigger", tiled.ShapeRectangle, geom.Rect{X: -16, Y: 0, W: 16, H: 32}},
		{"hill", "", tiled.ShapePolygon, geom.Rect{X: 10, Y: 5, W: 20, H: 20}},
		{"rope", "", tiled.ShapePolyline, geom.Rect{X: 0, Y: 0, W: 3, H: 4}},
		{"pond", "", tiled.ShapeEllipse, geom.Rect{X: 2, Y: 2, W: 4, H: 6}},
		{"label", "", tiled.ShapeText, geom.Rect{X: 1, Y: 1, W: 30, H: 10}},
	}
	if len(og.Objects) != len(want) {
		t.Fatalf("got %d objects", len(og.Objects))
	}
	for i, w := range want {
		o := &og.Objects[i]
		b := o.Bounds()
		if o.Name != w.name || o.Class != w.class || o.Shape != w.shape || !approxRect(b, w.bounds) {
			t.Errorf("object %d = %s/%s/%s %+v, want %s/%s/%s %+v", i, o.Name, o.Class, o.Shape, b, w.name, w.class, w.shape, w.bounds)
		}
	}
	if og.Objects[1].Properties.Int("value", 0) != 5 || og.Objects[1].GID != 10 {
		t.Errorf("coin = %+v", og.Objects[1])
	}
	if og.Objects[6].Text != "Hello" || og.Objects[6].Visible {
		t.Errorf("label = %+v", og.Objects[6])
	}
}

// approxRect compares with a tolerance: rotation goes through sin/cos.
func approxRect(a, b geom.Rect) bool {
	near := func(x, y float64) bool { return math.Abs(x-y) < 1e-9 }
	return near(a.X, b.X) && near(a.Y, b.Y) && near(a.W, b.W) && near(a.H, b.H)
}

func TestInfiniteMapChunksAreStitched(t *testing.T) {
	doc := map[string]any{
		"type": "map", "orientation": "orthogonal", "infinite": true,
		"width": 0, "height": 0, "tilewidth": 8, "tileheight": 8,
		"tilesets": []any{map[string]any{"firstgid": 1, "name": "t", "tilewidth": 8, "tileheight": 8,
			"tilecount": 4, "columns": 2, "image": "t.png", "imagewidth": 16, "imageheight": 16}},
		"layers": []any{map[string]any{"name": "l", "type": "tilelayer", "chunks": []any{
			map[string]any{"x": -2, "y": 0, "width": 2, "height": 1, "data": []int{1, 2}},
			map[string]any{"x": 0, "y": 1, "width": 2, "height": 1, "data": []int{3, 4}},
		}}},
	}
	fsys := fsWith(t, doc)
	m, err := tiled.Load(fsys, "maps/level.tmj")
	if err != nil {
		t.Fatal(err)
	}
	l, _ := m.TileLayer("l")
	if l.StartX != -2 || l.StartY != 0 || l.Width != 4 || l.Height != 2 {
		t.Fatalf("stitched layer = start (%d,%d) size %dx%d", l.StartX, l.StartY, l.Width, l.Height)
	}
	checks := map[[2]int]uint32{{-2, 0}: 1, {-1, 0}: 2, {0, 1}: 3, {1, 1}: 4, {0, 0}: 0, {-2, 1}: 0}
	for p, want := range checks {
		if got := l.At(p[0], p[1]).ID(); got != want {
			t.Errorf("At%v = %d, want %d", p, got, want)
		}
	}
	if b := m.TileBounds(); b.Min.X != -2 || b.Max.X != 2 || b.Max.Y != 2 {
		t.Errorf("TileBounds = %v", b)
	}
}

func TestLoadErrors(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(doc map[string]any, fsys fstest.MapFS)
		want   error
		substr string
	}{
		{"isometric", func(d map[string]any, _ fstest.MapFS) { d["orientation"] = "isometric" }, tiled.ErrUnsupported, "isometric"},
		{"no orientation", func(d map[string]any, _ fstest.MapFS) { delete(d, "orientation") }, tiled.ErrInvalid, "orientation"},
		{"zero tile size", func(d map[string]any, _ fstest.MapFS) { d["tilewidth"] = 0 }, tiled.ErrInvalid, "tile size"},
		{"gid owned by no tileset", func(d map[string]any, _ fstest.MapFS) {
			groundLayer(d)["data"] = []int{1, 1, 1, 1, 1, 7, 1, 1, 1, 1, 1, 1}
		}, tiled.ErrInvalid, `layer "world": layer "ground": tiled: invalid map data: tile (1,1) has gid 7`},
		{"short csv", func(d map[string]any, _ fstest.MapFS) { groundLayer(d)["data"] = []int{1} }, tiled.ErrInvalid, "1 entries, want 12"},
		{"zstd", func(d map[string]any, _ fstest.MapFS) {
			g := groundLayer(d)
			g["encoding"], g["compression"], g["data"] = "base64", "zstd", "AAAA"
		}, tiled.ErrUnsupported, "zstd"},
		{"corrupt zlib", func(d map[string]any, _ fstest.MapFS) {
			g := groundLayer(d)
			g["encoding"], g["compression"], g["data"] = "base64", "zlib", "AAAAAAAA"
		}, tiled.ErrInvalid, "zlib"},
		{"overlapping tilesets", func(d map[string]any, _ fstest.MapFS) {
			d["tilesets"].([]any)[0].(map[string]any)["firstgid"] = 3
		}, tiled.ErrInvalid, "overlapping"},
		{"tsx tileset", func(d map[string]any, _ fstest.MapFS) {
			d["tilesets"].([]any)[1].(map[string]any)["source"] = "sets/terrain.tsx"
		}, tiled.ErrUnsupported, "XML"},
		{"template", func(d map[string]any, _ fstest.MapFS) {
			objs := layers(d)[1]["objects"].([]any)
			objs[0].(map[string]any)["template"] = "enemy.tx"
		}, tiled.ErrUnsupported, `object 1 "start"`},
		{"bad property value", func(d map[string]any, _ fstest.MapFS) {
			d["properties"].([]any)[2].(map[string]any)["value"] = "three"
		}, tiled.ErrInvalid, `property "lives"`},
		{"bad color", func(d map[string]any, _ fstest.MapFS) { d["backgroundcolor"] = "#12" }, tiled.ErrInvalid, "backgroundcolor"},
		{"tiles exceed image", func(_ map[string]any, f fstest.MapFS) {
			f["maps/sets/terrain.tsj"] = &fstest.MapFile{Data: []byte(strings.Replace(terrainTileset, `"imagewidth": 36`, `"imagewidth": 30`, 1))}
		}, tiled.ErrInvalid, "do not fit"},
		{"unknown layer type", func(d map[string]any, _ fstest.MapFS) { layers(d)[1]["type"] = "mystery" }, tiled.ErrInvalid, "mystery"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := fixture(t)
			fsys := fsWith(t, doc)
			tc.mutate(doc, fsys)
			// Re-encode the mutated document over the original.
			fsys["maps/level.tmj"] = fsWith(t, doc)["maps/level.tmj"]
			_, err := tiled.Load(fsys, "maps/level.tmj")
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if !strings.Contains(err.Error(), tc.substr) {
				t.Errorf("err %q does not mention %q", err, tc.substr)
			}
		})
	}
}

func TestMissingExternalTilesetWrapsNotExist(t *testing.T) {
	fsys := fsWith(t, fixture(t))
	delete(fsys, "maps/sets/terrain.tsj")
	_, err := tiled.Load(fsys, "maps/level.tmj")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v, want fs.ErrNotExist", err)
	}
}

func TestPathsCannotEscapeTheFilesystem(t *testing.T) {
	doc := fixture(t)
	doc["tilesets"].([]any)[1].(map[string]any)["source"] = "../../../etc/terrain.tsj"
	_, err := tiled.Load(fsWith(t, doc), "maps/level.tmj")
	if err == nil {
		t.Fatal("tileset outside the filesystem root was loaded")
	}
}

func TestXMLMapIsRejected(t *testing.T) {
	fsys := fstest.MapFS{
		"a.tmx":  {Data: []byte("<map/>")},
		"b.tmj":  {Data: []byte("  <?xml version=\"1.0\"?><map/>")},
		"c.json": {Data: []byte("{not json")},
	}
	for _, name := range []string{"a.tmx", "b.tmj"} {
		if _, err := tiled.Load(fsys, name); !errors.Is(err, tiled.ErrUnsupported) {
			t.Errorf("%s: err = %v, want ErrUnsupported", name, err)
		}
	}
	if _, err := tiled.Load(fsys, "c.json"); !errors.Is(err, tiled.ErrInvalid) {
		t.Errorf("malformed JSON: err = %v, want ErrInvalid", err)
	}
}

func TestLayerSizeLimit(t *testing.T) {
	_, err := tiled.LoadWithOptions(fsWith(t, fixture(t)), "maps/level.tmj", tiled.Options{MaxLayerTiles: 8})
	if !errors.Is(err, tiled.ErrInvalid) || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("err = %v, want size-limit error", err)
	}
}
