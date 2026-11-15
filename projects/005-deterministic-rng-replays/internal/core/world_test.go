package core

import (
	"errors"
	"testing"

	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/fixed"
)

func newTestWorld(t *testing.T, seed uint64) *World {
	t.Helper()
	w, err := NewWorld(DefaultConfig(), seed)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// emptyWorld has no rocks and a wave timer that never fires, for tests that
// set up a precise situation by hand.
func emptyWorld(t *testing.T) *World {
	w := newTestWorld(t, 1)
	w.Asteroids = nil
	w.NextWaveIn = 1 << 30
	w.Ship.Invuln = 0
	return w
}

// scriptedInput is a fixed, varied input pattern independent of game state.
func scriptedInput(tick int) Input {
	switch (tick / 40) % 5 {
	case 0:
		return InputThrust | InputFire
	case 1:
		return InputLeft | InputFire
	case 2:
		return InputRight | InputThrust
	case 3:
		return InputFire
	default:
		return 0
	}
}

func TestConfigValidation(t *testing.T) {
	if err := DefaultConfig().Validate(); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
	breakers := map[string]func(*Config){
		"tiny field":    func(c *Config) { c.Width = 10 },
		"no turning":    func(c *Config) { c.TurnRate = 0 },
		"drag > 1":      func(c *Config) { c.Drag = 2 * fixed.One },
		"no bullets":    func(c *Config) { c.MaxBullets = 0 },
		"zero lives":    func(c *Config) { c.StartLives = 0 },
		"inverted wave": func(c *Config) { c.MaxWaveSize = c.FirstWaveSize - 1 },
		"slow > fast":   func(c *Config) { c.AsteroidMinSpeed = c.AsteroidMaxSpeed + 1 },
	}
	for name, breakIt := range breakers {
		c := DefaultConfig()
		breakIt(&c)
		if err := c.Validate(); !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("%s: got %v, want ErrInvalidConfig", name, err)
		}
		if _, err := NewWorld(c, 1); err == nil {
			t.Errorf("%s: NewWorld accepted an invalid config", name)
		}
	}
}

func TestNewWorldStartsFirstWaveAwayFromShip(t *testing.T) {
	for seed := uint64(0); seed < 50; seed++ {
		w := newTestWorld(t, seed)
		if w.Wave != 1 || len(w.Asteroids) != int(w.Cfg.FirstWaveSize) {
			t.Fatalf("seed %d: wave %d with %d rocks", seed, w.Wave, len(w.Asteroids))
		}
		for _, a := range w.Asteroids {
			if a.Size != Large {
				t.Fatalf("first wave should be all large rocks")
			}
			if w.delta(w.Ship.Pos, a.Pos).Len() < w.Cfg.SafeRadius {
				t.Fatalf("seed %d: rock spawned on top of the ship", seed)
			}
		}
	}
}

func TestInputValidity(t *testing.T) {
	if !(InputLeft | InputFire).Valid() || Input(0x80).Valid() {
		t.Fatal("Valid should accept defined bits only")
	}
	if !(InputLeft | InputFire).Has(InputFire) || InputLeft.Has(InputFire) {
		t.Fatal("Has wrong")
	}
}

func TestShipTurnsThrustsAndWraps(t *testing.T) {
	w := emptyWorld(t)
	start := w.Ship.Heading
	w.Step(InputRight)
	if w.Ship.Heading != start+w.Cfg.TurnRate {
		t.Fatal("right should turn clockwise")
	}
	w.Step(InputLeft)
	if w.Ship.Heading != start {
		t.Fatal("left should undo right")
	}
	// Pointing up (heading 270°) and thrusting moves toward y = 0.
	y0 := w.Ship.Pos.Y
	for i := 0; i < 30; i++ {
		w.Step(InputThrust)
	}
	if w.Ship.Pos.Y >= y0 || w.Ship.Vel.Y >= 0 {
		t.Fatalf("thrusting up should move up: pos %v vel %v", w.Ship.Pos.Y, w.Ship.Vel.Y)
	}
	// Keep thrusting: speed is capped, and the ship wraps past the top edge.
	for i := 0; i < 400; i++ {
		w.Step(InputThrust)
		if w.Ship.Vel.Len() > w.Cfg.MaxSpeed+2 {
			t.Fatalf("speed %v exceeds max %v", w.Ship.Vel.Len(), w.Cfg.MaxSpeed)
		}
		p := w.Ship.Pos
		if p.Y < 0 || p.Y >= w.height() || p.X < 0 || p.X >= w.width() {
			t.Fatalf("ship left the torus: %v", p)
		}
	}
}

