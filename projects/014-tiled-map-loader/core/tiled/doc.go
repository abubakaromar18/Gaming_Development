// Package tiled loads maps exported from the Tiled map editor in its JSON
// formats (.tmj maps and .tsj tilesets) into a validated, engine-neutral model.
//
// The package has no dependency on any renderer. Files are read through an
// fs.FS, so maps can come from disk (os.DirFS), from an embedded filesystem
// (embed.FS), or from an in-memory fixture in tests (testing/fstest.MapFS).
//
// Supported: orthogonal maps, finite and infinite (chunked) tile layers, CSV
// and base64 layer data with zlib or gzip compression, embedded and external
// tilesets, group layers, object layers with every shape Tiled exports, typed
// custom properties, per-tile collision shapes, tile animations, and the
// flip/rotate flags packed into global tile IDs.
//
// Rejected with ErrUnsupported: isometric/staggered/hexagonal orientation,
// XML (.tmx/.tsx) files, zstd compression, image-collection tilesets, and
// object templates.
package tiled

import "errors"

var (
	// ErrInvalid marks data that is malformed or internally inconsistent.
	ErrInvalid = errors.New("tiled: invalid map data")
	// ErrUnsupported marks valid Tiled data that this loader deliberately
	// does not handle.
	ErrUnsupported = errors.New("tiled: unsupported feature")
)
