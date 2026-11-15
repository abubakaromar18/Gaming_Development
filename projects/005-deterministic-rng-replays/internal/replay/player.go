package replay

import (
	"errors"
	"fmt"
	"sort"

	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/core"
)

// DesyncError reports the first checkpoint where playback disagreed with
// the recording. Everything before the previous checkpoint is known good,
// which narrows the search for the nondeterministic code.
type DesyncError struct {
	Tick      uint32
	Want, Got uint64
}

func (e *DesyncError) Error() string {
	return fmt.Sprintf("replay desynced at tick %d: recorded hash %#016x, playback %#016x", e.Tick, e.Want, e.Got)
}

// ErrEnded is returned by Step once every recorded tick has been played.
var ErrEnded = errors.New("replay: end of recording")

// DefaultKeyframeEvery caches a world snapshot every five seconds of play.
const DefaultKeyframeEvery = 5 * core.TicksPerSecond

// Player re-simulates a replay. Input replays can't jump straight to a tick,
// because state at tick N only exists by simulating ticks 0..N-1. So the
// player caches keyframes (World clones) as it goes: seeking restores the
// nearest keyframe at or before the target and simulates the rest, which
// bounds any seek to KeyframeEvery ticks of work once that region has been
// visited.
type Player struct {
	rep         *Replay
	world       *core.World
	checkpoints map[uint32]uint64
	keyframes   []*core.World // keyframes[i] is the world at tick i*every
	every       uint32
}

// NewPlayer prepares r for playback. keyframeEvery is in ticks.
func NewPlayer(r *Replay, keyframeEvery uint32) (*Player, error) {
	if keyframeEvery == 0 {
		return nil, fmt.Errorf("replay: keyframe interval must be positive")
	}
	if err := r.validate(); err != nil {
		return nil, err
	}
	w, err := core.NewWorld(r.Config, r.Seed)
	if err != nil {
		return nil, err
	}
	p := &Player{
		rep:         r,
		world:       w,
		checkpoints: make(map[uint32]uint64, len(r.Checkpoints)),
		keyframes:   []*core.World{w.Clone()},
		every:       keyframeEvery,
	}
	for _, c := range r.Checkpoints {
		p.checkpoints[c.Tick] = c.Hash
	}
	return p, nil
}

// World is the current playback state, for rendering.
func (p *Player) World() *core.World { return p.world }

// Replay is the recording being played.
func (p *Player) Replay() *Replay { return p.rep }

// Tick is the number of ticks played so far.
func (p *Player) Tick() uint32 { return p.world.Tick }

// Len is the total number of recorded ticks.
func (p *Player) Len() uint32 { return uint32(len(p.rep.Inputs)) }

// Done reports whether playback reached the end.
func (p *Player) Done() bool { return p.Tick() >= p.Len() }

// Input returns the recorded input for the next tick, so the renderer can
// show which keys were held. At the end it returns 0.
func (p *Player) Input() core.Input {
	if p.Done() {
		return 0
	}
	return p.rep.Inputs[p.Tick()]
}

// Step plays one tick and verifies the checkpoint for it, if there is one.
func (p *Player) Step() error {
	if p.Done() {
		return ErrEnded
	}
	p.world.Step(p.rep.Inputs[p.world.Tick])
	t := p.world.Tick
	if t%p.every == 0 && int(t/p.every) == len(p.keyframes) {
		p.keyframes = append(p.keyframes, p.world.Clone())
	}
	if want, ok := p.checkpoints[t]; ok {
		if got := p.world.Hash(); got != want {
			return &DesyncError{Tick: t, Want: want, Got: got}
		}
	}
	return nil
}

// Seek moves playback to tick target (clamped to the recording), forward or
// backward. Events from the skipped ticks are discarded; after Seek the
// world holds only the events of the final simulated tick.
func (p *Player) Seek(target uint32) error {
	if target > p.Len() {
		target = p.Len()
	}
	// The latest cached keyframe at or before target. Restore it when going
	// backward (the past can't be un-simulated) or when it lies ahead of the
	// current tick (skipping work); otherwise just simulate forward from here.
	i := sort.Search(len(p.keyframes), func(i int) bool { return uint32(i)*p.every > target }) - 1
	if base := p.keyframes[i]; target < p.Tick() || base.Tick > p.Tick() {
		p.world = base.Clone()
	}
	for p.Tick() < target {
		if err := p.Step(); err != nil {
			return err
		}
	}
	return nil
}

// Verify plays r from start to finish and checks every checkpoint. It
// returns the final world so callers can report the score and hash.
func Verify(r *Replay) (*core.World, error) {
	p, err := NewPlayer(r, DefaultKeyframeEvery)
	if err != nil {
		return nil, err
	}
	for !p.Done() {
		if err := p.Step(); err != nil {
			return p.World(), err
		}
	}
	return p.World(), nil
}
