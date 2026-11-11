# Project Roadmap

365 gaming projects, one per day, in 12 tracks that build on each other. Each line is `NNN | Title | Stack | Level | Track | Description`. The daily builder takes the first unchecked item.


## Track 01 — Foundations

_Game loops, input, state, and core patterns_

- [ ] 001 | Fixed-Timestep Pong | TypeScript + Canvas | Starter | Foundations | Pong on a fixed-timestep loop with interpolated rendering, decoupling simulation rate from frame rate
- [ ] 002 | Input Action Mapping | Python + pygame | Starter | Foundations | Breakout with an action-mapping input layer (rebindable keyboard/gamepad) separating devices from game commands
- [ ] 003 | Stack-Based State Machine | Lua + LÖVE | Starter | Foundations | Snake with menu/play/pause/game-over states managed by a pushdown automaton
- [ ] 004 | Scene Manager & Transitions | C# + MonoGame | Starter | Foundations | Multi-scene arcade shell with scene lifecycle hooks and fade transitions
- [ ] 005 | Deterministic RNG & Replays | Go + Ebitengine | Intermediate | Foundations | Asteroids with seeded RNG and input recording for frame-exact replay playback
- [ ] 006 | Command Pattern Undo | Python | Starter | Foundations | Sokoban whose moves are commands, giving unlimited undo/redo and a level solver check
- [ ] 007 | Tetris with SRS | Rust + macroquad | Intermediate | Foundations | Tetris with Super Rotation System wall kicks, 7-bag randomizer, lock delay, and T-spin detection
- [ ] 008 | Match-3 Board Engine | TypeScript + Canvas | Starter | Foundations | Match-3 with swap validation, cascade resolution, refill, and no-moves detection
- [ ] 009 | Minesweeper with Hint Solver | Java + libGDX | Starter | Foundations | Minesweeper with first-click safety, flood fill, and a constraint-based hint system
- [ ] 010 | 2048 as a Pure Reducer | TypeScript | Starter | Foundations | 2048 whose rules are a pure, fully unit-tested reducer behind a thin rendering shell
- [ ] 011 | Space Invaders with Object Pools | C++ + SDL2 | Intermediate | Foundations | Invaders using object pools for bullets and enemies to avoid per-frame allocation
- [ ] 012 | 2D Camera Systems | Python + pygame | Starter | Foundations | Scrolling demo with follow, deadzone, look-ahead, bounds, and trauma-based screen shake
- [ ] 013 | Coroutines & Timers | Lua + LÖVE | Starter | Foundations | Wave shooter scripted with coroutine-driven timers and tween sequences
- [ ] 014 | Tiled Map Loader | Go + Ebitengine | Intermediate | Foundations | Loads Tiled JSON maps with tile layers, object layers, and collision metadata
- [ ] 015 | Platformer Character Controller | C# + MonoGame | Intermediate | Foundations | Tight platformer controls: coyote time, jump buffering, variable jump height, apex hang
- [ ] 016 | Top-Down Movement & Dash | Rust + macroquad | Starter | Foundations | Twin-stick movement with acceleration curves, dash with invulnerability frames, and aim assist
- [ ] 017 | Game Settings System | Python | Starter | Foundations | Graphics/audio/control settings with schema validation, defaults, and persistence
- [ ] 018 | Versioned Save Files | C# (.NET) | Intermediate | Foundations | Save system with schema versions, migrations, checksums, and atomic writes
- [ ] 019 | Typed Event Bus | TypeScript | Starter | Foundations | Typed pub/sub bus decoupling score, audio, and UI in an arcade game
- [ ] 020 | Achievement System | Java | Starter | Foundations | Observer-driven achievements with progress tracking and unlock notifications
- [ ] 021 | Time Scaling & Hit-Stop | Lua + LÖVE | Starter | Foundations | Time-scale system with hit-stop, bullet time, and pause-aware timers
- [ ] 022 | Local High Score Service | Go | Starter | Foundations | Persistent leaderboard with validation, tie-breaking, and a small REST API
- [ ] 023 | Frogger with Data-Driven Lanes | Python + pygame | Starter | Foundations | Frogger built from data-driven lane spawners and a difficulty curve
- [ ] 024 | Pac-Man Ghost Targeting | TypeScript + Canvas | Intermediate | Foundations | Pac-Man with the original scatter/chase/frightened ghost targeting rules
- [ ] 025 | Turn-Based Game Core | Python | Starter | Foundations | Connect Four with a turn manager, win detection, and pluggable human/AI players
- [ ] 026 | Bitboard Chess Move Generator | Rust | Intermediate | Foundations | Legal chess move generation with bitboards, verified by perft tests
- [ ] 027 | Card Game Framework | C# (.NET) | Intermediate | Foundations | Reusable deck/hand/pile model powering Klondike Solitaire with auto-move and undo
- [ ] 028 | Debug Overlay & Console | C++ + raylib | Intermediate | Foundations | In-game debug HUD, frame-time graph, and command console with registered cvars
- [ ] 029 | Headless Simulation Harness | Python | Intermediate | Foundations | Test harness that runs thousands of simulated matches headlessly to find balance and logic bugs
- [ ] 030 | Foundations Capstone: Mini 2D Engine | TypeScript + Canvas | Advanced | Foundations | Small engine combining loop, input, scenes, events, and assets, shipping a demo arcade game

## Track 02 — Graphics & Animation

_2D rendering, animation, shaders, and game feel_

