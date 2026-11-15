package core

import "github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/fixed"

// Input is one tick of player intent as a bitmask. A whole tick fits in one
// byte, which is what makes input recording so compact: a minute of play is
// 3600 bytes before compression, and far less after run-length encoding.
type Input uint8

// Input bits.
const (
	InputLeft Input = 1 << iota
	InputRight
	InputThrust
	InputFire

	// InputMask covers every defined bit; anything outside it is corrupt.
	InputMask = InputLeft | InputRight | InputThrust | InputFire
)

// Has reports whether every bit in b is set.
func (in Input) Has(b Input) bool { return in&b == b }

// Valid reports whether the input uses only defined bits.
func (in Input) Valid() bool { return in&^InputMask == 0 }

// Size is an asteroid's size class. Large rocks split into two mediums,
// mediums into two smalls, and smalls are destroyed outright.
type Size uint8

// Asteroid sizes.
const (
	Small Size = iota + 1
	Medium
	Large
)

// Radius is the collision radius of the size class.
func (s Size) Radius() fixed.F {
	switch s {
	case Large:
		return fixed.FromInt(40)
	case Medium:
		return fixed.FromInt(22)
	default:
		return fixed.FromInt(12)
	}
}

// Points is the score for destroying a rock of this size. Small rocks are
// worth the most because they are fast and hard to hit.
func (s Size) Points() uint32 {
	switch s {
	case Large:
		return 20
	case Medium:
		return 50
	default:
		return 100
	}
}

// Ship is the player's ship.
type Ship struct {
	Pos, Vel  fixed.Vec
	Heading   fixed.Angle
	Alive     bool
	Thrusting bool  // for the renderer's flame
	Cooldown  int32 // ticks until the next shot is allowed
	Invuln    int32 // ticks of spawn protection left
	RespawnIn int32 // ticks until a dead ship tries to respawn
}

// Bullet is a short-lived projectile.
type Bullet struct {
	Pos, Vel fixed.Vec
	TTL      int32
}

// Asteroid is a drifting rock. Shape seeds the renderer's procedural outline,
// so every rock looks different but the look is part of the replay.
type Asteroid struct {
	Pos, Vel fixed.Vec
	Size     Size
	Rot      fixed.Angle
	Spin     int8 // binary degrees per tick
	Shape    uint32
}

// EventKind classifies something that happened during a tick.
type EventKind uint8

// Event kinds.
const (
	EventShot EventKind = iota + 1
	EventAsteroidDestroyed
	EventShipDestroyed
	EventShipRespawned
	EventWaveStarted
	EventGameOver
)

// Event tells the presentation layer what happened this tick so it can play
// sounds and spawn particles. Events are output only: they are not hashed
// and never feed back into the simulation, so effects can't cause desyncs.
type Event struct {
	Kind EventKind
	Pos  fixed.Vec
	Size Size
}
