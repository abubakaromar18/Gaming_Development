package core

import (
	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/fixed"
	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/rng"
)

// World is the complete simulation state. Everything that influences the
// future lives here, including the RNG, so cloning a World captures a moment
// that can be resumed exactly. That is what replay seeking is built on.
type World struct {
	Cfg  Config
	Tick uint32
	Rng  rng.Rand

	Ship      Ship
	Bullets   []Bullet
	Asteroids []Asteroid

	Score      uint32
	Lives      int32
	Wave       int32
	NextWaveIn int32 // ticks until the next wave, counting only while the field is empty
	Over       bool

	// Events holds what happened during the most recent Step.
	Events []Event
}

// NewWorld starts a game: ship in the centre, first wave on the field.
func NewWorld(cfg Config, seed uint64) (*World, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	w := &World{
		Cfg:   cfg,
		Rng:   rng.New(seed),
		Lives: cfg.StartLives,
	}
	w.spawnShip()
	w.startWave()
	return w, nil
}

// Clone returns a deep copy. The copy shares no slices with the original.
func (w *World) Clone() *World {
	c := *w
	c.Bullets = append([]Bullet(nil), w.Bullets...)
	c.Asteroids = append([]Asteroid(nil), w.Asteroids...)
	c.Events = nil
	return &c
}

func (w *World) width() fixed.F  { return fixed.FromInt(int(w.Cfg.Width)) }
func (w *World) height() fixed.F { return fixed.FromInt(int(w.Cfg.Height)) }
func (w *World) center() fixed.Vec {
	return fixed.Vec{X: w.width() / 2, Y: w.height() / 2}
}

// delta is the shortest vector from a to b across the wrapping playfield.
func (w *World) delta(a, b fixed.Vec) fixed.Vec {
	return fixed.WrapDeltaVec(a, b, w.width(), w.height())
}

func (w *World) wrap(p fixed.Vec) fixed.Vec { return fixed.WrapVec(p, w.width(), w.height()) }

func (w *World) emit(kind EventKind, pos fixed.Vec, size Size) {
	w.Events = append(w.Events, Event{Kind: kind, Pos: pos, Size: size})
}

// Step advances the simulation by exactly one tick. The order of operations
// is part of the replay contract: reordering any of these phases changes
// which bullet hits which rock, so it would invalidate saved replays.
func (w *World) Step(in Input) {
	w.Events = w.Events[:0]
	w.updateShip(in)
	w.updateBullets()
	w.updateAsteroids()
	w.collideBullets()
	w.collideShip()
	w.updateWave()
	w.Tick++
}

func (w *World) updateShip(in Input) {
	s := &w.Ship
	if !s.Alive {
		w.tryRespawn()
		return
	}
	if in.Has(InputLeft) {
		s.Heading -= w.Cfg.TurnRate
	}
	if in.Has(InputRight) {
		s.Heading += w.Cfg.TurnRate
	}
	dir := fixed.FromAngle(s.Heading)
	s.Thrusting = in.Has(InputThrust)
	if s.Thrusting {
		s.Vel = s.Vel.Add(dir.Scale(w.Cfg.ThrustAccel))
	}
	s.Vel = s.Vel.Scale(w.Cfg.Drag)
	if s.Vel.LenSq() > fixed.RawSq(w.Cfg.MaxSpeed) {
		s.Vel = s.Vel.Scale(w.Cfg.MaxSpeed.Div(s.Vel.Len()))
	}
	s.Pos = w.wrap(s.Pos.Add(s.Vel))

	if s.Cooldown > 0 {
		s.Cooldown--
	}
	if s.Invuln > 0 {
		s.Invuln--
	}
	if in.Has(InputFire) && s.Cooldown == 0 && int32(len(w.Bullets)) < w.Cfg.MaxBullets {
		nose := w.wrap(s.Pos.Add(dir.Scale(w.Cfg.ShipRadius)))
		w.Bullets = append(w.Bullets, Bullet{
			Pos: nose,
			Vel: s.Vel.Add(dir.Scale(w.Cfg.BulletSpeed)),
			TTL: w.Cfg.BulletLife,
		})
		s.Cooldown = w.Cfg.FireCooldown
		w.emit(EventShot, nose, 0)
	}
}

// tryRespawn counts down, then waits until the centre is clear so the
// player never respawns inside a rock.
func (w *World) tryRespawn() {
	if w.Over {
		return
	}
	s := &w.Ship
	if s.RespawnIn > 0 {
		s.RespawnIn--
		return
	}
	if w.clearOfAsteroids(w.center(), w.Cfg.SafeRadius) {
		w.spawnShip()
		w.emit(EventShipRespawned, s.Pos, 0)
	}
}

func (w *World) spawnShip() {
	w.Ship = Ship{
		Pos:     w.center(),
		Heading: fixed.Angle270, // pointing up
		Alive:   true,
		Invuln:  w.Cfg.InvulnTicks,
	}
}

func (w *World) clearOfAsteroids(p fixed.Vec, radius fixed.F) bool {
	for _, a := range w.Asteroids {
		if w.delta(p, a.Pos).LenSq() < fixed.RawSq(radius+a.Size.Radius()) {
			return false
		}
	}
	return true
}

