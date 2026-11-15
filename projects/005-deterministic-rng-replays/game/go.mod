module github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays/game

go 1.22

require (
	github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays v0.0.0
	github.com/hajimehoshi/ebiten/v2 v2.8.8
)

// The game reuses the simulation from the parent module in this repository.
replace github.com/abubakaromar18/gaming_development/projects/005-deterministic-rng-replays => ../
