package core

import "github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/fixed"

// Hash returns a 64-bit FNV-1a digest of every field that affects the future
// of the simulation. Replays store these at checkpoints; if playback ever
// produces a different hash, the run has desynced and the player can say
// exactly which tick it happened on instead of silently showing a wrong game.
//
// Fields are fed in a fixed order with fixed widths, never via reflection or
// map iteration, so the digest is stable across Go versions. Events are
// excluded: they are output, not state.
func (w *World) Hash() uint64 {
	h := hasher(14695981039346656037)
	h.u32(w.Tick)
	for _, s := range w.Rng.State() {
		h.u64(s)
	}
	s := w.Ship
	h.vec(s.Pos)
	h.vec(s.Vel)
	h.u8(uint8(s.Heading))
	h.bool(s.Alive)
	h.bool(s.Thrusting)
	h.i32(s.Cooldown)
	h.i32(s.Invuln)
	h.i32(s.RespawnIn)

	h.u32(uint32(len(w.Bullets)))
	for _, b := range w.Bullets {
		h.vec(b.Pos)
		h.vec(b.Vel)
		h.i32(b.TTL)
	}
	h.u32(uint32(len(w.Asteroids)))
	for _, a := range w.Asteroids {
		h.vec(a.Pos)
		h.vec(a.Vel)
		h.u8(uint8(a.Size))
		h.u8(uint8(a.Rot))
		h.u8(uint8(a.Spin))
		h.u32(a.Shape)
	}
	h.u32(w.Score)
	h.i32(w.Lives)
	h.i32(w.Wave)
	h.i32(w.NextWaveIn)
	h.bool(w.Over)
	return uint64(h)
}

type hasher uint64

func (h *hasher) u8(b uint8) {
	*h ^= hasher(b)
	*h *= 1099511628211
}

func (h *hasher) u32(v uint32) {
	for i := 0; i < 32; i += 8 {
		h.u8(uint8(v >> i))
	}
}

func (h *hasher) u64(v uint64) {
	for i := 0; i < 64; i += 8 {
		h.u8(uint8(v >> i))
	}
}

func (h *hasher) i32(v int32) { h.u32(uint32(v)) }

func (h *hasher) vec(v fixed.Vec) {
	h.i32(int32(v.X))
	h.i32(int32(v.Y))
}

func (h *hasher) bool(b bool) {
	if b {
		h.u8(1)
	} else {
		h.u8(0)
	}
}