- [ ] 031 | Sprite Sheet Animator | Python + pygame | Starter | Graphics & Animation | Clip-based sprite animation with frame events and loop/ping-pong modes
- [ ] 032 | Texture Atlas Packer | Go | Intermediate | Graphics & Animation | CLI that bin-packs sprites into atlases with MaxRects and emits JSON metadata
- [ ] 033 | Tweening & Easing Library | TypeScript | Starter | Graphics & Animation | Easing functions and a tween engine with chaining, yoyo, and timelines
- [ ] 034 | Pooled Particle System | C++ + raylib | Intermediate | Graphics & Animation | Emitter-based particles with pooling, over-lifetime curves, and burst presets
- [ ] 035 | Parallax Endless Runner | Lua + LÖVE | Starter | Graphics & Animation | Multi-layer parallax runner with seamless looping and speed ramping
- [ ] 036 | Animation State Machine | C# + MonoGame | Intermediate | Graphics & Animation | Character animator with transitions, conditions, and blend timings
- [ ] 037 | 2D Skeletal Animation | TypeScript + Canvas | Advanced | Graphics & Animation | Bone hierarchies, forward kinematics, and keyframe interpolation for 2D rigs
- [ ] 038 | Two-Bone IK | Rust + macroquad | Intermediate | Graphics & Animation | Analytic two-bone inverse kinematics for procedural foot placement
- [ ] 039 | Procedural Creature Animation | TypeScript + Canvas | Intermediate | Graphics & Animation | Chain-constraint lizard and snake animation driven by procedural IK
- [ ] 040 | Sprite Batch Renderer | C++ + SDL2 | Advanced | Graphics & Animation | Batched quad renderer that minimizes draw calls with texture and layer sorting
- [ ] 041 | 2D Lights & Shadows | TypeScript + WebGL2 | Advanced | Graphics & Animation | Point lights with shadow casting via visibility polygons
- [ ] 042 | Pixel-Perfect Scaling & CRT Shaders | TypeScript + WebGL2 | Intermediate | Graphics & Animation | Integer scaling, CRT, scanline, and palette-swap post effects
- [ ] 043 | Screen Transitions & Post FX | Go + Ebitengine | Starter | Graphics & Animation | Shader-based wipes, dissolves, and chromatic aberration
- [ ] 044 | Bitmap Font Renderer | C++ + SDL2 | Intermediate | Graphics & Animation | BMFont loader with kerning, word wrap, and inline rich-text tags
- [ ] 045 | Trails & Ribbons | TypeScript + Canvas | Starter | Graphics & Animation | Motion trails for projectiles and sword swings built from triangle strips
- [ ] 046 | Game Feel Toolkit | Lua + LÖVE | Starter | Graphics & Animation | Screen shake, squash and stretch, hit flash, and knockback presets
- [ ] 047 | Isometric Renderer | Python + pygame | Intermediate | Graphics & Animation | Isometric tile map with depth sorting and mouse picking
- [ ] 048 | Hex Grid Library | TypeScript | Intermediate | Graphics & Animation | Axial/cube hex coordinates, distance, line drawing, rings, and rendering
- [ ] 049 | Day/Night Cycle | C# + MonoGame | Starter | Graphics & Animation | Time-of-day color grading and ambient light transitions
- [ ] 050 | Spring-Based 2D Water | Rust + macroquad | Intermediate | Graphics & Animation | Spring-mesh water surface with splashes and buoyant objects
- [ ] 051 | Destructible Pixel Terrain | C++ + raylib | Advanced | Graphics & Animation | Worms-style terrain with bitmask destruction, collision, and explosions
- [ ] 052 | Metaball Rendering | TypeScript + WebGL2 | Intermediate | Graphics & Animation | Fluid-like blobs rendered from a metaball field shader
- [ ] 053 | Sprite Shader Pack | TypeScript + WebGL2 | Starter | Graphics & Animation | Outline, dissolve, damage flash, and hue-shift sprite shaders
- [ ] 054 | Cutscene Timeline | TypeScript | Intermediate | Graphics & Animation | Data-driven cutscene sequencer with camera, dialogue, and animation tracks
- [ ] 055 | Nine-Slice UI & Layout | C# + MonoGame | Intermediate | Graphics & Animation | Nine-slice panels and a flexbox-style layout system for game UI
- [ ] 056 | Fog of War & Minimap | Go + Ebitengine | Intermediate | Graphics & Animation | Explored-area fog of war with a live minimap
- [ ] 057 | Floating Combat Text | Lua + LÖVE | Starter | Graphics & Animation | Pooled damage numbers with stacking, crit styling, and easing
- [ ] 058 | Render Layers & Y-Sorting | Python + pygame | Starter | Graphics & Animation | Layered render queue with y-sorting for top-down scenes
- [ ] 059 | Asset Hot Reloading | Rust + macroquad | Intermediate | Graphics & Animation | File-watcher hot reload of textures and tuning config during play
- [ ] 060 | Graphics Capstone: Metroidvania Room | C# + MonoGame | Advanced | Graphics & Animation | One polished Metroidvania room combining animation, lighting, parallax, and juice

## Track 03 — Physics & Collision

_Collision detection, simulation, and physics engines_

- [ ] 061 | Tile AABB Collision | Python + pygame | Starter | Physics & Collision | Axis-separated AABB resolution for a tile-based platformer
- [ ] 062 | Swept AABB | TypeScript | Intermediate | Physics & Collision | Continuous collision detection that prevents tunneling for fast movers
- [ ] 063 | Separating Axis Theorem | C++ | Intermediate | Physics & Collision | SAT for convex polygons with minimum translation vectors
- [ ] 064 | GJK & EPA | Rust | Advanced | Physics & Collision | GJK intersection and EPA penetration depth for arbitrary convex shapes
- [ ] 065 | Verlet Ropes & Cloth | TypeScript + Canvas | Starter | Physics & Collision | Verlet integration rope and cloth with distance constraints and tearing
- [ ] 066 | Spatial Hash Broadphase | Go | Intermediate | Physics & Collision | Spatial hashing broadphase with benchmarks against brute force
- [ ] 067 | Dynamic Quadtree | C# (.NET) | Intermediate | Physics & Collision | Dynamic quadtree for collision and range queries with benchmarks
- [ ] 068 | 2D Rigid Body Engine | C++ + raylib | Advanced | Physics & Collision | Impulse-based rigid bodies with rotation, friction, and restitution
- [ ] 069 | Grid Raycasting | Lua + LÖVE | Starter | Physics & Collision | DDA raycasts for visibility, bullets, and lasers
- [ ] 070 | Ballistics & Trajectory Prediction | Python + pygame | Starter | Physics & Collision | Slingshot launcher with drag and predicted trajectory arcs
- [ ] 071 | Box2D Physics Puzzle | C++ + SDL2 | Intermediate | Physics & Collision | Physics puzzle using Box2D bodies, joints, and contact listeners
- [ ] 072 | Top-Down Vehicle Physics | Rust + macroquad | Intermediate | Physics & Collision | Car with tire grip, drift, and handbrake model
- [ ] 073 | Soft Body Jelly | TypeScript + Canvas | Intermediate | Physics & Collision | Pressure-based soft bodies built from spring meshes
- [ ] 074 | Kinematic Character Controller | C# + MonoGame | Intermediate | Physics & Collision | Slopes, moving platforms, and one-way platforms for a kinematic body
- [ ] 075 | Pinball Physics | TypeScript + Canvas | Intermediate | Physics & Collision | Pinball with flippers, bumpers, continuous collision, and sub-stepping
- [ ] 076 | Billiards Simulation | Python | Intermediate | Physics & Collision | Elastic ball collisions with spin, friction, and cushion response
- [ ] 077 | SPH Fluid Simulation | Rust | Advanced | Physics & Collision | 2D smoothed-particle hydrodynamics accelerated with spatial hashing
- [ ] 078 | Falling Sand Simulation | Go + Ebitengine | Intermediate | Physics & Collision | Cellular material simulation: sand, water, fire, smoke, and wood
- [ ] 079 | Constraint Solver | TypeScript | Advanced | Physics & Collision | Sequential impulse solver for distance, revolute, and weld joints
- [ ] 080 | Grappling Hook | Lua + LÖVE | Starter | Physics & Collision | Swinging grapple with pendulum constraints and momentum conservation
- [ ] 081 | Force Volumes | C# + MonoGame | Starter | Physics & Collision | Buoyancy, wind, and gravity-well zones that apply forces to bodies
- [ ] 082 | Orbital Mechanics | Python + pygame | Intermediate | Physics & Collision | N-body gravity with symplectic integration and orbit prediction
- [ ] 083 | Deterministic Fixed-Point Physics | C++ | Advanced | Physics & Collision | Fixed-point physics for lockstep games, with cross-run state-hash tests
- [ ] 084 | Collision Layers & Triggers | TypeScript | Starter | Physics & Collision | Bitmask collision filtering with trigger and solid volumes
- [ ] 085 | Verlet Ragdoll | Rust + macroquad | Intermediate | Physics & Collision | 2D ragdoll with joint angle limits, triggered on enemy death
- [ ] 086 | Voronoi Fracture | Go + Ebitengine | Intermediate | Physics & Collision | Breakable polygons that fracture into Voronoi shards on impact
- [ ] 087 | Physics Debug Visualizer | C++ + raylib | Starter | Physics & Collision | Draws contacts, normals, AABBs, and broadphase cells
- [ ] 088 | Shape-Cast Bullets | C# (.NET) | Intermediate | Physics & Collision | Shape casting for high-speed projectiles with reliable hit registration
- [ ] 089 | Islands & Body Sleeping | Rust | Intermediate | Physics & Collision | Contact island detection and body sleeping with performance metrics
- [ ] 090 | Physics Capstone: Physics Puzzler | TypeScript + Canvas | Advanced | Physics & Collision | Complete physics puzzle game on a custom engine, with a level editor

