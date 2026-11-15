// Command game is the playable Ebitengine front end for the deterministic
// Asteroids core. Play live, press P for an instant replay of the run so far,
// save it with F5, and load saved files with -replay.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/core"
	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/replay"
)

func main() {
	seed := flag.Uint64("seed", 0, "RNG seed for the first game (0 = pick from the clock)")
	replayPath := flag.String("replay", "", "open this .arep file in playback mode")
	saveDir := flag.String("save-dir", "replays", "directory F5 writes replays into")
	autopilot := flag.Bool("autopilot", false, "let the bot fly the ship (toggle in game with B)")
	flag.Parse()

	// The clock is consulted here, in the shell, and nowhere in the
	// simulation: it only chooses seeds and seeds the cosmetic effects.
	now := uint64(time.Now().UnixNano())
	if *seed == 0 {
		*seed = now
	}
	g := newGame(gameOptions{
		cfg:       core.DefaultConfig(),
		seed:      *seed,
		saveDir:   *saveDir,
		autopilot: *autopilot,
		fxSeed:    now,
	})

	if *replayPath != "" {
		r, err := loadReplay(*replayPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot open replay: %v\n", err)
			os.Exit(1)
		}
		if err := g.startPlayback(r); err != nil {
			fmt.Fprintf(os.Stderr, "cannot play replay: %v\n", err)
			os.Exit(1)
		}
	} else if err := g.startLive(g.opts.seed); err != nil {
		fmt.Fprintf(os.Stderr, "cannot start game: %v\n", err)
		os.Exit(1)
	}

	cfg := g.opts.cfg
	ebiten.SetWindowSize(int(cfg.Width), int(cfg.Height))
	ebiten.SetWindowTitle("Deterministic Asteroids")
	// Ebitengine calls Update at a fixed TPS, independent of the display's
	// refresh rate, so one Update is exactly one simulation tick.
	ebiten.SetTPS(core.TicksPerSecond)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

func loadReplay(path string) (*replay.Replay, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return replay.Decode(f)
}
