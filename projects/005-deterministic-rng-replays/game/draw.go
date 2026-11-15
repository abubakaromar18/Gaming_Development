package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/core"
	"github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/internal/fixed"
)

var (
	colBackground = color.RGBA{8, 10, 20, 255}
	colLine       = color.RGBA{220, 230, 255, 255}
	colFlame      = color.RGBA{255, 170, 60, 255}
	colBullet     = color.RGBA{255, 255, 200, 255}
	colHUDDim     = color.RGBA{70, 80, 110, 255}
	colHUDLit     = color.RGBA{120, 220, 140, 255}
	colDesync     = color.RGBA{255, 80, 80, 255}
)

// Draw renders the current world. It only reads state; the simulation's
// fixed-point values become floats here and nowhere else.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(colBackground)
	w := g.world()
	for _, s := range g.fx.stars {
		c := color.RGBA{s.brightness, s.brightness, s.brightness, 255}
		vector.DrawFilledRect(screen, float32(s.x), float32(s.y), 1, 1, c, false)
	}
	for _, a := range w.Asteroids {
		drawWrapped(w, a.Pos, a.Size.Radius(), func(x, y float32) { drawAsteroid(screen, a, x, y) })
	}
	for _, b := range w.Bullets {
		vector.DrawFilledCircle(screen, b.Pos.X.Float32(), b.Pos.Y.Float32(), 2, colBullet, true)
	}
	if s := w.Ship; s.Alive && blinkVisible(s) {
		drawWrapped(w, s.Pos, w.Cfg.ShipRadius, func(x, y float32) { drawShip(screen, s, w.Cfg.ShipRadius, x, y) })
	}
	for _, p := range g.fx.particles {
		alpha := uint8(255 * p.life / p.max)
		vector.DrawFilledRect(screen, float32(p.x), float32(p.y), 2, 2, color.RGBA{alpha, alpha, alpha, alpha}, false)
	}
	g.drawHUD(screen, w)
}

// blinkVisible flashes the ship while spawn protection is active.
func blinkVisible(s core.Ship) bool { return s.Invuln == 0 || (s.Invuln/6)%2 == 0 }

// drawWrapped draws an object once, plus ghost copies on the opposite edges
// when it straddles the seam, so rocks slide smoothly off one side and onto
// the other instead of popping.
func drawWrapped(w *core.World, pos fixed.Vec, radius fixed.F, draw func(x, y float32)) {
	width, height := float32(w.Cfg.Width), float32(w.Cfg.Height)
	x, y, r := pos.X.Float32(), pos.Y.Float32(), radius.Float32()
	xs := []float32{x}
	if x < r {
		xs = append(xs, x+width)
	} else if x > width-r {
		xs = append(xs, x-width)
	}
	ys := []float32{y}
	if y < r {
		ys = append(ys, y+height)
	} else if y > height-r {
		ys = append(ys, y-height)
	}
	for _, dx := range xs {
		for _, dy := range ys {
			draw(dx, dy)
		}
	}
}

// drawAsteroid strokes a lumpy polygon generated from the rock's Shape
// seed: 9–12 vertices, each pushed in by up to 30% of the radius. The
// outline is a pure function of the seed, so the same rock looks the same
// in every replay without storing any geometry.
func drawAsteroid(screen *ebiten.Image, a core.Asteroid, cx, cy float32) {
	h := a.Shape | 1
	next := func() float64 { // xorshift32: tiny, local, render-only
		h ^= h << 13
		h ^= h >> 17
		h ^= h << 5
		return float64(h) / float64(math.MaxUint32)
	}
	n := 9 + int(a.Shape%4)
	r := a.Size.Radius().Float64()
	rot := a.Rot.Radians()
	pts := make([][2]float32, n)
	for i := range pts {
		angle := rot + float64(i)*2*math.Pi/float64(n)
		dist := r * (0.7 + 0.3*next())
		pts[i] = [2]float32{cx + float32(math.Cos(angle)*dist), cy + float32(math.Sin(angle)*dist)}
	}
	for i := range pts {
		p, q := pts[i], pts[(i+1)%n]
		vector.StrokeLine(screen, p[0], p[1], q[0], q[1], 1.5, colLine, true)
	}
}

