package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/bot"
	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/core"
	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/replay"
)

type mode int

const (
	modeLive mode = iota
	modePlayback
)

// seekStep is how far the arrow keys jump during playback.
const seekStep = 5 * core.TicksPerSecond

var playbackSpeeds = [...]int{1, 2, 4, 8}

type gameOptions struct {
	cfg       core.Config
	seed      uint64
	saveDir   string
	autopilot bool
	fxSeed    uint64
}

// Game is the Ebitengine shell. It owns either a Recorder (live play) or a
// Player (playback) and never touches World state directly; all state
// changes go through Step so the recording is always complete.
type Game struct {
	opts gameOptions
	mode mode

	rec    *replay.Recorder
	player *replay.Player
	paused bool
	speed  int // index into playbackSpeeds
	desync error

	fx      *effects
	message string
	msgTTL  int
}

func newGame(opts gameOptions) *Game {
	return &Game{opts: opts, fx: newEffects(opts.cfg, opts.fxSeed)}
}

func (g *Game) startLive(seed uint64) error {
	rec, err := replay.NewRecorder(g.opts.cfg, seed, replay.DefaultCheckpointEvery)
	if err != nil {
		return err
	}
	g.mode, g.rec, g.player, g.desync = modeLive, rec, nil, nil
	g.fx.reset()
	g.say(fmt.Sprintf("seed %d", seed))
	return nil
}

func (g *Game) startPlayback(r *replay.Replay) error {
	p, err := replay.NewPlayer(r, replay.DefaultKeyframeEvery)
	if err != nil {
		return err
	}
	g.mode, g.player, g.paused, g.speed, g.desync = modePlayback, p, false, 0, nil
	g.fx.reset()
	g.say("playback")
	return nil
}

func (g *Game) world() *core.World {
	if g.mode == modePlayback {
		return g.player.World()
	}
	return g.rec.World()
}

func (g *Game) say(msg string) { g.message, g.msgTTL = msg, 2*core.TicksPerSecond }

// Update runs exactly once per simulation tick (see ebiten.SetTPS in main).
func (g *Game) Update() error {
	if g.msgTTL > 0 {
		g.msgTTL--
	}
	var err error
	if g.mode == modeLive {
		err = g.updateLive()
	} else {
		err = g.updatePlayback()
	}
	g.fx.update()
	return err
}

func (g *Game) updateLive() error {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		return ebiten.Termination
	case inpututil.IsKeyJustPressed(ebiten.KeyR):
		g.opts.seed++
		return g.startLive(g.opts.seed)
	case inpututil.IsKeyJustPressed(ebiten.KeyB):
		g.opts.autopilot = !g.opts.autopilot
	case inpututil.IsKeyJustPressed(ebiten.KeyF5):
		g.save(g.rec.Replay())
	case inpututil.IsKeyJustPressed(ebiten.KeyP),
		g.rec.World().Over && inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		// Instant replay: the recording so far, played from tick 0.
		return g.startPlayback(g.rec.Replay())
	}

	in := keyboardInput()
	if g.opts.autopilot {
		in = bot.Decide(g.rec.World())
	}
	g.rec.Step(in)
	g.fx.consume(g.rec.World().Events)
	return nil
}

func (g *Game) updatePlayback() error {
	p := g.player
	seekTo := func(t uint32) {
		g.desync = p.Seek(t)
		g.fx.reset()
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		g.opts.seed++
		return g.startLive(g.opts.seed)
	case inpututil.IsKeyJustPressed(ebiten.KeySpace):
		g.paused = !g.paused
	case inpututil.IsKeyJustPressed(ebiten.KeyHome):
		seekTo(0)
	case inpututil.IsKeyJustPressed(ebiten.KeyArrowRight):
		seekTo(p.Tick() + seekStep)
	case inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft):
		if p.Tick() < seekStep {
			seekTo(0)
		} else {
			seekTo(p.Tick() - seekStep)
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyF5):
		g.save(p.Replay())
	case g.paused && inpututil.IsKeyJustPressed(ebiten.KeyPeriod):
		g.stepPlayback()
	}
	for i, k := range []ebiten.Key{ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3, ebiten.KeyDigit4} {
		if inpututil.IsKeyJustPressed(k) {
			g.speed = i
		}
	}
	if g.paused || g.desync != nil {
		return nil
	}
	for i := 0; i < playbackSpeeds[g.speed] && !p.Done(); i++ {
		if !g.stepPlayback() {
			break
		}
	}
	return nil
}

// stepPlayback advances one tick and reports whether playback can continue.
func (g *Game) stepPlayback() bool {
	err := g.player.Step()
	if errors.Is(err, replay.ErrEnded) {
		return false
	}
	if err != nil {
		g.desync, g.paused = err, true
		return false
	}
	g.fx.consume(g.player.World().Events)
	return true
}

// save writes r to the save directory and reports the outcome on screen.
func (g *Game) save(r *replay.Replay) {
	if err := os.MkdirAll(g.opts.saveDir, 0o755); err != nil {
		g.say("save failed: " + err.Error())
		return
	}
	path := filepath.Join(g.opts.saveDir, fmt.Sprintf("asteroids-%d-%d.arep", r.Seed, len(r.Inputs)))
	f, err := os.Create(path)
	if err != nil {
		g.say("save failed: " + err.Error())
		return
	}
	err = replay.Encode(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		g.say("save failed: " + err.Error())
		return
	}
	g.say("saved " + path)
}

// keyboardInput maps physical keys to the core's intent bits. Project 002
// covers rebindable action maps; this game keeps a fixed layout.
func keyboardInput() core.Input {
	var in core.Input
	pressed := func(keys ...ebiten.Key) bool {
		for _, k := range keys {
			if ebiten.IsKeyPressed(k) {
				return true
			}
		}
		return false
	}
	if pressed(ebiten.KeyArrowLeft, ebiten.KeyA) {
		in |= core.InputLeft
	}
	if pressed(ebiten.KeyArrowRight, ebiten.KeyD) {
		in |= core.InputRight
	}
	if pressed(ebiten.KeyArrowUp, ebiten.KeyW) {
		in |= core.InputThrust
	}
	if pressed(ebiten.KeySpace) {
		in |= core.InputFire
	}
	return in
}

// Layout keeps a fixed logical resolution; Ebitengine scales it to the window.
func (g *Game) Layout(_, _ int) (int, int) {
	return int(g.opts.cfg.Width), int(g.opts.cfg.Height)
}
