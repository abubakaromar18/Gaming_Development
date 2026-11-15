// Package replay records and plays back games as (seed, config, inputs).
//
// Because the simulation is deterministic, a replay doesn't store any game
// state at all: a 10-minute game is a 50-byte header plus a few kilobytes of
// run-length-encoded input bytes. Periodic state hashes ("checkpoints") are
// stored alongside so playback can prove it is reproducing the original run,
// and report the exact tick where it stopped doing so.
package replay

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"

	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/core"
)

// Version is the current file format version. Bump it whenever the format
// or the simulation's behaviour changes, so old files fail loudly with
// ErrUnsupportedVersion rather than playing back as a different game.
const Version uint16 = 1

var magic = [4]byte{'A', 'R', 'E', 'P'}

// Limits that keep a hostile or corrupt file from allocating unbounded memory.
const (
	maxTicks       = 60 * 60 * 60 * 4 // four hours at 60 Hz
	maxCheckpoints = maxTicks
)

// Decoding errors. Use errors.Is to classify a failure.
var (
	ErrBadMagic           = errors.New("replay: not a replay file")
	ErrUnsupportedVersion = errors.New("replay: unsupported format version")
	ErrChecksum           = errors.New("replay: checksum mismatch, file is corrupt")
	ErrMalformed          = errors.New("replay: malformed data")
)

// Checkpoint is the world hash observed after Tick ticks had been simulated.
type Checkpoint struct {
	Tick uint32
	Hash uint64
}

// Replay is everything needed to reproduce a game.
type Replay struct {
	Seed        uint64
	Config      core.Config
	Inputs      []core.Input // Inputs[i] drives tick i
	Checkpoints []Checkpoint // ascending by Tick
}

// header is the fixed-size front of the file, written with encoding/binary.
type header struct {
	Magic   [4]byte
	Version uint16
	Seed    uint64
	Config  core.Config
	Ticks   uint32
}

// Encode writes r in the binary replay format:
//
//	header | runs | checkpoints | crc32
//
// Inputs are run-length encoded as (input byte, uvarint length) pairs.
// Players hold a key for many ticks at a time, so runs compress well.
func Encode(dst io.Writer, r *Replay) error {
	if err := r.validate(); err != nil {
		return err
	}
	var buf bytes.Buffer
	h := header{Magic: magic, Version: Version, Seed: r.Seed, Config: r.Config, Ticks: uint32(len(r.Inputs))}
	if err := binary.Write(&buf, binary.LittleEndian, h); err != nil {
		return fmt.Errorf("replay: writing header: %w", err)
	}

	runs := encodeRuns(r.Inputs)
	buf.Write(binary.AppendUvarint(nil, uint64(len(runs))))
	for _, run := range runs {
		buf.WriteByte(byte(run.input))
		buf.Write(binary.AppendUvarint(nil, uint64(run.length)))
	}

	buf.Write(binary.AppendUvarint(nil, uint64(len(r.Checkpoints))))
	for _, c := range r.Checkpoints {
		buf.Write(binary.AppendUvarint(nil, uint64(c.Tick)))
		buf.Write(binary.LittleEndian.AppendUint64(nil, c.Hash))
	}

	sum := crc32.ChecksumIEEE(buf.Bytes())
	buf.Write(binary.LittleEndian.AppendUint32(nil, sum))
	_, err := dst.Write(buf.Bytes())
	return err
}

// Decode reads and validates a replay. A truncated file reports an error
// wrapping io.ErrUnexpectedEOF; other problems wrap one of the Err values.
func Decode(src io.Reader) (*Replay, error) {
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("replay: reading: %w", err)
	}
	if len(data) < 4 || !bytes.Equal(data[:4], magic[:]) {
		return nil, ErrBadMagic
	}
	if len(data) < 6 {
		return nil, fmt.Errorf("replay: header: %w", io.ErrUnexpectedEOF)
	}
	if v := binary.LittleEndian.Uint16(data[4:6]); v != Version {
		return nil, fmt.Errorf("%w: file is v%d, this build reads v%d", ErrUnsupportedVersion, v, Version)
	}
	if len(data) < binary.Size(header{})+4 {
		return nil, fmt.Errorf("replay: header: %w", io.ErrUnexpectedEOF)
	}
	body, trailer := data[:len(data)-4], data[len(data)-4:]
	if crc32.ChecksumIEEE(body) != binary.LittleEndian.Uint32(trailer) {
		return nil, ErrChecksum
	}

	rd := bufio.NewReader(bytes.NewReader(body))
	var h header
	if err := binary.Read(rd, binary.LittleEndian, &h); err != nil {
		return nil, fmt.Errorf("replay: header: %w", err)
	}
	if h.Ticks > maxTicks {
		return nil, fmt.Errorf("%w: %d ticks exceeds the %d limit", ErrMalformed, h.Ticks, maxTicks)
	}
	r := &Replay{Seed: h.Seed, Config: h.Config}

	if r.Inputs, err = decodeRuns(rd, h.Ticks); err != nil {
		return nil, err
	}
	if r.Checkpoints, err = decodeCheckpoints(rd); err != nil {
		return nil, err
	}
	if rd.Buffered() > 0 {
		return nil, fmt.Errorf("%w: %d trailing bytes", ErrMalformed, rd.Buffered())
	}
	if err := r.validate(); err != nil {
		return nil, err
	}
	return r, nil
}

