package main

import (
	"math"
	"math/rand/v2"

	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/core"
)

// effects owns purely cosmetic state: explosion particles and the
// starfield. It has its own RNG, seeded from the clock, deliberately
// separate from the simulation's. If cosmetics drew from the gameplay RNG,
// then turning particles off, or a frame dropping a particle, would shift
// every later asteroid split and desync every replay. Separate streams make
// "cosmetic" a guarantee instead of a convention.
type effects struct {
	rng       *rand.Rand
	particles []particle
	stars     []star
}

type particle struct {
	x, y, vx, vy float64
	life, max    int
}

type star struct {
	x, y       float64
	brightness uint8
}

func newEffects(cfg core.Config, seed uint64) *effects {
	fx := &effects{rng: rand.New(rand.NewPCG(seed, seed^0x5bd1e995))}
	for i := 0; i < 120; i++ {
		fx.stars = append(fx.stars, star{
			x:          fx.rng.Float64() * float64(cfg.Width),
			y:          fx.rng.Float64() * float64(cfg.Height),
			brightness: uint8(40 + fx.rng.IntN(120)),
		})
	}
	return fx
}

// reset drops in-flight particles, e.g. after a seek where they would
// belong to a different moment.
func (fx *effects) reset() { fx.particles = fx.particles[:0] }

// consume turns simulation events into particle bursts.
func (fx *effects) consume(events []core.Event) {
	for _, e := range events {
		x, y := e.Pos.X.Float64(), e.Pos.Y.Float64()
		switch e.Kind {
		case core.EventAsteroidDestroyed:
			fx.burst(x, y, 6*int(e.Size)+4, 1.5+float64(e.Size)*0.5)
		case core.EventShipDestroyed:
			fx.burst(x, y, 40, 3.5)
		}
	}
}

func (fx *effects) burst(x, y float64, count int, speed float64) {
	for i := 0; i < count; i++ {
		angle := fx.rng.Float64() * 2 * math.Pi
		v := speed * (0.3 + 0.7*fx.rng.Float64())
		life := 20 + fx.rng.IntN(30)
		fx.particles = append(fx.particles, particle{
			x: x, y: y,
			vx: math.Cos(angle) * v, vy: math.Sin(angle) * v,
			life: life, max: life,
		})
	}
}

func (fx *effects) update() {
	live := fx.particles[:0]
	for _, p := range fx.particles {
		p.life--
		if p.life <= 0 {
			continue
		}
		p.x += p.vx
		p.y += p.vy
		p.vx *= 0.96
		p.vy *= 0.96
		live = append(live, p)
	}
	fx.particles = live
}
