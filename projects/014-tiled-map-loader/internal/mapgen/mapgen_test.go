package mapgen_test

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/tiled"
	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/internal/mapgen"
)

func TestGenerateIsDeterministic(t *testing.T) {
	a, err := mapgen.Generate(mapgen.DefaultSeed)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := mapgen.Generate(mapgen.DefaultSeed)
	for name := range a {
		if !bytes.Equal(a[name], b[name]) {
			t.Errorf("%s differs between runs", name)
		}
	}
	c, _ := mapgen.Generate(mapgen.DefaultSeed + 1)
	if bytes.Equal(a[mapgen.MapFile], c[mapgen.MapFile]) {
		t.Error("a different seed produced the same map")
	}
}

// TestCommittedMapIsUpToDate guards against editing the generator without
// regenerating maps/. It compares loaded models and decoded pixels rather
// than raw bytes, because zlib and PNG encoders may produce different (but
// equivalent) bytes across Go releases.
func TestCommittedMapIsUpToDate(t *testing.T) {
	files, err := mapgen.Generate(mapgen.DefaultSeed)
	if err != nil {
		t.Fatal(err)
	}
	generated := fstest.MapFS{}
	for name, data := range files {
		generated[name] = &fstest.MapFile{Data: data}
	}
	dir := filepath.Join("..", "..", "maps")
	want, err := tiled.Load(generated, mapgen.MapFile)
	if err != nil {
		t.Fatalf("generated map does not load: %v", err)
	}
	got, err := tiled.Load(os.DirFS(dir), mapgen.MapFile)
	if err != nil {
		t.Fatalf("committed map does not load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("maps/ is stale: run `go run ./cmd/genmap -out maps`")
	}
	for _, name := range []string{mapgen.TerrainImage, mapgen.MarkersImage} {
		committed, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		a, errA := png.Decode(bytes.NewReader(committed))
		b, errB := png.Decode(bytes.NewReader(files[name]))
		if errA != nil || errB != nil {
			t.Fatalf("%s: decode: %v / %v", name, errA, errB)
		}
		if !reflect.DeepEqual(a, b) {
			t.Errorf("%s is stale: run `go run ./cmd/genmap -out maps`", name)
		}
	}
}

func TestTilesetImagesMatchDeclaredSize(t *testing.T) {
	files, _ := mapgen.Generate(mapgen.DefaultSeed)
	fsys := fstest.MapFS{}
	for name, data := range files {
		fsys[name] = &fstest.MapFile{Data: data}
	}
	m, err := tiled.Load(fsys, mapgen.MapFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, ts := range m.Tilesets {
		img, err := png.Decode(bytes.NewReader(files[ts.Image]))
		if err != nil {
			t.Fatalf("%s: %v", ts.Image, err)
		}
		if b := img.Bounds(); b.Dx() != ts.ImageWidth || b.Dy() != ts.ImageHeight {
			t.Errorf("%s is %v, tileset declares %dx%d", ts.Image, b, ts.ImageWidth, ts.ImageHeight)
		}
	}
}
