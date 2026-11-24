// Command genmap writes the procedurally generated demo map, its external
// tileset, and both tileset images into a directory.
//
//	go run ./cmd/genmap -out maps -seed 14
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/internal/mapgen"
)

func main() {
	out := flag.String("out", "maps", "output directory")
	seed := flag.Uint64("seed", mapgen.DefaultSeed, "generation seed")
	flag.Parse()
	if err := run(*out, *seed); err != nil {
		fmt.Fprintln(os.Stderr, "genmap:", err)
		os.Exit(1)
	}
}

func run(dir string, seed uint64) error {
	files, err := mapgen.Generate(seed)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, files[name], 0o644); err != nil {
			return err
		}
		fmt.Printf("wrote %s (%d bytes)\n", p, len(files[name]))
	}
	return nil
}
