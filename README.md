# 🎮 Gaming Development

**365 game development projects in 365 days.** Each project is a small, complete, tested codebase that demonstrates one major game feature or technique, such as fixed-timestep loops, rollback netcode, or wave function collapse. Each one also applies the software engineering practices behind it: clean architecture, design patterns, automated tests, and documented trade-offs.

## How the repo is organized

```
gaming_development/
├── README.md            ← you are here (overview + index of built projects)
├── PROJECTS.md          ← the full 365-day roadmap, checked off as projects ship
├── CONVENTIONS.md       ← the rules every project follows
├── templates/
│   └── PROJECT_README.md← README skeleton used by every project
└── projects/
    ├── 001-fixed-timestep-pong/
    ├── 002-input-action-mapping/
    └── ...              ← one self-contained folder per day: NNN-kebab-title/
```

Every project folder is self-contained. It has its own build files, source, tests, and a README that explains the **gaming technique**, the **software engineering concepts**, the design trade-offs, and how to run it.

## The 12 tracks

| # | Track | Days | Focus |
|---|-------|------|-------|
| 01 | Foundations | 001–030 | Game loops, input, state machines, core patterns |
| 02 | Graphics & Animation | 031–060 | Sprites, tweening, particles, shaders, game feel |
| 03 | Physics & Collision | 061–090 | AABB, SAT, GJK, rigid bodies, fluids, determinism |
| 04 | Game AI | 091–120 | A*, navmesh, behavior trees, GOAP, MCTS, flocking |
| 05 | Procedural Generation | 121–150 | Noise, dungeons, WFC, terrain, loot, worlds |
| 06 | Engine Architecture | 151–180 | ECS, job systems, allocators, scripting, tooling |
| 07 | Gameplay Systems | 181–210 | Inventory, quests, combat, economy, genre systems |
| 08 | Audio, UI & Tools | 211–240 | Adaptive audio, GUI, editors, analytics |
| 09 | Networking & Multiplayer | 241–270 | Prediction, rollback, lockstep, matchmaking, backend |
| 10 | 3D Graphics & Engines | 271–300 | Rasterizers, PBR, shadows, voxels, Three.js, Godot |
| 11 | Performance & Internals | 301–330 | Profiling, DOD, SIMD, VMs, emulators |
| 12 | Capstones & Live Ops | 331–365 | Complete games and production game services |

**Stacks used:** TypeScript/Canvas/WebGL2, Python/pygame, C++/SDL2/raylib/OpenGL, Rust/macroquad/Bevy, Go/Ebitengine, C#/MonoGame/.NET, Lua/LÖVE, Java/libGDX, Three.js, Godot 4, and more.

## Engineering principles across every project

- **Simulation separate from presentation.** Game rules live in pure, framework-free modules, so they can be unit-tested headlessly. Rendering is a thin shell on top.
- **Tests that run.** Every project ships automated tests for its core logic.
- **Documented decisions.** Each README explains why the design was chosen and what it costs.
- **Idiomatic code** for each language and ecosystem, with standard build files.

See [CONVENTIONS.md](CONVENTIONS.md) for the full standard.

## Built projects

| # | Project | Stack | Level |
|---|---------|-------|-------|
