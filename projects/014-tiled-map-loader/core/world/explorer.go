package world

import (
	"fmt"
	"time"

	"github.com/abubakaromar18/gaming_development/projects/014-tiled-map-loader/core/geom"
)

// ExplorerConfig names the entity classes the demo game reacts to and tunes
// the player. Every value is data the map or the caller controls.
type ExplorerConfig struct {
	SpawnClass   string
	TriggerClass string // entities whose "message" property is shown on contact
	ChestClass   string // entities whose "loot" and "gold" are collected once
	PlayerSize   float64
	PlayerSpeed  float64 // pixels per second
	ChestHold    float64 // seconds a chest's reward message stays on screen
	TriggerHold  float64 // seconds a trigger's message lingers after leaving it
}

// DefaultExplorerConfig matches the bundled demo map.
func DefaultExplorerConfig() ExplorerConfig {
	return ExplorerConfig{
		SpawnClass: "spawn", TriggerClass: "trigger", ChestClass: "chest",
		PlayerSize: 10, PlayerSpeed: 90, ChestHold: 3, TriggerHold: 0.25,
	}
}

// Explorer is the demo's gameplay loop: walk the map, read signs, open
// chests. It is small on purpose; its job is to show map data driving
// behavior, with every rule living here rather than in the renderer.
type Explorer struct {
	World     *World
	Player    Actor
	Gold      int
	Inventory []string
	Message   string // text to display this frame, empty for none

	cfg        ExplorerConfig
	messageTTL float64
	opened     map[int]bool
}

// NewExplorer places the player, centered, on the map's spawn entity.
func NewExplorer(w *World, cfg ExplorerConfig) (*Explorer, error) {
	spawn, err := w.Spawn(cfg.SpawnClass)
	if err != nil {
		return nil, err
	}
	s := cfg.PlayerSize
	return &Explorer{
		World:  w,
		Player: Actor{Box: geom.Rect{X: spawn.X - s/2, Y: spawn.Y - s/2, W: s, H: s}, Speed: cfg.PlayerSpeed},
		cfg:    cfg,
		opened: map[int]bool{},
	}, nil
}

// Update advances the game by dt seconds with the player pushing intent.
func (e *Explorer) Update(intent geom.Vec, dt float64) {
	e.Player.Step(intent, dt, e.World.Collider)
	e.World.Animator.Advance(time.Duration(dt * float64(time.Second)))

	if e.messageTTL -= dt; e.messageTTL <= 0 {
		e.Message, e.messageTTL = "", 0
	}
	for _, t := range e.World.Touching(e.Player.Box, e.cfg.TriggerClass) {
		e.say(t.Properties.String("message", t.Name), e.cfg.TriggerHold)
	}
	for _, c := range e.World.Touching(e.Player.Box, e.cfg.ChestClass) {
		if e.opened[c.ID] {
			e.say("The chest is empty.", e.cfg.TriggerHold)
			continue
		}
		e.opened[c.ID] = true
		gold, loot := c.Properties.Int("gold", 0), c.Properties.String("loot", "")
		e.Gold += gold
		if loot != "" {
			e.Inventory = append(e.Inventory, loot)
		}
		e.say(fmt.Sprintf("Found %s and %d gold!", firstNonEmpty(loot, "nothing"), gold), e.cfg.ChestHold)
	}
}

// say shows text for hold seconds unless a longer-lived message (a chest
// reward) is still on screen.
func (e *Explorer) say(text string, hold float64) {
	if hold >= e.messageTTL {
		e.Message, e.messageTTL = text, hold
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
