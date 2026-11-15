// Package bot is a simple autopilot that plays Asteroids by reading the
// World. It exists to generate long, varied replays for tests, the demo
// command and the game's attract mode, without a human at the keyboard.
//
// The bot is deliberately just another input source: the simulation can't
// tell it apart from a keyboard, and replays record its output like any
// player's. It uses only integer math, so it is deterministic too.
package bot

import (
	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/core"
	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/fixed"
)

// Tuning for the autopilot. A tighter aim cone shoots less but wastes fewer
// bullets; a larger cruise distance makes the bot more cautious.
var (
	aimCone        = fixed.FromRatio(1, 10) // |cross| < dot * aimCone counts as "on target"
	cruiseDistance = fixed.FromInt(220)     // thrust toward targets farther than this
	panicDistance  = fixed.FromInt(90)      // burn away from anything this close
)

// Decide returns the bot's input for the next tick.
func Decide(w *core.World) core.Input {
	s := w.Ship
	if !s.Alive || len(w.Asteroids) == 0 {
		return 0
	}
	width := fixed.FromInt(int(w.Cfg.Width))
	height := fixed.FromInt(int(w.Cfg.Height))

	// Pick the nearest rock, measured across the wrapping edges.
	var target fixed.Vec
	best := int64(-1)
	for _, a := range w.Asteroids {
		d := fixed.WrapDeltaVec(s.Pos, a.Pos, width, height)
		if dist := d.LenSq(); best < 0 || dist < best {
			best, target = dist, d
		}
	}
	dir := fixed.FromAngle(s.Heading)

	// cross > 0 means the target is clockwise of the heading (y grows down),
	// which is the direction InputRight turns.
	cross := dir.X.Mul(target.Y) - dir.Y.Mul(target.X)
	dot := dir.X.Mul(target.X) + dir.Y.Mul(target.Y)

	var in core.Input
	aimed := dot > 0 && cross.Abs() < dot.Mul(aimCone)
	switch {
	case aimed:
	case cross > 0:
		in |= core.InputRight
	default:
		in |= core.InputLeft
	}
	if aimed {
		in |= core.InputFire
		if best > fixed.RawSq(cruiseDistance) {
			in |= core.InputThrust
		}
	}
	// Facing away from a rock that is about to hit us: thrust to escape.
	if best < fixed.RawSq(panicDistance) && dot < 0 {
		in |= core.InputThrust
	}
	return in
}
