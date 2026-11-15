package replay

import (
	"fmt"

	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/core"
)

// DefaultCheckpointEvery records a state hash once per second of play.
// Denser checkpoints pin down a desync more precisely; sparser ones make
// files smaller. At 12 bytes each, one per second costs about 720 B/minute.
const DefaultCheckpointEvery = core.TicksPerSecond

// Recorder runs a live game and captures everything needed to replay it.
// It is the only way input reaches the World during live play, so nothing
// the player does can be missed by the recording.
type Recorder struct {
	world *core.World
	every uint32
	rep   Replay
}

// NewRecorder starts a new game. every is the checkpoint interval in ticks.
func NewRecorder(cfg core.Config, seed uint64, every uint32) (*Recorder, error) {
	if every == 0 {
		return nil, fmt.Errorf("replay: checkpoint interval must be positive")
	}
	w, err := core.NewWorld(cfg, seed)
	if err != nil {
		return nil, err
	}
	return &Recorder{world: w, every: every, rep: Replay{Seed: seed, Config: cfg}}, nil
}

// World is the live game. Callers read it to render; only Step changes it.
func (r *Recorder) World() *core.World { return r.world }

// Step records in and advances the game one tick.
func (r *Recorder) Step(in core.Input) {
	in &= core.InputMask
	r.rep.Inputs = append(r.rep.Inputs, in)
	r.world.Step(in)
	if r.world.Tick%r.every == 0 {
		r.rep.Checkpoints = append(r.rep.Checkpoints, Checkpoint{Tick: r.world.Tick, Hash: r.world.Hash()})
	}
}

// Replay returns a copy of the recording so far, always ending with a
// checkpoint for the final tick so verification covers the whole run.
// The recorder can keep going afterwards.
func (r *Recorder) Replay() *Replay {
	out := Replay{
		Seed:        r.rep.Seed,
		Config:      r.rep.Config,
		Inputs:      append([]core.Input(nil), r.rep.Inputs...),
		Checkpoints: append([]Checkpoint(nil), r.rep.Checkpoints...),
	}
	if n := len(out.Checkpoints); r.world.Tick > 0 && (n == 0 || out.Checkpoints[n-1].Tick != r.world.Tick) {
		out.Checkpoints = append(out.Checkpoints, Checkpoint{Tick: r.world.Tick, Hash: r.world.Hash()})
	}
	return &out
}