## Track 04 — Game AI

_Pathfinding, decision making, and agent behavior_

- [ ] 091 | A* Pathfinding Visualizer | Python + pygame | Starter | Game AI | Grid A* with heuristic comparison, diagonal rules, and step-through visualization
- [ ] 092 | Jump Point Search | C++ | Intermediate | Game AI | JPS on uniform grids, benchmarked against A*
- [ ] 093 | Flow Field Pathfinding | Go + Ebitengine | Intermediate | Game AI | Flow fields steering hundreds of units toward a goal
- [ ] 094 | 2D Navigation Mesh | TypeScript | Advanced | Game AI | Navmesh from polygons with funnel-algorithm path smoothing
- [ ] 095 | Steering Behaviors | Lua + LÖVE | Starter | Game AI | Seek, flee, arrive, wander, pursue, and evade with weighted blending
- [ ] 096 | Boids Flocking | Rust + macroquad | Starter | Game AI | Reynolds flocking accelerated with spatial partitioning
- [ ] 097 | Behavior Tree Library | C# (.NET) | Intermediate | Game AI | Behavior trees with composites, decorators, and a blackboard, driving a guard demo
- [ ] 098 | Hierarchical FSM Enemies | TypeScript + Canvas | Intermediate | Game AI | Patrol, alert, and combat enemies built on a hierarchical state machine with perception
- [ ] 099 | Utility AI | Python | Intermediate | Game AI | Utility scoring with response curves for a survival-sim agent
- [ ] 100 | GOAP Planner | C# (.NET) | Advanced | Game AI | Goal-oriented action planning with A* over symbolic world states
- [ ] 101 | Alpha-Beta Tic-Tac-Toe | Java | Starter | Game AI | Unbeatable tic-tac-toe with minimax, alpha-beta pruning, and move ordering
- [ ] 102 | Chess Engine Search | Rust | Advanced | Game AI | Negamax with iterative deepening, transposition tables, and quiescence search
- [ ] 103 | Monte Carlo Tree Search | Python | Intermediate | Game AI | MCTS with UCT playing Othello
- [ ] 104 | Stealth Perception | Go + Ebitengine | Intermediate | Game AI | Guards with vision cones, noise hearing, and suspicion meters
- [ ] 105 | Influence Maps | TypeScript | Intermediate | Game AI | Influence maps for RTS territory control and tactical positioning
- [ ] 106 | Squad Tactics & Cover | C++ + raylib | Advanced | Game AI | Cover point evaluation, flanking, and suppression coordination
- [ ] 107 | RTS Formations & RVO | C# + MonoGame | Intermediate | Game AI | Group movement with formations and reciprocal velocity obstacles
- [ ] 108 | AI Director Pacing | Python | Intermediate | Game AI | Intensity tracking and spawn pacing in the style of an AI director
- [ ] 109 | Dynamic Difficulty Adjustment | TypeScript | Starter | Game AI | Performance-based difficulty tuning with telemetry
- [ ] 110 | Q-Learning Agent | Python | Intermediate | Game AI | Tabular Q-learning agent that learns to play a grid-world game
- [ ] 111 | Neuroevolution Flappy Bird | TypeScript + Canvas | Intermediate | Game AI | Small neural nets evolved with a genetic algorithm to play Flappy Bird
- [ ] 112 | Racing AI on Splines | Rust + macroquad | Intermediate | Game AI | Racing-line following with spline look-ahead and overtaking
- [ ] 113 | Fighting Game AI | Lua + LÖVE | Intermediate | Game AI | Frame-data-aware opponent with reaction delays and difficulty profiles
- [ ] 114 | Companion Follow AI | C# + MonoGame | Starter | Game AI | Follower with leash distance, catch-up teleport, and obstacle handling
- [ ] 115 | Crowd Simulation | C++ | Advanced | Game AI | Thousands of agents with ORCA avoidance and a data-oriented layout
- [ ] 116 | Hierarchical Pathfinding (HPA*) | Go | Advanced | Game AI | Cluster abstraction for large maps with dynamic obstacle updates
- [ ] 117 | Poker Bot | Java | Intermediate | Game AI | Fast hand evaluator and Monte Carlo equity-based Texas Hold'em AI
- [ ] 118 | NPC Response System | Python | Starter | Game AI | Rule-and-blackboard-driven NPC barks and contextual reactions
- [ ] 119 | AI Debugger | TypeScript + Canvas | Intermediate | Game AI | Visual debugger showing behavior tree state, paths, and perception
- [ ] 120 | AI Capstone: Stealth Level | Go + Ebitengine | Advanced | Game AI | Complete stealth level with behavior-tree guards, navmesh, and alert propagation

## Track 05 — Procedural Generation

_Noise, levels, worlds, and content generation_

