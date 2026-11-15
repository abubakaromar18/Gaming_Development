package main

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/core"
)

func runCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestDemoThenVerifyInfoAndFrame(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo.arep")
	code, out, errOut := runCLI(t, "demo", "-seed", "7", "-seconds", "30", "-o", path)
	if code != 0 || !strings.Contains(out, "recorded 1800 ticks") {
		t.Fatalf("demo: code %d out %q err %q", code, out, errOut)
	}
	recordedHash := out[strings.LastIndex(out, "hash ")+5:]

	code, out, _ = runCLI(t, "verify", path)
	if code != 0 || !strings.HasPrefix(out, "OK: 1800 ticks") || !strings.HasSuffix(out, recordedHash) {
		t.Fatalf("verify: code %d out %q, want final hash %q", code, out, recordedHash)
	}

	code, out, _ = runCLI(t, "info", path)
	if code != 0 || !strings.Contains(out, "seed        7") || !strings.Contains(out, "(30.0s)") {
		t.Fatalf("info: %q", out)
	}

	code, out, _ = runCLI(t, "frame", "-tick", "600", path)
	if code != 0 || !strings.Contains(out, "tick 600") || !strings.ContainsAny(out, "@Oo") {
		t.Fatalf("frame: %q", out)
	}
	if code, _, errOut = runCLI(t, "frame", "-tick", "99999", path); code != 1 || !strings.Contains(errOut, "past the end") {
		t.Fatalf("frame past end: code %d err %q", code, errOut)
	}
}

func TestVerifyReportsDesyncWithExitCode3(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo.arep")
	if code, _, errOut := runCLI(t, "demo", "-seconds", "10", "-o", path); code != 0 {
		t.Fatal(errOut)
	}
	// Corrupt the seed and re-sign the file, as a buggy build would produce.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[6] ^= 1
	if err := os.WriteFile(path, resign(data), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := runCLI(t, "verify", path)
	if code != 3 || !strings.Contains(errOut, "desynced at tick 60") {
		t.Fatalf("code %d err %q", code, errOut)
	}
}

func TestErrors(t *testing.T) {
	cases := [][]string{
		{},
		{"bogus"},
		{"demo"},
		{"demo", "-seconds", "0", "-o", "x"},
		{"verify"},
		{"verify", filepath.Join(t.TempDir(), "missing.arep")},
	}
	for _, args := range cases {
		if code, _, errOut := runCLI(t, args...); code == 0 || errOut == "" {
			t.Errorf("%v: code %d, stderr %q", args, code, errOut)
		}
	}
	garbage := filepath.Join(t.TempDir(), "garbage.arep")
	_ = os.WriteFile(garbage, []byte("not a replay"), 0o644)
	if _, _, errOut := runCLI(t, "verify", garbage); !strings.Contains(errOut, "not a replay file") {
		t.Errorf("garbage file: %q", errOut)
	}
	if code, out, _ := runCLI(t, "help"); code != 0 || !strings.Contains(out, "usage") {
		t.Error("help should print usage")
	}
}

func TestRenderASCIIShowsShipHeading(t *testing.T) {
	w, _ := core.NewWorld(core.DefaultConfig(), 1)
	w.Asteroids = nil
	if !strings.Contains(renderASCII(w, 40, 12), "^") {
		t.Fatal("a fresh ship points up")
	}
}

// resign recomputes the CRC32 trailer after a deliberate edit.
func resign(data []byte) []byte {
	body := data[:len(data)-4]
	sum := crc32.ChecksumIEEE(body)
	return binary.LittleEndian.AppendUint32(append([]byte(nil), body...), sum)
}
