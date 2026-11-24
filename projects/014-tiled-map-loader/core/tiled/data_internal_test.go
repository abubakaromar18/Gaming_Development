package tiled

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"testing"
)

func pack(t *testing.T, gids []uint32, compression string) json.RawMessage {
	t.Helper()
	raw := make([]byte, 4*len(gids))
	for i, g := range gids {
		binary.LittleEndian.PutUint32(raw[i*4:], g)
	}
	var buf bytes.Buffer
	var w io.WriteCloser
	switch compression {
	case "zlib":
		w = zlib.NewWriter(&buf)
	case "gzip":
		w = gzip.NewWriter(&buf)
	}
	if w != nil {
		if _, err := w.Write(raw); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		raw = buf.Bytes()
	}
	out, _ := json.Marshal(base64.StdEncoding.EncodeToString(raw))
	return out
}

func TestDecodeTileDataRoundTrips(t *testing.T) {
	gids := []uint32{0, 1, 0x80000002, 0xffffffff, 42, 7}
	csv, _ := json.Marshal(gids)
	inputs := map[string]struct {
		data              json.RawMessage
		encoding, compres string
	}{
		"csv":         {csv, "csv", ""},
		"csv-default": {csv, "", ""},
		"base64":      {pack(t, gids, ""), "base64", ""},
		"zlib":        {pack(t, gids, "zlib"), "base64", "zlib"},
		"gzip":        {pack(t, gids, "gzip"), "base64", "gzip"},
	}
	for name, in := range inputs {
		got, err := decodeTileData(in.data, in.encoding, in.compres, len(gids))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for i := range gids {
			if uint32(got[i]) != gids[i] {
				t.Fatalf("%s: tile %d = %#x, want %#x", name, i, uint32(got[i]), gids[i])
			}
		}
	}
}

func TestDecompressionIsBounded(t *testing.T) {
	// A stream that inflates to far more than the layer needs must be
	// rejected without reading it all.
	big := make([]uint32, 1<<16)
	_, err := decodeTileData(pack(t, big, "zlib"), "base64", "zlib", 4)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestDecodeRejectsUnknownEncoding(t *testing.T) {
	if _, err := decodeTileData(json.RawMessage(`""`), "rle", "", 1); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v", err)
	}
	if _, err := decodeTileData(json.RawMessage(`"!!!"`), "base64", "", 1); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad base64: err = %v", err)
	}
}