- [ ] 121 | Noise Library | Rust | Starter | Procedural Generation | Perlin, simplex, and fractal Brownian motion noise with an image visualizer
- [ ] 122 | BSP Dungeons | Python + pygame | Starter | Procedural Generation | Binary space partition rooms joined by corridors
- [ ] 123 | Cellular Automata Caves | Lua + LÖVE | Starter | Procedural Generation | Cave generation with smoothing passes and flood-fill connectivity
- [ ] 124 | Wave Function Collapse | TypeScript | Advanced | Procedural Generation | Tiled and overlapping WFC with propagation and backtracking
- [ ] 125 | Drunkard's Walk Levels | Go | Starter | Procedural Generation | Random-walk level carving with guaranteed connectivity
- [ ] 126 | Heightmap Terrain & Biomes | C++ + raylib | Intermediate | Procedural Generation | Noise terrain with moisture and temperature biome classification
- [ ] 127 | Poisson Disk Placement | TypeScript + Canvas | Starter | Procedural Generation | Blue-noise placement of trees, rocks, and loot
- [ ] 128 | L-System Flora | Python | Starter | Procedural Generation | Lindenmayer systems for procedural trees and plants
- [ ] 129 | Voronoi Island Maps | Rust + macroquad | Intermediate | Procedural Generation | Voronoi/Delaunay island maps with elevation and rivers
- [ ] 130 | Markov Name Generator | Python | Starter | Procedural Generation | Markov-chain names for places, characters, and items
- [ ] 131 | Room-Template Levels | C# + MonoGame | Intermediate | Procedural Generation | Templated rooms laid along a guaranteed solution path
- [ ] 132 | Lock-and-Key Mission Graphs | TypeScript | Advanced | Procedural Generation | Mission graph generation with locks, keys, and backtracking
- [ ] 133 | Affix Loot Generator | Java | Intermediate | Procedural Generation | Affix-based weapons with rarity tiers and stat budgets
- [ ] 134 | Infinite Chunked World | Go + Ebitengine | Intermediate | Procedural Generation | Chunk streaming with deterministic per-chunk seeds
- [ ] 135 | Road & City Generation | TypeScript + Canvas | Advanced | Procedural Generation | Road networks with building lots and districts
- [ ] 136 | Hydraulic Erosion | Rust | Intermediate | Procedural Generation | Droplet-based hydraulic erosion on heightmaps
- [ ] 137 | Generative Game Music | TypeScript + Web Audio | Intermediate | Procedural Generation | Scale-aware generative music that reacts to gameplay intensity
- [ ] 138 | Planet Generator | JavaScript + Three.js | Advanced | Procedural Generation | Cube-sphere planets with noise terrain and level of detail
- [ ] 139 | Quest Grammar Generator | Python | Intermediate | Procedural Generation | Grammar-based quests constrained by world state
- [ ] 140 | Pixel Sprite Generator | Python | Starter | Procedural Generation | Mirrored-mask pixel spaceships and creatures exported as PNG
- [ ] 141 | Maze Algorithm Gallery | C++ + SDL2 | Starter | Procedural Generation | Backtracker, Prim's, Kruskal's, and Wilson's mazes, visualized
- [ ] 142 | Composable Level Pipeline | Rust | Intermediate | Procedural Generation | Roguelike generation as composable passes with seed-reproducibility tests
- [ ] 143 | Procedural Walk Cycles | TypeScript + Canvas | Intermediate | Procedural Generation | Gait generation for multi-legged creatures
- [ ] 144 | Galaxy Generator | C# (.NET) | Intermediate | Procedural Generation | Spiral galaxy star systems generated from hierarchical seeds
- [ ] 145 | Level Solvability Validator | Python | Intermediate | Procedural Generation | Checks that generated puzzle levels are solvable and rejects bad seeds
- [ ] 146 | Shader Textures | TypeScript + WebGL2 | Intermediate | Procedural Generation | Shader-generated wood, marble, and rock textures
- [ ] 147 | Encounter Tables | Lua + LÖVE | Starter | Procedural Generation | Weighted spawn tables with budget-based encounter building
- [ ] 148 | Marching Squares Terrain | Go + Ebitengine | Intermediate | Procedural Generation | Smooth editable terrain contours from density fields
- [ ] 149 | Daily Seeded Challenge | TypeScript + Node | Starter | Procedural Generation | Daily seeded runs with shareable seed codes and a leaderboard
- [ ] 150 | ProcGen Capstone: Roguelike | Rust + macroquad | Advanced | Procedural Generation | Turn-based roguelike with generated dungeons, field of view, items, and permadeath

## Track 06 — Engine Architecture

_ECS, data-driven design, and engine infrastructure_

- [ ] 151 | Archetype ECS | C++ | Advanced | Engine Architecture | Archetype-based entity component system with queries and benchmarks
- [ ] 152 | Sparse-Set ECS | Rust | Intermediate | Engine Architecture | Sparse-set ECS with system scheduling and change detection
- [ ] 153 | Bevy Arcade Game | Rust + Bevy | Intermediate | Engine Architecture | Arcade game built idiomatically with Bevy plugins, resources, and events
- [ ] 154 | Composition over Inheritance | C# (.NET) | Starter | Engine Architecture | Component-based game objects compared side by side with a deep inheritance tree
- [ ] 155 | Data-Driven Prefabs | Python | Intermediate | Engine Architecture | JSON/YAML entity definitions with schema validation and prefab inheritance
- [ ] 156 | Service Locator vs DI | C# (.NET) | Intermediate | Engine Architecture | Engine services via a service locator and a DI container, with test doubles
- [ ] 157 | Deferred Message Queue | Go | Intermediate | Engine Architecture | Message dispatch with priorities and per-frame time budgets
- [ ] 158 | Sandboxed Lua Mods | C + Lua | Advanced | Engine Architecture | Embedding Lua for mods behind a sandboxed, versioned API
- [ ] 159 | Engine Scripting Bindings | C++ + Lua | Advanced | Engine Architecture | Binding engine types and functions to Lua for gameplay scripting
- [ ] 160 | Resource Manager | C++ | Intermediate | Engine Architecture | Handle-based resources with caching, reference counting, and async loading
- [ ] 161 | Job System | C++ | Advanced | Engine Architecture | Work-stealing job system with dependencies and parallel-for
- [ ] 162 | Custom Memory Allocators | C++ | Advanced | Engine Architecture | Linear, stack, and pool allocators with leak and usage tracking
- [ ] 163 | Versioned Serialization | Rust | Intermediate | Engine Architecture | Serde-based world serialization with format versioning
- [ ] 164 | Reflection & Inspector | C# (.NET) | Intermediate | Engine Architecture | Attribute-driven reflection powering a generic property inspector
- [ ] 165 | Editor Undo Architecture | TypeScript | Intermediate | Engine Architecture | Command stack for a level editor with command merging and transactions
- [ ] 166 | Tile & Entity Level Editor | TypeScript + Canvas | Advanced | Engine Architecture | Level editor with layers, entity placement, and JSON export
- [ ] 167 | World Snapshots & Diffing | Go | Intermediate | Engine Architecture | Snapshot and restore of world state with diffing
- [ ] 168 | Hexagonal Architecture Game | Java | Intermediate | Engine Architecture | Domain, use cases, and adapters separated so game rules run without the framework
- [ ] 169 | Test-Driven Game Kata | Python | Starter | Engine Architecture | A small game built strictly test-first, with the red-green-refactor history documented
- [ ] 170 | Gameplay Ability System | C# (.NET) | Advanced | Engine Architecture | Abilities, effects, attributes, tags, and cooldowns
- [ ] 171 | Stats & Modifiers | TypeScript | Intermediate | Engine Architecture | Stacking flat and percent modifiers with dependency invalidation
- [ ] 172 | Gameplay Tags | Rust | Starter | Engine Architecture | Hierarchical gameplay tags with fast containment queries
- [ ] 173 | Localization System | Python | Starter | Engine Architecture | String tables, plural rules, fallbacks, and pseudo-localization
- [ ] 174 | Game Telemetry Client | Go | Intermediate | Engine Architecture | Structured gameplay event telemetry with batching and retry
- [ ] 175 | Crash Reporter | C++ | Intermediate | Engine Architecture | Signal handlers, stack traces, and crash report files
- [ ] 176 | Asset Cooking Pipeline | Python | Intermediate | Engine Architecture | Incremental asset processing with content hashing and caching
- [ ] 177 | CI for Games | TypeScript + Node | Intermediate | Engine Architecture | GitHub Actions pipeline with headless tests, linting, and web builds
- [ ] 178 | Remote Config & Feature Flags | Go | Starter | Engine Architecture | Live tuning service with typed flags and client caching
- [ ] 179 | Game Programming Patterns | Java | Intermediate | Engine Architecture | Flyweight, prototype, type object, dirty flag, and double buffer, each with a demo
- [ ] 180 | Architecture Capstone: Mini Engine | C++ + SDL2 | Advanced | Engine Architecture | ECS engine with resources, jobs, scripting, and a demo game