func drawShip(screen *ebiten.Image, s core.Ship, radius fixed.F, cx, cy float32) {
	heading := s.Heading.Radians()
	r := radius.Float64()
	point := func(angleOffset, dist float64) (float32, float32) {
		a := heading + angleOffset
		return cx + float32(math.Cos(a)*dist), cy + float32(math.Sin(a)*dist)
	}
	noseX, noseY := point(0, r)
	leftX, leftY := point(2.5, r)
	rightX, rightY := point(-2.5, r)
	backX, backY := point(math.Pi, r*0.4)
	vector.StrokeLine(screen, noseX, noseY, leftX, leftY, 1.5, colLine, true)
	vector.StrokeLine(screen, noseX, noseY, rightX, rightY, 1.5, colLine, true)
	vector.StrokeLine(screen, leftX, leftY, backX, backY, 1.5, colLine, true)
	vector.StrokeLine(screen, rightX, rightY, backX, backY, 1.5, colLine, true)
	if s.Thrusting {
		flameX, flameY := point(math.Pi, r*1.1)
		vector.StrokeLine(screen, backX, backY, flameX, flameY, 2, colFlame, true)
	}
}

func (g *Game) drawHUD(screen *ebiten.Image, w *core.World) {
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("SCORE %06d   LIVES %d   WAVE %d", w.Score, w.Lives, w.Wave), 10, 8)
	width, height := float32(w.Cfg.Width), float32(w.Cfg.Height)

	if g.mode == modeLive {
		help := "arrows/WASD fly  SPACE fire  P instant replay  F5 save  R new seed  B autopilot  ESC quit"
		if g.opts.autopilot {
			help = "AUTOPILOT  " + help
		}
		ebitenutil.DebugPrintAt(screen, help, 10, int(height)-20)
		if w.Over {
			ebitenutil.DebugPrintAt(screen, "GAME OVER - press ENTER to watch the replay", int(width)/2-130, int(height)/2)
		}
	} else {
		g.drawPlaybackHUD(screen, width, height)
	}
	if g.msgTTL > 0 {
		ebitenutil.DebugPrintAt(screen, g.message, 10, 24)
	}
}

// drawPlaybackHUD shows a timeline, the recorded inputs for this tick (lit
// when held) and the controls. Seeing the inputs light up is a good way to
// convince yourself the replay really is just a key log.
func (g *Game) drawPlaybackHUD(screen *ebiten.Image, width, height float32) {
	p := g.player
	barY := height - 40
	vector.DrawFilledRect(screen, 10, barY, width-20, 4, colHUDDim, false)
	if p.Len() > 0 {
		frac := float32(p.Tick()) / float32(p.Len())
		vector.DrawFilledRect(screen, 10, barY, (width-20)*frac, 4, colHUDLit, false)
	}

	in := p.Input()
	keys := []struct {
		label string
		bit   core.Input
	}{{"L", core.InputLeft}, {"R", core.InputRight}, {"T", core.InputThrust}, {"F", core.InputFire}}
	for i, k := range keys {
		c := colHUDDim
		if in.Has(k.bit) {
			c = colHUDLit
		}
		x := width - 120 + float32(i)*26
		vector.DrawFilledRect(screen, x, 8, 20, 16, c, false)
		ebitenutil.DebugPrintAt(screen, k.label, int(x)+7, 8)
	}

	state := fmt.Sprintf("REPLAY  tick %d/%d  %dx", p.Tick(), p.Len(), playbackSpeeds[g.speed])
	if g.paused {
		state += "  PAUSED"
	}
	if p.Done() {
		state += "  END"
	}
	ebitenutil.DebugPrintAt(screen, state, 10, int(barY)-18)
	ebitenutil.DebugPrintAt(screen, "SPACE pause  . step  <-/-> seek 5s  1-4 speed  HOME start  F5 save  ESC new game", 10, int(height)-20)
	if g.desync != nil {
		vector.DrawFilledRect(screen, 0, height/2-14, width, 28, colDesync, false)
		ebitenutil.DebugPrintAt(screen, g.desync.Error(), 12, int(height/2)-8)
	}
}
