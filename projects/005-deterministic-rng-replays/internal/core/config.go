// Package core is the Asteroids simulation: a pure, deterministic function
// from (seed, config, input stream) to game state. It has no rendering,
// audio, file or clock dependencies, and it never touches floating point, so
// the same inputs reproduce the same game bit for bit on any machine.
package core

import (
	"errors"
	"fmt"

	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/fixed"
)

// TicksPerSecond is the fixed simulation rate. Every duration in Config is
// counted in ticks, never in seconds, so there is no dt to round.
const TicksPerSecond = 60

// Config holds every gameplay tuning value. All fields are fixed-size so the
// whole struct can be written into a replay header with encoding/binary: a
// replay carries the rules it was recorded under, and still plays back after
// the defaults are retuned.
type Config struct {
	Width, Height int32 // playfield size in pixels

	TurnRate    fixed.Angle // binary degrees per tick
	ThrustAccel fixed.F     // pixels/tick² while thrusting
	Drag        fixed.F     // velocity multiplier applied every tick
	MaxSpeed    fixed.F     // pixels/tick
	ShipRadius  fixed.F

	BulletSpeed  fixed.F // pixels/tick, added to the ship's velocity
	BulletLife   int32   // ticks
	FireCooldown int32   // ticks between shots
	MaxBullets   int32

	StartLives   int32
	RespawnDelay int32 // ticks before a dead ship tries to respawn
	InvulnTicks  int32 // spawn protection after (re)spawning

	WaveDelay        int32 // ticks between clearing a wave and the next one
	FirstWaveSize    int32
	MaxWaveSize      int32
	AsteroidMinSpeed fixed.F
	AsteroidMaxSpeed fixed.F
	SafeRadius       fixed.F // no asteroid spawns, and no ship respawns, this close
}

// DefaultConfig is tuned to feel like the 1979 arcade game at 60 Hz.
func DefaultConfig() Config {
	return Config{
		Width:            800,
		Height:           600,
		TurnRate:         3,
		ThrustAccel:      fixed.FromRatio(1, 8),
		Drag:             fixed.FromRatio(99, 100),
		MaxSpeed:         fixed.FromInt(6),
		ShipRadius:       fixed.FromInt(12),
		BulletSpeed:      fixed.FromInt(8),
		BulletLife:       50,
		FireCooldown:     8,
		MaxBullets:       6,
		StartLives:       3,
		RespawnDelay:     90,
		InvulnTicks:      120,
		WaveDelay:        120,
		FirstWaveSize:    4,
		MaxWaveSize:      11,
		AsteroidMinSpeed: fixed.FromRatio(1, 2),
		AsteroidMaxSpeed: fixed.FromRatio(3, 2),
		SafeRadius:       fixed.FromInt(120),
	}
}

// ErrInvalidConfig is wrapped by every error Validate returns.
var ErrInvalidConfig = errors.New("invalid config")

// Validate rejects settings the simulation can't run with. Replays are
// untrusted input, so the decoder calls this too.
func (c Config) Validate() error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidConfig, fmt.Sprintf(format, args...))
	}
	switch {
	case c.Width < 100 || c.Height < 100 || c.Width > 8000 || c.Height > 8000:
		return bad("playfield %dx%d must be between 100 and 8000 on each side", c.Width, c.Height)
	case c.TurnRate == 0:
		return bad("TurnRate must be positive")
	case c.ThrustAccel <= 0 || c.MaxSpeed <= 0 || c.BulletSpeed <= 0:
		return bad("ThrustAccel, MaxSpeed and BulletSpeed must be positive")
	case c.Drag <= 0 || c.Drag > fixed.One:
		return bad("Drag must be in (0, 1], got %v", c.Drag)
	case c.ShipRadius <= 0 || c.SafeRadius < 0:
		return bad("ShipRadius must be positive and SafeRadius non-negative")
	case c.BulletLife <= 0 || c.FireCooldown < 0 || c.MaxBullets <= 0:
		return bad("BulletLife and MaxBullets must be positive, FireCooldown non-negative")
	case c.StartLives <= 0 || c.RespawnDelay < 0 || c.InvulnTicks < 0 || c.WaveDelay < 0:
		return bad("StartLives must be positive and delays non-negative")
	case c.FirstWaveSize <= 0 || c.MaxWaveSize < c.FirstWaveSize || c.MaxWaveSize > 64:
		return bad("wave sizes must satisfy 0 < FirstWaveSize <= MaxWaveSize <= 64")
	case c.AsteroidMinSpeed <= 0 || c.AsteroidMaxSpeed < c.AsteroidMinSpeed:
		return bad("asteroid speeds must satisfy 0 < min <= max")
	}
	return nil
}