## Track 07 — Gameplay Systems

_Inventory, combat, progression, economy, and genre systems_

- [ ] 181 | Grid Inventory | C# (.NET) | Intermediate | Gameplay Systems | Slot and grid inventories with stacking, weight limits, and drag-and-drop logic
- [ ] 182 | Crafting System | Python | Intermediate | Gameplay Systems | Recipe graph with discovery, ingredients, and crafting queues
- [ ] 183 | Branching Dialogue | TypeScript | Intermediate | Gameplay Systems | Branching dialogue with conditions, variables, and a small script format
- [ ] 184 | Quest System | C# (.NET) | Intermediate | Gameplay Systems | Event-driven quest objectives, states, and rewards
- [ ] 185 | JRPG Turn-Based Combat | Python | Intermediate | Gameplay Systems | Initiative order, skills, status effects, and enemy AI
- [ ] 186 | Tactics Grid Combat | TypeScript + Canvas | Advanced | Gameplay Systems | Movement ranges, hit chance, cover, and overwatch
- [ ] 187 | Hitboxes & Combos | Lua + LÖVE | Intermediate | Gameplay Systems | Frame-based hitboxes and hurtboxes, combo chains, and cancel windows
- [ ] 188 | Fighting Game Input Buffer | C++ | Intermediate | Gameplay Systems | Motion input parsing (quarter-circles, charges), buffering, and priority
- [ ] 189 | Loot Tables & Pity Timers | Go | Starter | Gameplay Systems | Weighted loot tables with pity timers, verified with statistical tests
- [ ] 190 | Skill Trees | TypeScript | Intermediate | Gameplay Systems | DAG skill trees with prerequisites, point costs, and respec
- [ ] 191 | RPG Progression Curves | Java | Starter | Gameplay Systems | XP curves, leveling, and stat growth with tuning tools
- [ ] 192 | Game Economy Simulator | Python | Intermediate | Gameplay Systems | Currency sources and sinks, dynamic pricing, and inflation simulation
- [ ] 193 | Tower Defense | Go + Ebitengine | Intermediate | Gameplay Systems | Waves, tower targeting priorities, and upgrades
- [ ] 194 | Card Battler Rules Engine | TypeScript | Advanced | Gameplay Systems | Effect stack, enemy intents, and deckbuilding
- [ ] 195 | Survival Needs | C# + MonoGame | Starter | Gameplay Systems | Hunger, thirst, and temperature with decay and recovery
- [ ] 196 | Building Placement | Rust + macroquad | Intermediate | Gameplay Systems | Grid placement with validation, rotation, and ghost preview
- [ ] 197 | Farming Sim Systems | Lua + LÖVE | Intermediate | Gameplay Systems | Crop growth, calendar, seasons, and weather
- [ ] 198 | City Builder Simulation | C++ + raylib | Advanced | Gameplay Systems | Zoning, demand, resource flow, and simple traffic
- [ ] 199 | Factory Automation | Rust | Advanced | Gameplay Systems | Belts, inserters, and assemblers with throughput simulation
- [ ] 200 | Idle Incremental Game | TypeScript | Starter | Gameplay Systems | Big-number math, offline progress, and prestige loops
- [ ] 201 | Match-3 Boosters & Goals | C# (.NET) | Intermediate | Gameplay Systems | Special tiles, combo rules, and level goals
- [ ] 202 | Rhythm Game Timing | TypeScript + Web Audio | Intermediate | Gameplay Systems | Audio-synced note highway with judgment windows and latency calibration
- [ ] 203 | Racing Game Systems | Go + Ebitengine | Intermediate | Gameplay Systems | Laps, checkpoints, race positions, and ghost replays
- [ ] 204 | Status Effects Framework | Python | Starter | Gameplay Systems | Buffs and debuffs with stacking rules, durations, and ticks
- [ ] 205 | Interaction System | C# + MonoGame | Starter | Gameplay Systems | Interactable prompts with priority, range, and focus
- [ ] 206 | Weather System | Lua + LÖVE | Starter | Gameplay Systems | Weather states, transitions, and gameplay effects
- [ ] 207 | Factions & Reputation | Python | Intermediate | Gameplay Systems | Faction relationships with reputation propagation
- [ ] 208 | Tutorial & Onboarding | TypeScript | Starter | Gameplay Systems | Step-based tutorials with triggers and feature gating
- [ ] 209 | Accessibility Options | TypeScript + Canvas | Intermediate | Gameplay Systems | Remapping, colorblind palettes, subtitles, and assist modes
- [ ] 210 | Systems Capstone: RPG Vertical Slice | C# + MonoGame | Advanced | Gameplay Systems | Inventory, dialogue, quests, combat, and saves in one playable slice

## Track 08 — Audio, UI & Tools

_Sound, interface, editors, and developer tooling_

