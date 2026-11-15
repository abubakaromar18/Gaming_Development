package replay

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"testing"

	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/bot"
	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/core"
)

// recordBotGame plays ticks of bot-driven Asteroids and returns the
// recording and the final live world.
func recordBotGame(t testing.TB, seed uint64, ticks int) (*Replay, *core.World) {
	t.Helper()
	rec, err := NewRecorder(core.DefaultConfig(), seed, DefaultCheckpointEvery)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < ticks; i++ {
		rec.Step(bot.Decide(rec.World()))
	}
	return rec.Replay(), rec.World()
}

func encode(t testing.TB, r *Replay) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := Encode(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// resign recomputes the CRC trailer so tests can craft structurally broken
// files that still pass the checksum and reach the deeper validation.
func resign(data []byte) []byte {
	body := data[:len(data)-4]
	return binary.LittleEndian.AppendUint32(append([]byte(nil), body...), crc32.ChecksumIEEE(body))
}

func TestBotGameIsEventful(t *testing.T) {
	_, w := recordBotGame(t, 11, 3*60*60)
	if w.Score == 0 || w.Wave < 2 {
		t.Fatalf("bot should score and clear waves; score=%d wave=%d", w.Score, w.Wave)
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	rep, _ := recordBotGame(t, 1, 2000)
	data := encode(t, rep)
	got, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if got.Seed != rep.Seed || got.Config != rep.Config {
		t.Fatal("header fields changed in round trip")
	}
	if !bytes.Equal(inputBytes(got.Inputs), inputBytes(rep.Inputs)) {
		t.Fatal("inputs changed in round trip")
	}
	if len(got.Checkpoints) != len(rep.Checkpoints) || got.Checkpoints[len(got.Checkpoints)-1] != rep.Checkpoints[len(rep.Checkpoints)-1] {
		t.Fatal("checkpoints changed in round trip")
	}
	// Run-length encoding keeps the input stream far below one byte per tick.
	if inputSize := len(data) - binary.Size(header{}) - 12*len(rep.Checkpoints); inputSize > len(rep.Inputs)/2 {
		t.Fatalf("inputs took %d bytes for %d ticks; RLE should do better", inputSize, len(rep.Inputs))
	}
}

func inputBytes(in []core.Input) []byte {
	b := make([]byte, len(in))
	for i, v := range in {
		b[i] = byte(v)
	}
	return b
}

// The headline property: playing back a recording reproduces the live game.
func TestPlaybackReproducesLiveGame(t *testing.T) {
	for _, seed := range []uint64{1, 2, 3, 99, 2026} {
		rep, live := recordBotGame(t, seed, 4000)
		decoded, err := Decode(bytes.NewReader(encode(t, rep)))
		if err != nil {
			t.Fatal(err)
		}
		final, err := Verify(decoded)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if final.Hash() != live.Hash() || final.Score != live.Score {
			t.Fatalf("seed %d: playback ended at score %d, live game scored %d", seed, final.Score, live.Score)
		}
	}
}

func TestTamperedInputIsCaughtAtNextCheckpoint(t *testing.T) {
	rep, _ := recordBotGame(t, 5, 3000)
	const tampered = 1234
	rep.Inputs[tampered] ^= core.InputFire | core.InputThrust
	_, err := Verify(rep)
	var desync *DesyncError
	if !errors.As(err, &desync) {
		t.Fatalf("want DesyncError, got %v", err)
	}
	// 1234 lies between the checkpoints at 1200 and 1260.
	if desync.Tick != 1260 {
		t.Fatalf("desync reported at tick %d, want 1260", desync.Tick)
	}
}

func TestWrongSeedOrConfigDesyncs(t *testing.T) {
	rep, _ := recordBotGame(t, 8, 600)
	wrongSeed := *rep
	wrongSeed.Seed++
	if _, err := Verify(&wrongSeed); !isDesync(err) {
		t.Fatalf("wrong seed: %v", err)
	}
	retuned := *rep
	retuned.Config.BulletSpeed++
	if _, err := Verify(&retuned); !isDesync(err) {
		t.Fatalf("retuned config: %v", err)
	}
}

func isDesync(err error) bool {
	var d *DesyncError
	return errors.As(err, &d)
}

func TestSeekMatchesLinearPlayback(t *testing.T) {
	rep, _ := recordBotGame(t, 21, 2500)
	linear, err := NewPlayer(rep, 100)
	if err != nil {
		t.Fatal(err)
	}
	hashAt := map[uint32]uint64{}
	for !linear.Done() {
		if err := linear.Step(); err != nil {
			t.Fatal(err)
		}
		hashAt[linear.Tick()] = linear.World().Hash()
	}

	p, err := NewPlayer(rep, 100)
	if err != nil {
		t.Fatal(err)
	}
	// Forward past uncached keyframes, back, forward within a window,
	// exactly onto a keyframe, and past the end (clamped).
	for _, target := range []uint32{1700, 350, 420, 1999, 1300, 400, 2400, 9999} {
		if err := p.Seek(target); err != nil {
			t.Fatalf("seek %d: %v", target, err)
		}
		want := target
		if want > p.Len() {
			want = p.Len()
		}
		if p.Tick() != want || p.World().Hash() != hashAt[want] {
			t.Fatalf("seek to %d landed on tick %d with the wrong state", target, p.Tick())
		}
	}
	if err := p.Step(); !errors.Is(err, ErrEnded) {
		t.Fatalf("step past the end: %v", err)
	}
	if err := p.Seek(0); err != nil || p.Tick() != 0 {
		t.Fatalf("seek to start: tick %d err %v", p.Tick(), err)
	}
	if p.Input() != rep.Inputs[0] {
		t.Fatal("Input should expose the next recorded input")
	}
}

func TestSeekingDoesNotMutateKeyframes(t *testing.T) {
	rep, _ := recordBotGame(t, 4, 1000)
	p, _ := NewPlayer(rep, 50)
	if err := p.Seek(500); err != nil {
		t.Fatal(err)
	}
	if err := p.Seek(100); err != nil {
		t.Fatal(err)
	}
	h := p.World().Hash()
	_ = p.Seek(900)
	_ = p.Seek(100)
	if p.World().Hash() != h {
		t.Fatal("restoring the same keyframe twice gave different states: keyframe was mutated")
	}
}

func TestRecorderSnapshotsAreIndependentAndEndWithCheckpoint(t *testing.T) {
	rec, err := NewRecorder(core.DefaultConfig(), 3, 60)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 130; i++ {
		rec.Step(core.InputFire | 0xF0) // undefined bits are masked off
	}
	snap := rec.Replay()
	if len(snap.Inputs) != 130 || snap.Inputs[0] != core.InputFire {
		t.Fatalf("recorded %d inputs, first %#x", len(snap.Inputs), snap.Inputs[0])
	}
	ticks := [3]uint32{snap.Checkpoints[0].Tick, snap.Checkpoints[1].Tick, snap.Checkpoints[2].Tick}
	if ticks != [3]uint32{60, 120, 130} {
		t.Fatalf("checkpoints at %v, want [60 120 130]", ticks)
	}
	rec.Step(0)
	if len(snap.Inputs) != 130 {
		t.Fatal("snapshot shares memory with the live recorder")
	}
	if _, err := Verify(snap); err != nil {
		t.Fatal(err)
	}
	if _, err := NewRecorder(core.DefaultConfig(), 1, 0); err == nil {
		t.Fatal("zero checkpoint interval accepted")
	}
}

func TestDecodeRejectsBadFiles(t *testing.T) {
	rep, _ := recordBotGame(t, 9, 500)
	good := encode(t, rep)

	cases := map[string]struct {
		data []byte
		want error
	}{
		"empty":         {nil, ErrBadMagic},
		"wrong magic":   {append([]byte("NOPE"), good[4:]...), ErrBadMagic},
		"short header":  {good[:20], io.ErrUnexpectedEOF},
		"flipped bit":   {flip(good, len(good)/2), ErrChecksum},
		"truncated":     {good[:len(good)-7], ErrChecksum},
		"future format": {resign(withVersion(good, Version+1)), ErrUnsupportedVersion},
		"trailing junk": {resign(append(append([]byte(nil), good[:len(good)-4]...), 0, 0, 0, 0, 0)), ErrMalformed},
		"invalid config": {
			resign(withUint32At(good, 14, 5)), // Config.Width = 5
			core.ErrInvalidConfig,
		},
		"tick count mismatch": {
			resign(withUint32At(good, binary.Size(header{})-4, uint32(len(rep.Inputs)+1))),
			ErrMalformed,
		},
	}
	for name, c := range cases {
		_, err := Decode(bytes.NewReader(c.data))
		if !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", name, err, c.want)
		}
	}
}

func flip(data []byte, i int) []byte {
	out := append([]byte(nil), data...)
	out[i] ^= 0x01
	return out
}

func withVersion(data []byte, v uint16) []byte {
	out := append([]byte(nil), data...)
	binary.LittleEndian.PutUint16(out[4:6], v)
	return out
}

func withUint32At(data []byte, off int, v uint32) []byte {
	out := append([]byte(nil), data...)
	binary.LittleEndian.PutUint32(out[off:], v)
	return out
}

func TestEncodeRejectsInvalidReplays(t *testing.T) {
	rep, _ := recordBotGame(t, 1, 100)
	bad := *rep
	bad.Inputs = append([]core.Input(nil), rep.Inputs...)
	bad.Inputs[3] = 0x80
	if err := Encode(io.Discard, &bad); !errors.Is(err, ErrMalformed) {
		t.Fatalf("undefined input bits: %v", err)
	}
	unordered := *rep
	unordered.Checkpoints = []Checkpoint{{Tick: 50}, {Tick: 40}}
	if err := Encode(io.Discard, &unordered); !errors.Is(err, ErrMalformed) {
		t.Fatalf("unordered checkpoints: %v", err)
	}
	if _, err := NewPlayer(&unordered, 10); !errors.Is(err, ErrMalformed) {
		t.Fatalf("player accepted unordered checkpoints: %v", err)
	}
}

func BenchmarkSimulateOneMinute(b *testing.B) {
	rep, _ := recordBotGame(b, 1, 60*60)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Verify(rep); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBackwardSeek(b *testing.B) {
	rep, _ := recordBotGame(b, 1, 60*60)
	p, _ := NewPlayer(rep, DefaultKeyframeEvery)
	if err := p.Seek(p.Len()); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Alternate between two far-apart ticks; each seek restores a keyframe.
		target := uint32(1000 + (i%2)*2000)
		if err := p.Seek(target); err != nil {
			b.Fatal(err)
		}
	}
}