type run struct {
	input  core.Input
	length uint32
}

func encodeRuns(inputs []core.Input) []run {
	var runs []run
	for _, in := range inputs {
		if n := len(runs); n > 0 && runs[n-1].input == in {
			runs[n-1].length++
			continue
		}
		runs = append(runs, run{input: in, length: 1})
	}
	return runs
}

func decodeRuns(rd *bufio.Reader, ticks uint32) ([]core.Input, error) {
	count, err := readUvarint(rd, "run count")
	if err != nil {
		return nil, err
	}
	if count > uint64(ticks) {
		return nil, fmt.Errorf("%w: %d runs for %d ticks", ErrMalformed, count, ticks)
	}
	inputs := make([]core.Input, 0, ticks)
	for i := uint64(0); i < count; i++ {
		b, err := rd.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("replay: run %d: %w", i, io.ErrUnexpectedEOF)
		}
		length, err := readUvarint(rd, "run length")
		if err != nil {
			return nil, err
		}
		if length == 0 || uint64(len(inputs))+length > uint64(ticks) {
			return nil, fmt.Errorf("%w: run %d has bad length %d", ErrMalformed, i, length)
		}
		for j := uint64(0); j < length; j++ {
			inputs = append(inputs, core.Input(b))
		}
	}
	if uint32(len(inputs)) != ticks {
		return nil, fmt.Errorf("%w: runs cover %d ticks, header says %d", ErrMalformed, len(inputs), ticks)
	}
	return inputs, nil
}

func decodeCheckpoints(rd *bufio.Reader) ([]Checkpoint, error) {
	count, err := readUvarint(rd, "checkpoint count")
	if err != nil {
		return nil, err
	}
	if count > maxCheckpoints {
		return nil, fmt.Errorf("%w: %d checkpoints", ErrMalformed, count)
	}
	out := make([]Checkpoint, 0, count)
	for i := uint64(0); i < count; i++ {
		tick, err := readUvarint(rd, "checkpoint tick")
		if err != nil {
			return nil, err
		}
		var raw [8]byte
		if _, err := io.ReadFull(rd, raw[:]); err != nil {
			return nil, fmt.Errorf("replay: checkpoint %d: %w", i, io.ErrUnexpectedEOF)
		}
		out = append(out, Checkpoint{Tick: uint32(tick), Hash: binary.LittleEndian.Uint64(raw[:])})
	}
	return out, nil
}

func readUvarint(rd *bufio.Reader, what string) (uint64, error) {
	v, err := binary.ReadUvarint(rd)
	if err != nil {
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return 0, fmt.Errorf("replay: %s: %w", what, err)
	}
	return v, nil
}

// validate checks invariants shared by Encode and Decode.
func (r *Replay) validate() error {
	if err := r.Config.Validate(); err != nil {
		return fmt.Errorf("replay: %w", err)
	}
	if len(r.Inputs) > maxTicks {
		return fmt.Errorf("%w: %d ticks exceeds the %d limit", ErrMalformed, len(r.Inputs), maxTicks)
	}
	for i, in := range r.Inputs {
		if !in.Valid() {
			return fmt.Errorf("%w: input %#x at tick %d uses undefined bits", ErrMalformed, uint8(in), i)
		}
	}
	prev := int64(-1)
	for _, c := range r.Checkpoints {
		if int64(c.Tick) <= prev || int(c.Tick) > len(r.Inputs) {
			return fmt.Errorf("%w: checkpoint at tick %d is out of order or past the end", ErrMalformed, c.Tick)
		}
		prev = int64(c.Tick)
	}
	return nil
}