- [ ] 211 | Audio Manager | C# (.NET) | Starter | Audio, UI & Tools | Mixer buses, volume groups, and voice pooling
- [ ] 212 | Adaptive Music | TypeScript + Web Audio | Intermediate | Audio, UI & Tools | Layered stems and horizontal re-sequencing driven by game state
- [ ] 213 | Positional Audio | C++ | Intermediate | Audio, UI & Tools | Distance attenuation, panning, and Doppler using miniaudio
- [ ] 214 | Procedural Sound Effects | Python | Starter | Audio, UI & Tools | sfxr-style sound effect synthesizer that exports WAV files
- [ ] 215 | Audio Visualizer | TypeScript + Web Audio | Starter | Audio, UI & Tools | FFT and beat detection driving visuals
- [ ] 216 | Immediate-Mode GUI | C++ | Advanced | Audio, UI & Tools | A small immediate-mode GUI library from scratch
- [ ] 217 | Retained UI with Gamepad Focus | TypeScript | Intermediate | Audio, UI & Tools | Widget tree, layout, and directional focus navigation
- [ ] 218 | Game HUD | Lua + LÖVE | Starter | Audio, UI & Tools | Health bars, ammo counters, and radial cooldowns
- [ ] 219 | Gamepad Menu System | C# + MonoGame | Intermediate | Audio, UI & Tools | Menus navigable by keyboard and gamepad with back stack
- [ ] 220 | Dear ImGui Dev Tools | C++ + Dear ImGui | Intermediate | Audio, UI & Tools | Entity inspector, profiler view, and tuning panels
- [ ] 221 | Dialogue Graph Editor | TypeScript + React | Advanced | Audio, UI & Tools | Node-graph dialogue editor exporting JSON
- [ ] 222 | Sprite Animation Editor | Python + pygame | Intermediate | Audio, UI & Tools | Tool for slicing sheets and authoring animation clips
- [ ] 223 | Autotiling | Go + Ebitengine | Intermediate | Audio, UI & Tools | 47-tile blob autotiling with bitmasks
- [ ] 224 | Particle Editor | TypeScript + Canvas | Intermediate | Audio, UI & Tools | Live particle editor with preset export
- [ ] 225 | Gameplay Analytics | Python | Intermediate | Audio, UI & Tools | Death heatmaps and funnel analysis from telemetry logs
- [ ] 226 | Replay Viewer | Rust + macroquad | Intermediate | Audio, UI & Tools | Scrubbable viewer for recorded matches
- [ ] 227 | Localization Tooling | Python | Starter | Audio, UI & Tools | CSV-to-JSON pipeline with missing-key and overflow reports
- [ ] 228 | Balance Simulator | Python | Intermediate | Audio, UI & Tools | Monte Carlo balance testing for weapons and units
- [ ] 229 | Screenshot & GIF Capture | Go + Ebitengine | Starter | Audio, UI & Tools | In-game screenshot and animated GIF capture
- [ ] 230 | Input Rebinding UI | TypeScript | Starter | Audio, UI & Tools | Rebinding screen with conflict detection and persistence
- [ ] 231 | SDF Text Rendering | TypeScript + WebGL2 | Advanced | Audio, UI & Tools | Signed distance field fonts with outlines and glow
- [ ] 232 | Toast Notifications | C# (.NET) | Starter | Audio, UI & Tools | Queued in-game notifications with priorities
- [ ] 233 | Weapon Wheel | Lua + LÖVE | Starter | Audio, UI & Tools | Radial selection menu for mouse and gamepad
- [ ] 234 | Spring UI Motion | TypeScript | Starter | Audio, UI & Tools | Spring physics for UI transitions
- [ ] 235 | Cheat Console | Rust | Starter | Audio, UI & Tools | Command console with parsing, autocomplete, and history
- [ ] 236 | Asset Usage Auditor | Go | Starter | Audio, UI & Tools | CLI that indexes assets and finds unused or missing references
- [ ] 237 | Shader Playground | TypeScript + WebGL2 | Intermediate | Audio, UI & Tools | Live GLSL editor with uniforms and error reporting
- [ ] 238 | Behavior Tree Editor | TypeScript + React | Advanced | Audio, UI & Tools | Visual behavior tree editor with live debugging
- [ ] 239 | Automated Playtest Bot | Python | Intermediate | Audio, UI & Tools | Scripted bot that explores a game to find soft-locks
- [ ] 240 | Tools Capstone: Level Editor Suite | TypeScript + React | Advanced | Audio, UI & Tools | Level editor with tilemaps, entities, validation, and playtest mode

## Track 09 — Networking & Multiplayer

_Netcode, backend services, and online play_

- [ ] 241 | Game Packets over UDP | Go | Starter | Networking & Multiplayer | Packet framing, serialization, and a UDP echo test bed
- [ ] 242 | Reliable UDP | C++ | Advanced | Networking & Multiplayer | Sequence numbers, acks, and resends on top of UDP
- [ ] 243 | WebSocket Pong | TypeScript + Node | Starter | Networking & Multiplayer | Two-player Pong over WebSockets
- [ ] 244 | Authoritative Server | Go | Intermediate | Networking & Multiplayer | Server-authoritative simulation with clients as input senders
- [ ] 245 | Client Prediction & Reconciliation | TypeScript + Node | Advanced | Networking & Multiplayer | Prediction with server reconciliation and input replay
- [ ] 246 | Entity Interpolation | TypeScript + Canvas | Intermediate | Networking & Multiplayer | Snapshot buffering and interpolation of remote entities
- [ ] 247 | Lag Compensation | C# (.NET) | Advanced | Networking & Multiplayer | Server-side rewind for hit detection
- [ ] 248 | Deterministic Lockstep | Rust | Advanced | Networking & Multiplayer | Lockstep RTS networking with checksum desync detection
- [ ] 249 | Rollback Netcode | Rust | Advanced | Networking & Multiplayer | Rollback with input delay, prediction, and resimulation
- [ ] 250 | Delta Snapshot Compression | C++ | Intermediate | Networking & Multiplayer | Delta-compressed snapshots against acked baselines
- [ ] 251 | Bit-Packing Serializer | Rust | Intermediate | Networking & Multiplayer | Bit-level serializer with quantization
- [ ] 252 | Matchmaking Service | Go | Intermediate | Networking & Multiplayer | Skill-based queue with widening search windows
- [ ] 253 | Lobbies & Rooms | TypeScript + Node | Intermediate | Networking & Multiplayer | Room creation, join codes, and ready checks
- [ ] 254 | Network Condition Simulator | Python | Starter | Networking & Multiplayer | Proxy that injects latency, jitter, and packet loss
- [ ] 255 | Async Turn-Based Multiplayer | Python + FastAPI | Intermediate | Networking & Multiplayer | Turn submission, notifications, and game history
- [ ] 256 | Game Chat Server | Go | Starter | Networking & Multiplayer | Channels, rate limiting, and moderation hooks
- [ ] 257 | Redis Leaderboards | Go + Redis | Intermediate | Networking & Multiplayer | Sorted-set leaderboards with seasons and pagination
- [ ] 258 | Player Accounts | TypeScript + Node | Intermediate | Networking & Multiplayer | Registration, sessions, and token refresh
- [ ] 259 | Server-Side Validation | C# (.NET) | Intermediate | Networking & Multiplayer | Defensive server checks for movement, rate limits, and impossible actions
- [ ] 260 | Interest Management | C++ | Advanced | Networking & Multiplayer | Area-of-interest filtering for replication
- [ ] 261 | WebRTC Peer-to-Peer | TypeScript | Intermediate | Networking & Multiplayer | P2P data channels with signaling server
- [ ] 262 | MMO Zone Sharding | Go | Advanced | Networking & Multiplayer | Zone servers with player handoff
- [ ] 263 | Glicko-2 Ratings | Python | Starter | Networking & Multiplayer | Glicko-2 rating system with tests against reference values
- [ ] 264 | Dedicated Server Orchestrator | Go + Docker | Intermediate | Networking & Multiplayer | Allocates and recycles game server instances
- [ ] 265 | Property Replication | C# (.NET) | Advanced | Networking & Multiplayer | Replicated properties with dirty tracking and RPCs
- [ ] 266 | Net Stats HUD | TypeScript | Starter | Networking & Multiplayer | RTT, jitter, and packet-loss graphs
- [ ] 267 | Multiplayer Snake Arena | TypeScript + Node | Intermediate | Networking & Multiplayer | .io-style multiplayer snake
- [ ] 268 | Cloud Saves API | Python + FastAPI | Intermediate | Networking & Multiplayer | Cloud saves with conflict detection and resolution
- [ ] 269 | Game Server Load Testing | Go | Intermediate | Networking & Multiplayer | Bot clients that stress-test a game server
- [ ] 270 | Netcode Capstone: Arena Shooter | TypeScript + Node | Advanced | Networking & Multiplayer | Multiplayer arena shooter with prediction, interpolation, and lag compensation

## Track 10 — 3D Graphics & Engines

_3D math, rendering, and 3D gameplay_