func TestDragSlowsDriftingShip(t *testing.T) {
	w := emptyWorld(t)
	w.Ship.Vel = fixed.V(4, 0)
	for i := 0; i < 120; i++ {
		w.Step(0)
	}
	if w.Ship.Vel.X >= fixed.FromInt(2) {
		t.Fatalf("drag should have bled off speed, vel = %v", w.Ship.Vel.X)
	}
}

func TestFiringRespectsCooldownAndLimit(t *testing.T) {
	w := emptyWorld(t)
	w.Step(InputFire)
	if len(w.Bullets) != 1 || len(w.Events) != 1 || w.Events[0].Kind != EventShot {
		t.Fatalf("first shot: %d bullets, events %v", len(w.Bullets), w.Events)
	}
	w.Step(InputFire)
	if len(w.Bullets) != 1 {
		t.Fatal("cooldown should block an immediate second shot")
	}
	w.Cfg.FireCooldown = 0
	for i := 0; i < 20; i++ {
		w.Step(InputFire)
	}
	if len(w.Bullets) != int(w.Cfg.MaxBullets) {
		t.Fatalf("%d bullets on screen, limit is %d", len(w.Bullets), w.Cfg.MaxBullets)
	}
}

func TestBulletsExpire(t *testing.T) {
	w := emptyWorld(t)
	w.Step(InputFire)
	for i := int32(0); i < w.Cfg.BulletLife; i++ {
		w.Step(0)
	}
	if len(w.Bullets) != 0 {
		t.Fatal("bullet outlived BulletLife")
	}
}

func TestBulletSplitsLargeRockIntoTwoMediums(t *testing.T) {
	w := emptyWorld(t)
	// Park a rock just above the ship's nose and fire straight up at it.
	target := w.Ship.Pos.Add(fixed.V(0, -60))
	w.Asteroids = []Asteroid{{Pos: target, Size: Large}}
	w.Step(InputFire)
	var destroyed bool
	for i := 0; i < 10 && !destroyed; i++ {
		for _, e := range w.Events {
			destroyed = destroyed || e.Kind == EventAsteroidDestroyed
		}
		if !destroyed {
			w.Step(0)
		}
	}
	if !destroyed {
		t.Fatal("bullet never hit the rock")
	}
	if w.Score != Large.Points() {
		t.Fatalf("score %d, want %d", w.Score, Large.Points())
	}
	if len(w.Asteroids) != 2 || w.Asteroids[0].Size != Medium || w.Asteroids[1].Size != Medium {
		t.Fatalf("want two mediums, got %+v", w.Asteroids)
	}
	if len(w.Bullets) != 0 {
		t.Fatal("bullet should be consumed by the hit")
	}
}

func TestSmallRockIsDestroyedOutright(t *testing.T) {
	w := emptyWorld(t)
	rock := Asteroid{Pos: w.Ship.Pos, Size: Small}
	if frags := w.split(rock); frags != nil {
		t.Fatalf("small rock split into %d", len(frags))
	}
}

func TestCollisionAcrossTheWrapSeam(t *testing.T) {
	w := emptyWorld(t)
	w.Ship.Pos = fixed.V(2, 300)
	w.Asteroids = []Asteroid{{Pos: fixed.V(int(w.Cfg.Width)-10, 300), Size: Small}}
	w.Step(0)
	if w.Ship.Alive {
		t.Fatal("rock on the far edge is 12px away across the seam and should hit")
	}
}

