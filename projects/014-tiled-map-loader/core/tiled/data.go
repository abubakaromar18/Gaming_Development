package tiled

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// decodeTileData turns a layer's (or chunk's) "data" field into exactly want
// GIDs. Tiled writes either a JSON array of numbers ("csv" in the editor) or
// a base64 string of little-endian uint32s, optionally compressed.
func decodeTileData(data json.RawMessage, encoding, compression string, want int) ([]GID, error) {
	switch encoding {
	case "", "csv":
		var ids []uint32
		if err := json.Unmarshal(data, &ids); err != nil {
			return nil, fmt.Errorf("%w: tile data is not an array of tile IDs: %v", ErrInvalid, err)
		}
		if len(ids) != want {
			return nil, fmt.Errorf("%w: tile data has %d entries, want %d", ErrInvalid, len(ids), want)
		}
		gids := make([]GID, want)
		for i, id := range ids {
			gids[i] = GID(id)
		}
		return gids, nil
	case "base64":
		return decodeBase64(data, compression, want)
	default:
		return nil, fmt.Errorf("%w: layer encoding %q", ErrUnsupported, encoding)
	}
}

func decodeBase64(data json.RawMessage, compression string, want int) ([]GID, error) {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return nil, fmt.Errorf("%w: base64 tile data is not a string: %v", ErrInvalid, err)
	}
	packed, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, fmt.Errorf("%w: bad base64 tile data: %v", ErrInvalid, err)
	}
	wantBytes := want * 4
	if compression != "" {
		if packed, err = inflate(packed, compression, wantBytes); err != nil {
			return nil, err
		}
	}
	if len(packed) != wantBytes {
		return nil, fmt.Errorf("%w: tile data has %d bytes, want %d", ErrInvalid, len(packed), wantBytes)
	}
	gids := make([]GID, want)
	for i := range gids {
		gids[i] = GID(binary.LittleEndian.Uint32(packed[i*4:]))
	}
	return gids, nil
}

// inflate decompresses at most limit+1 bytes. Reading one byte past the
// expected size is enough to detect oversized data, and the cap means a
// malicious "zip bomb" layer cannot exhaust memory.
func inflate(packed []byte, compression string, limit int) ([]byte, error) {
	var r io.ReadCloser
	var err error
	switch compression {
	case "zlib":
		r, err = zlib.NewReader(bytes.NewReader(packed))
	case "gzip":
		r, err = gzip.NewReader(bytes.NewReader(packed))
	default:
		return nil, fmt.Errorf("%w: compression %q (re-export with zlib, gzip, or none)", ErrUnsupported, compression)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: bad %s stream: %v", ErrInvalid, compression, err)
	}
	defer r.Close()
	out, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if err != nil {
		return nil, fmt.Errorf("%w: bad %s stream: %v", ErrInvalid, compression, err)
	}
	return out, nil
}