- [ ] 271 | Software Rasterizer | C++ | Advanced | 3D Graphics & Engines | Triangle rasterizer with depth buffer, perspective-correct texturing, and clipping
- [ ] 272 | Wolfenstein Raycaster | C++ + SDL2 | Intermediate | 3D Graphics & Engines | Grid raycasting renderer with textured walls and sprites
- [ ] 273 | First-Person Controller | JavaScript + Three.js | Starter | 3D Graphics & Engines | Pointer-lock FPS controller with collision
- [ ] 274 | 3D Math Library | Rust | Intermediate | 3D Graphics & Engines | Vectors, matrices, and quaternions with property-based tests
- [ ] 275 | 3D Camera Rigs | C++ + raylib | Intermediate | 3D Graphics & Engines | Orbit, first-person, and third-person cameras with collision
- [ ] 276 | glTF Loader | Rust | Intermediate | 3D Graphics & Engines | glTF 2.0 mesh, material, and node hierarchy loader
- [ ] 277 | Phong & Blinn-Phong Lighting | TypeScript + WebGL2 | Starter | 3D Graphics & Engines | Classic lighting models with multiple lights
- [ ] 278 | PBR Shading | TypeScript + WebGL2 | Advanced | 3D Graphics & Engines | Physically based rendering with image-based lighting
- [ ] 279 | Shadow Mapping | C++ + OpenGL | Advanced | 3D Graphics & Engines | Directional shadow maps with PCF filtering
- [ ] 280 | Skybox & Reflections | JavaScript + Three.js | Starter | 3D Graphics & Engines | Cubemap skyboxes and environment reflections
- [ ] 281 | Frustum Culling & BVH | C++ | Intermediate | 3D Graphics & Engines | Bounding volume hierarchy with frustum culling
- [ ] 282 | Terrain LOD | JavaScript + Three.js | Advanced | 3D Graphics & Engines | Quadtree terrain level of detail with crack fixing
- [ ] 283 | Voxel Engine | C++ + raylib | Advanced | 3D Graphics & Engines | Chunked voxels with greedy meshing and block editing
- [ ] 284 | 3D Skeletal Animation | JavaScript + Three.js | Intermediate | 3D Graphics & Engines | Skinned glTF characters with animation blending
- [ ] 285 | 3D Physics with Rapier | Rust | Intermediate | 3D Graphics & Engines | Rigid bodies, colliders, and a character controller
- [ ] 286 | Post-Processing Stack | TypeScript + WebGL2 | Intermediate | 3D Graphics & Engines | Bloom, tone mapping, and FXAA
- [ ] 287 | Deferred Renderer | C++ + OpenGL | Advanced | 3D Graphics & Engines | G-buffer deferred shading with many lights
- [ ] 288 | GPU Particles | TypeScript + WebGL2 | Advanced | 3D Graphics & Engines | Transform-feedback particles simulated on the GPU
- [ ] 289 | CPU Ray Tracer | Rust | Intermediate | 3D Graphics & Engines | Path tracer with materials and multithreading
- [ ] 290 | Toon Shading & Outlines | JavaScript + Three.js | Starter | 3D Graphics & Engines | Cel shading and inverted-hull outlines
- [ ] 291 | Water Shader | TypeScript + WebGL2 | Intermediate | 3D Graphics & Engines | Gerstner waves, fresnel, and foam
- [ ] 292 | Instanced Grass | JavaScript + Three.js | Intermediate | 3D Graphics & Engines | GPU-instanced grass with wind
- [ ] 293 | Third-Person Controller | Godot 4 + GDScript | Intermediate | 3D Graphics & Engines | Third-person movement, camera, and animation tree
- [ ] 294 | Recast Navmesh | C++ | Advanced | 3D Graphics & Engines | 3D navmesh generation and pathing with Recast/Detour
- [ ] 295 | Decals | JavaScript + Three.js | Intermediate | 3D Graphics & Engines | Projected decals for bullet holes and splats
- [ ] 296 | Scene Graph | TypeScript | Starter | 3D Graphics & Engines | Transform hierarchy with dirty-flag world matrix caching
- [ ] 297 | Raymarched SDF Scenes | TypeScript + WebGL2 | Intermediate | 3D Graphics & Engines | Signed distance field scenes with soft shadows
- [ ] 298 | 3D Platformer | Godot 4 + GDScript | Intermediate | 3D Graphics & Engines | 3D platformer with collectibles and checkpoints
- [ ] 299 | Godot C# Gameplay | Godot 4 + C# | Intermediate | 3D Graphics & Engines | Gameplay scripts in C# with logic unit-tested outside the engine
- [ ] 300 | 3D Capstone: Low-Poly Exploration | JavaScript + Three.js | Advanced | 3D Graphics & Engines | Exploration game with terrain, day/night, collectibles, and audio

## Track 11 — Performance & Internals

_Profiling, data-oriented design, VMs, and emulators_

- [ ] 301 | Frame Profiler | C++ | Intermediate | Performance & Internals | Scoped timers with Chrome trace export
- [ ] 302 | Data-Oriented Design | C++ | Intermediate | Performance & Internals | AoS vs SoA benchmarks on a particle workload
- [ ] 303 | SIMD Math | C++ | Advanced | Performance & Internals | SIMD vector math with benchmarks and scalar fallbacks
- [ ] 304 | Cache-Friendly Updates | Rust | Intermediate | Performance & Internals | Cache-aware entity updates with benchmark comparisons
- [ ] 305 | Multithreaded Game Loop | C++ | Advanced | Performance & Internals | Render/update thread split with double-buffered state
- [ ] 306 | Async Asset Streaming | Rust | Intermediate | Performance & Internals | Prioritized background loading with cancellation
- [ ] 307 | WebAssembly Game | Rust + WebAssembly | Intermediate | Performance & Internals | Rust game compiled to WASM with a JS host
- [ ] 308 | Scripting Bytecode VM | C | Advanced | Performance & Internals | Stack-based VM and compiler for a tiny game scripting language
- [ ] 309 | Game Event DSL | Go | Advanced | Performance & Internals | Lexer, parser, and interpreter for a game event DSL
- [ ] 310 | Hot Code Reload | C | Advanced | Performance & Internals | Game code in a shared library reloaded at runtime
- [ ] 311 | Frame Pacing Analysis | C++ + SDL2 | Intermediate | Performance & Internals | Frame-time measurement, vsync modes, and stutter detection
- [ ] 312 | Allocation-Free Gameplay | C# (.NET) | Intermediate | Performance & Internals | Zero-allocation update loops verified with allocation tests
- [ ] 313 | Dynamic BVH | Rust | Advanced | Performance & Internals | Dynamic AABB tree for queries and raycasts
- [ ] 314 | Fixed-Point Math | C | Intermediate | Performance & Internals | Fixed-point types, trig tables, and accuracy tests
- [ ] 315 | Asset Compression | C++ | Intermediate | Performance & Internals | LZ77-style compressor for game assets
- [ ] 316 | Pak Files & Virtual FS | C++ | Intermediate | Performance & Internals | Archive format with a mountable virtual file system
- [ ] 317 | Input Latency Measurement | TypeScript | Starter | Performance & Internals | Tools to measure and reduce input-to-display latency
- [ ] 318 | CHIP-8 Emulator | Rust | Intermediate | Performance & Internals | Complete CHIP-8 interpreter with public-domain test programs
- [ ] 319 | Game Boy CPU Core | C++ | Advanced | Performance & Internals | LR35902 CPU core validated with public test ROMs (no commercial ROMs)
- [ ] 320 | 6502 CPU Core | Rust | Advanced | Performance & Internals | Cycle-counted 6502 core validated with open test suites
- [ ] 321 | Tile & Sprite PPU | Rust | Advanced | Performance & Internals | Retro-style tile and sprite picture processing unit
- [ ] 322 | Fantasy CPU Assembler | Python | Intermediate | Performance & Internals | Assembler and VM for a small fantasy CPU
- [ ] 323 | Lua Hot-Swap Logic | C++ + Lua | Intermediate | Performance & Internals | Gameplay logic in Lua reloaded without restarting
- [ ] 324 | Benchmark Regression Suite | Rust | Intermediate | Performance & Internals | Criterion benchmarks with regression thresholds
- [ ] 325 | Memory Leak Detector | C++ | Intermediate | Performance & Internals | Allocation tracking with leak reports
- [ ] 326 | ECS Benchmark Shootout | Rust | Intermediate | Performance & Internals | Comparing ECS storage strategies under game workloads
- [ ] 327 | Adaptive Quality Scaling | TypeScript + Canvas | Starter | Performance & Internals | Frame budgets and dynamic quality settings
- [ ] 328 | Instanced 2D Rendering | C++ + raylib | Intermediate | Performance & Internals | Rendering 100k sprites with instancing
- [ ] 329 | Determinism Test Harness | C++ | Intermediate | Performance & Internals | Harness that detects nondeterminism in simulations
- [ ] 330 | Internals Capstone: Fantasy Console | C + Lua | Advanced | Performance & Internals | PICO-8-style fantasy console with editor, VM limits, and cartridges

