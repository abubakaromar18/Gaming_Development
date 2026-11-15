// Command replaytool records, inspects and verifies Asteroids replays
// without opening a window. It doubles as a determinism check: record a
// demo on one machine, verify it on another, and the final hashes must match.
//
//	replaytool demo   -seed 7 -seconds 120 -o demo.arep
//	replaytool info   demo.arep
//	replaytool verify demo.arep
//	replaytool frame  -tick 1800 demo.arep
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/bot"
	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/core"
	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/replay"
)

const usage = `usage: replaytool <command> [flags]

commands:
  demo    record a bot-played game:      demo [-seed N] [-seconds N] -o FILE
  info    print a replay's header:       info FILE
  verify  re-simulate and check hashes:  verify FILE
  frame   draw one tick as ASCII art:    frame [-tick N] FILE
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is main without the process exit, so tests can drive it.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "demo":
		err = demo(args[1:], stdout)
	case "info":
		err = withReplay("info", args[1:], func(r *replay.Replay) error { return info(r, stdout) })
	case "verify":
		err = withReplay("verify", args[1:], func(r *replay.Replay) error { return verify(r, stdout) })
	case "frame":
		var tick uint
		err = withReplay("frame", args[1:], func(r *replay.Replay) error {
			return frame(r, uint32(tick), stdout)
		}, func(fs *flag.FlagSet) { fs.UintVar(&tick, "tick", 0, "tick to draw") })
	case "-h", "-help", "--help", "help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		err = fmt.Errorf("unknown command %q", args[0])
	}
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(stderr, "replaytool: %v\n", err)
		}
		var desync *replay.DesyncError
		if errors.As(err, &desync) {
			return 3
		}
		return 1
	}
	return 0
}

func demo(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("demo", flag.ContinueOnError)
	seed := fs.Uint64("seed", 1, "RNG seed")
	seconds := fs.Uint("seconds", 60, "length of the recording")
	out := fs.String("o", "", "output file (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return errors.New("demo: -o FILE is required")
	}
	if *seconds == 0 || *seconds > 4*60*60 {
		return errors.New("demo: -seconds must be between 1 and 14400")
	}
	rec, err := replay.NewRecorder(core.DefaultConfig(), *seed, replay.DefaultCheckpointEvery)
	if err != nil {
		return err
	}
	for i := uint(0); i < *seconds*core.TicksPerSecond; i++ {
		rec.Step(bot.Decide(rec.World()))
	}
	if err := writeReplay(*out, rec.Replay()); err != nil {
		return err
	}
	w := rec.World()
	fmt.Fprintf(stdout, "recorded %d ticks to %s: score %d, wave %d, hash %#016x\n", w.Tick, *out, w.Score, w.Wave, w.Hash())
	return nil
}

func writeReplay(path string, r *replay.Replay) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	return replay.Encode(f, r)
}

// withReplay parses flags plus one FILE argument, decodes it and calls fn.
func withReplay(name string, args []string, fn func(*replay.Replay) error, extraFlags ...func(*flag.FlagSet)) error {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	for _, add := range extraFlags {
		add(fs)
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%s: expected exactly one replay file", name)
	}
	f, err := os.Open(fs.Arg(0))
	if err != nil {
		return err
	}
	defer f.Close()
	r, err := replay.Decode(f)
	if err != nil {
		return err
	}
	return fn(r)
}

func info(r *replay.Replay, stdout io.Writer) error {
	secs := float64(len(r.Inputs)) / core.TicksPerSecond
	fmt.Fprintf(stdout, "format      v%d\nseed        %d\nplayfield   %dx%d\nticks       %d (%.1fs)\ncheckpoints %d\n",
		replay.Version, r.Seed, r.Config.Width, r.Config.Height, len(r.Inputs), secs, len(r.Checkpoints))
	return nil
}

func verify(r *replay.Replay, stdout io.Writer) error {
	w, err := replay.Verify(r)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "OK: %d ticks, %d checkpoints matched, score %d, final hash %#016x\n",
		w.Tick, len(r.Checkpoints), w.Score, w.Hash())
	return nil
}

func frame(r *replay.Replay, tick uint32, stdout io.Writer) error {
	p, err := replay.NewPlayer(r, replay.DefaultKeyframeEvery)
	if err != nil {
		return err
	}
	if tick > p.Len() {
		return fmt.Errorf("frame: tick %d is past the end (%d)", tick, p.Len())
	}
	if err := p.Seek(tick); err != nil {
		return err
	}
	fmt.Fprint(stdout, renderASCII(p.World(), 64, 24))
	return nil
}

// renderASCII draws a coarse picture of the world: rocks by size (o O @),
// bullets (.), the ship as an arrow for its heading, and a status line.
func renderASCII(w *core.World, cols, rows int) string {
	grid := make([][]byte, rows)
	for i := range grid {
		grid[i] = []byte(strings.Repeat(" ", cols))
	}
	plot := func(x, y int, c byte) {
		col := x * cols / int(w.Cfg.Width)
		row := y * rows / int(w.Cfg.Height)
		if col >= 0 && col < cols && row >= 0 && row < rows {
			grid[row][col] = c
		}
	}
	for _, b := range w.Bullets {
		plot(b.Pos.X.Int(), b.Pos.Y.Int(), '.')
	}
	rockGlyph := map[core.Size]byte{core.Small: 'o', core.Medium: 'O', core.Large: '@'}
	for _, a := range w.Asteroids {
		plot(a.Pos.X.Int(), a.Pos.Y.Int(), rockGlyph[a.Size])
	}
	if w.Ship.Alive {
		arrows := []byte{'>', '\\', 'v', '/', '<', '\\', '^', '/'}
		plot(w.Ship.Pos.X.Int(), w.Ship.Pos.Y.Int(), arrows[(int(w.Ship.Heading)+16)/32%8])
	}
	var sb strings.Builder
	border := "+" + strings.Repeat("-", cols) + "+\n"
	sb.WriteString(border)
	for _, line := range grid {
		sb.WriteString("|" + string(line) + "|\n")
	}
	sb.WriteString(border)
	fmt.Fprintf(&sb, "tick %d  score %d  lives %d  wave %d\n", w.Tick, w.Score, w.Lives, w.Wave)
	return sb.String()
}