func (w *World) updateBullets() {
	live := w.Bullets[:0]
	for _, b := range w.Bullets {
		b.TTL--
		if b.TTL <= 0 {
			continue
		}
		b.Pos = w.wrap(b.Pos.Add(b.Vel))
		live = append(live, b)
	}
	w.Bullets = live
}

func (w *World) updateAsteroids() {
	for i := range w.Asteroids {
		a := &w.Asteroids[i]
		a.Pos = w.wrap(a.Pos.Add(a.Vel))
		a.Rot += fixed.Angle(a.Spin)
	}
}

// collideBullets tests every bullet against every rock, in slice order.
// Fragments are collected separately and added after the loop, so a bullet
// can't hit a fragment spawned by another bullet in the same tick.
func (w *World) collideBullets() {
	var fragments []Asteroid
	liveBullets := w.Bullets[:0]
	for _, b := range w.Bullets {
		hit := -1
		for i, a := range w.Asteroids {
			if w.delta(b.Pos, a.Pos).LenSq() < fixed.RawSq(a.Size.Radius()) {
				hit = i
				break
			}
		}
		if hit < 0 {
			liveBullets = append(liveBullets, b)
			continue
		}
		rock := w.Asteroids[hit]
		w.Asteroids = append(w.Asteroids[:hit], w.Asteroids[hit+1:]...)
		w.Score += rock.Size.Points()
		fragments = append(fragments, w.split(rock)...)
		w.emit(EventAsteroidDestroyed, rock.Pos, rock.Size)
	}
	w.Bullets = liveBullets
	w.Asteroids = append(w.Asteroids, fragments...)
}

func (w *World) collideShip() {
	s := &w.Ship
	if !s.Alive || s.Invuln > 0 {
		return
	}
	for i, a := range w.Asteroids {
		if w.delta(s.Pos, a.Pos).LenSq() >= fixed.RawSq(w.Cfg.ShipRadius+a.Size.Radius()) {
			continue
		}
		w.Asteroids = append(w.Asteroids[:i], w.Asteroids[i+1:]...)
		w.Asteroids = append(w.Asteroids, w.split(a)...)
		w.emit(EventAsteroidDestroyed, a.Pos, a.Size)
		w.killShip()
		return
	}
}

func (w *World) killShip() {
	s := &w.Ship
	w.emit(EventShipDestroyed, s.Pos, 0)
	*s = Ship{Pos: s.Pos, RespawnIn: w.Cfg.RespawnDelay}
	w.Lives--
	if w.Lives <= 0 {
		w.Over = true
		w.emit(EventGameOver, s.Pos, 0)
	}
}

// split breaks a rock into two smaller, faster ones flying off at random
// angles. This is one of the two places gameplay consumes randomness; the
// other is wave spawning.
func (w *World) split(parent Asteroid) []Asteroid {
	if parent.Size == Small {
		return nil
	}
	out := make([]Asteroid, 2)
	for i := range out {
		size := parent.Size - 1
		out[i] = w.newAsteroid(parent.Pos, size)
		out[i].Vel = out[i].Vel.Add(parent.Vel.Scale(fixed.FromRatio(1, 2)))
	}
	return out
}

// newAsteroid rolls a random heading, speed, spin and outline. Smaller rocks
// get a speed bonus, so later fragments are harder to hit.
func (w *World) newAsteroid(pos fixed.Vec, size Size) Asteroid {
	speed := w.Rng.FixedRange(w.Cfg.AsteroidMinSpeed, w.Cfg.AsteroidMaxSpeed)
	speed = speed.Mul(fixed.FromRatio(int(7-size), 4)) // large ×1, medium ×1.25, small ×1.5
	return Asteroid{
		Pos:   pos,
		Vel:   fixed.FromAngle(w.Rng.Angle()).Scale(speed),
		Size:  size,
		Rot:   w.Rng.Angle(),
		Spin:  int8(w.Rng.Range(-3, 3)),
		Shape: w.Rng.Uint32(),
	}
}

func (w *World) updateWave() {
	if w.Over || len(w.Asteroids) > 0 {
		return
	}
	if w.NextWaveIn > 0 {
		w.NextWaveIn--
		return
	}
	w.startWave()
}

// startWave spawns large rocks along the playfield edges (which, on a
// torus, are all one seam), re-rolling any spot too close to the ship.
func (w *World) startWave() {
	w.Wave++
	count := w.Cfg.FirstWaveSize + w.Wave - 1
	if count > w.Cfg.MaxWaveSize {
		count = w.Cfg.MaxWaveSize
	}
	const maxTries = 16
	for i := int32(0); i < count; i++ {
		var pos fixed.Vec
		for try := 0; try < maxTries; try++ {
			pos = w.edgePoint()
			if w.delta(pos, w.Ship.Pos).LenSq() >= fixed.RawSq(w.Cfg.SafeRadius+Large.Radius()) {
				break
			}
		}
		w.Asteroids = append(w.Asteroids, w.newAsteroid(pos, Large))
	}
	w.NextWaveIn = w.Cfg.WaveDelay
	w.emit(EventWaveStarted, w.center(), 0)
}

func (w *World) edgePoint() fixed.Vec {
	if w.Rng.Chance(1, 2) {
		return fixed.Vec{X: w.Rng.FixedRange(0, w.width())}
	}
	return fixed.Vec{Y: w.Rng.FixedRange(0, w.height())}
}