## Track 12 — Capstones & Live Ops

_Complete games and production game services_

- [ ] 331 | Live-Ops Event Scheduler | Go | Intermediate | Capstones & Live Ops | Time-limited events with schedules and rewards
- [ ] 332 | Store Backend (Mock Payments) | TypeScript + Node | Intermediate | Capstones & Live Ops | Catalog, purchases, receipts, and idempotency with a mocked payment provider
- [ ] 333 | Inventory Microservice | Go + PostgreSQL | Intermediate | Capstones & Live Ops | Player inventory service with transactions
- [ ] 334 | Battle Pass Service | Python + FastAPI | Intermediate | Capstones & Live Ops | Seasonal progression tracks and rewards
- [ ] 335 | Telemetry Ingestion Pipeline | Go + Kafka | Advanced | Capstones & Live Ops | Event ingestion with batching and schema validation
- [ ] 336 | A/B Testing for Tuning | Python | Intermediate | Capstones & Live Ops | Experiment assignment and significance testing for game tuning
- [ ] 337 | Modding SDK | Lua + LÖVE | Intermediate | Capstones & Live Ops | Mod loader, API docs, and example mods
- [ ] 338 | Achievements Service | C# + ASP.NET Core | Intermediate | Capstones & Live Ops | Achievement and stats API
- [ ] 339 | Discord Mini-Game Bot | TypeScript + Node | Starter | Capstones & Live Ops | Trivia mini-game bot with mocked Discord API in tests
- [ ] 340 | Game Jam Template | TypeScript + Phaser | Starter | Capstones & Live Ops | Ready-to-jam template with scenes, audio, and build scripts
- [ ] 341 | Capstone: Bullet Hell | C++ + raylib | Advanced | Capstones & Live Ops | Bullet hell with pattern scripting and boss phases
- [ ] 342 | Capstone: Precision Platformer | Go + Ebitengine | Advanced | Capstones & Live Ops | Platformer with level select, timers, and replays
- [ ] 343 | Capstone: Card Roguelike | TypeScript + Canvas | Advanced | Capstones & Live Ops | Deckbuilding roguelike with map, relics, and events
- [ ] 344 | Capstone: Tower Defense | C# + MonoGame | Advanced | Capstones & Live Ops | Full tower defense with campaign and upgrades
- [ ] 345 | Capstone: Twin-Stick Roguelite | Rust + macroquad | Advanced | Capstones & Live Ops | Roguelite with procedural rooms and meta progression
- [ ] 346 | Capstone: Puzzle Game & Editor | Lua + LÖVE | Advanced | Capstones & Live Ops | Puzzle game with a built-in level editor and sharing
- [ ] 347 | Capstone: Turn-Based Tactics | Python + pygame | Advanced | Capstones & Live Ops | Tactics game with campaign and AI
- [ ] 348 | Capstone: Arcade Racer | JavaScript + Three.js | Advanced | Capstones & Live Ops | 3D racer with AI opponents and ghost laps
- [ ] 349 | Capstone: Survival Crafting | Godot 4 + GDScript | Advanced | Capstones & Live Ops | Survival game with gathering, crafting, and day/night
- [ ] 350 | Capstone: Fighting Game | C++ + SDL2 | Advanced | Capstones & Live Ops | Two-character fighter with frame data and training mode
- [ ] 351 | Capstone: Rhythm Game | TypeScript + Web Audio | Advanced | Capstones & Live Ops | Rhythm game with chart editor and calibration
- [ ] 352 | Capstone: Colony Sim | Rust | Advanced | Capstones & Live Ops | Colony simulation with jobs, needs, and pathing
- [ ] 353 | Capstone: Party Game | TypeScript + Node | Advanced | Capstones & Live Ops | Multiplayer party game with phone-as-controller
- [ ] 354 | Capstone: Metroidvania | C# + MonoGame | Advanced | Capstones & Live Ops | Metroidvania with abilities gating exploration
- [ ] 355 | Capstone: Interactive Fiction Engine | Python | Intermediate | Capstones & Live Ops | Parser-based text adventure engine with save games
- [ ] 356 | Capstone: Online Deckbuilder | Go + TypeScript | Advanced | Capstones & Live Ops | Two-player online card game with server-authoritative rules
- [ ] 357 | Capstone: 3D Arena | JavaScript + Three.js | Advanced | Capstones & Live Ops | 3D arena combat with enemy waves
- [ ] 358 | Capstone: Factory Game | Rust + Bevy | Advanced | Capstones & Live Ops | Factory automation game on Bevy ECS
- [ ] 359 | Capstone: Space Trading Sim | C# (.NET) | Advanced | Capstones & Live Ops | Trading simulation with economy and NPC traders
- [ ] 360 | Capstone: Chess Platform | TypeScript + Node | Advanced | Capstones & Live Ops | Online chess with clocks, ratings, and an engine opponent
- [ ] 361 | Capstone: Voxel Sandbox | C++ + raylib | Advanced | Capstones & Live Ops | Voxel sandbox with building, lighting, and saving
- [ ] 362 | Capstone: Endless Runner | TypeScript + Phaser | Intermediate | Capstones & Live Ops | Mobile-friendly runner with touch controls
- [ ] 363 | Capstone: Moddable Roguelike | Lua + LÖVE | Advanced | Capstones & Live Ops | Roguelike whose content is entirely data and mods
- [ ] 364 | Year Retrospective & Repo Stats | Python | Intermediate | Capstones & Live Ops | Analyzes all projects in this repo: languages, tests, concepts, and growth
- [ ] 365 | Portfolio Site for All 365 Projects | TypeScript + Vite | Intermediate | Capstones & Live Ops | Static site indexing every project with filters by track, stack, and level