func TestShipDeathRespawnAndGameOver(t *testing.T) {
	w := emptyWorld(t)
	rockOnShip := func() { w.Asteroids = []Asteroid{{Pos: w.Ship.Pos, Size: Small}} }

	rockOnShip()
	w.Step(0)
	if w.Ship.Alive || w.Lives != w.Cfg.StartLives-1 {
		t.Fatalf("ship should die: alive=%v lives=%d", w.Ship.Alive, w.Lives)
	}
	// A rock sitting in the centre blocks respawn indefinitely.
	w.Asteroids = []Asteroid{{Pos: w.center(), Size: Large}}
	for i := int32(0); i < w.Cfg.RespawnDelay+30; i++ {
		w.Step(0)
		w.Asteroids[0].Pos = w.center()
	}
	if w.Ship.Alive {
		t.Fatal("ship respawned inside a rock")
	}
	w.Asteroids = nil
	w.Step(0)
	if !w.Ship.Alive || w.Ship.Invuln != w.Cfg.InvulnTicks {
		t.Fatal("ship should respawn once the centre is clear, with spawn protection")
	}
	// Spawn protection ignores rocks.
	rockOnShip()
	w.Step(0)
	if !w.Ship.Alive {
		t.Fatal("invulnerable ship died")
	}

	for w.Lives > 0 {
		w.Ship.Alive, w.Ship.Invuln = true, 0
		rockOnShip()
		w.Step(0)
	}
	if !w.Over || w.Events[len(w.Events)-1].Kind != EventGameOver {
		t.Fatal("losing the last life should end the game")
	}
	w.Asteroids = nil
	for i := 0; i < 500; i++ {
		w.Step(InputFire)
	}
	if w.Ship.Alive || len(w.Bullets) != 0 {
		t.Fatal("nothing should respawn or fire after game over")
	}
}

func TestNextWaveIsBiggerAfterDelay(t *testing.T) {
	w := newTestWorld(t, 3)
	w.Asteroids = nil
	for i := int32(0); i < w.Cfg.WaveDelay; i++ {
		w.Step(0)
		if w.Wave != 1 {
			t.Fatalf("wave started early at tick %d", i)
		}
	}
	w.Step(0)
	if w.Wave != 2 || len(w.Asteroids) != int(w.Cfg.FirstWaveSize)+1 {
		t.Fatalf("wave %d with %d rocks", w.Wave, len(w.Asteroids))
	}
}

// The heart of the project: the same seed and inputs give the same game.
func TestSameSeedAndInputsProduceIdenticalRuns(t *testing.T) {
	a, b := newTestWorld(t, 2026), newTestWorld(t, 2026)
	for tick := 0; tick < 5000; tick++ {
		in := scriptedInput(tick)
		a.Step(in)
		b.Step(in)
		if a.Hash() != b.Hash() {
			t.Fatalf("runs diverged at tick %d", tick)
		}
	}
}

func TestDifferentSeedsProduceDifferentGames(t *testing.T) {
	a, b := newTestWorld(t, 1), newTestWorld(t, 2)
	if a.Hash() == b.Hash() {
		t.Fatal("different seeds should lay out different first waves")
	}
}

func TestCloneIsIndependentAndResumable(t *testing.T) {
	w := newTestWorld(t, 77)
	for i := 0; i < 300; i++ {
		w.Step(scriptedInput(i))
	}
	snap := w.Clone()
	if snap.Hash() != w.Hash() {
		t.Fatal("clone should hash equal")
	}
	for i := 300; i < 900; i++ {
		w.Step(scriptedInput(i))
	}
	if snap.Tick != 300 {
		t.Fatal("stepping the original changed the clone")
	}
	for i := 300; i < 900; i++ {
		snap.Step(scriptedInput(i))
	}
	if snap.Hash() != w.Hash() {
		t.Fatal("resuming from a clone should reproduce the original run")
	}
}

func TestHashCoversRngAndEntities(t *testing.T) {
	w := newTestWorld(t, 5)
	base := w.Hash()
	mutations := map[string]func(*World){
		"rng":      func(c *World) { c.Rng.Uint64() },
		"asteroid": func(c *World) { c.Asteroids[0].Shape++ },
		"ship":     func(c *World) { c.Ship.Heading++ },
		"score":    func(c *World) { c.Score++ },
		"bullet":   func(c *World) { c.Bullets = append(c.Bullets, Bullet{TTL: 1}) },
	}
	for name, mutate := range mutations {
		c := w.Clone()
		mutate(c)
		if c.Hash() == base {
			t.Errorf("hash ignores %s", name)
		}
	}
	c := w.Clone()
	c.Events = append(c.Events, Event{Kind: EventShot})
	if c.Hash() != base {
		t.Error("events are presentation output and must not affect the hash")
	}
}
