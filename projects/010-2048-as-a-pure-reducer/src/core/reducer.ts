/**
 * The whole game as one pure function: `(state, action) => state`.
 *
 * - No mutation: every input is treated as read-only, and a new state is
 *   returned only when something changed. A no-op (a blocked move, a move after
 *   game over) returns the *same reference*, which callers can use for cheap
 *   change detection (`next !== prev`).
 * - No hidden inputs: randomness is the `rng` field of the state and the clock
 *   is never read, so the next state depends only on the arguments.
 */
import { validateConfig, type GameConfig } from "./config";
import { emptyCells, emptyGrid, maxValue, withCell, type Grid, type Pos } from "./grid";
import { applyMove, hasAnyMove, type Direction, type MergeEvent, type SlideEvent } from "./moves";
import { nextInt, nextRandom, seedRng, type RngState } from "./rng";

export type Status = "playing" | "won" | "lost";

export interface SpawnEvent {
  readonly id: number;
  readonly value: number;
  readonly at: Pos;
}

/** What happened on the last accepted action, for animation and HUD only. */
export interface TurnInfo {
  readonly direction: Direction | null;
  readonly slides: readonly SlideEvent[];
  readonly merges: readonly MergeEvent[];
  readonly spawns: readonly SpawnEvent[];
  readonly gained: number;
}

export interface GameState {
  readonly grid: Grid;
  readonly score: number;
  readonly status: Status;
  /** True after the player chose to continue past the target tile. */
  readonly keepPlaying: boolean;
  readonly moves: number;
  readonly seed: number;
  readonly rng: RngState;
  readonly nextId: number;
  readonly turn: TurnInfo;
}

export type Action =
  | { readonly type: "move"; readonly direction: Direction }
  | { readonly type: "newGame"; readonly seed: number }
  | { readonly type: "keepPlaying" };

export type Reducer<S, A> = (state: S, action: A) => S;

interface Spawned {
  readonly grid: Grid;
  readonly rng: RngState;
  readonly nextId: number;
  readonly spawn: SpawnEvent | null;
}

/** Place one 2 (or, with `fourChance`, a 4) on a random empty cell. */
export function spawnTile(grid: Grid, rng: RngState, nextId: number, fourChance: number): Spawned {
  const empty = emptyCells(grid);
  if (empty.length === 0) return { grid, rng, nextId, spawn: null };
  const [index, r1] = nextInt(rng, empty.length);
  const [roll, r2] = nextRandom(r1);
  const spawn: SpawnEvent = { id: nextId, value: roll < fourChance ? 4 : 2, at: empty[index]! };
  return { grid: withCell(grid, spawn.at, { id: spawn.id, value: spawn.value }), rng: r2, nextId: nextId + 1, spawn };
}

export function initialState(config: GameConfig, seed: number): GameState {
  validateConfig(config);
  let s: Spawned = { grid: emptyGrid(config.size), rng: seedRng(seed), nextId: 1, spawn: null };
  const spawns: SpawnEvent[] = [];
  for (let i = 0; i < config.startTiles; i++) {
    s = spawnTile(s.grid, s.rng, s.nextId, config.fourChance);
    if (s.spawn) spawns.push(s.spawn);
  }
  return {
    grid: s.grid,
    score: 0,
    status: hasAnyMove(s.grid) ? "playing" : "lost",
    keepPlaying: false,
    moves: 0,
    seed,
    rng: s.rng,
    nextId: s.nextId,
    turn: { direction: null, slides: [], merges: [], spawns, gained: 0 },
  };
}

function statusAfterMove(grid: Grid, config: GameConfig, keepPlaying: boolean): Status {
  if (!keepPlaying && maxValue(grid) >= config.target) return "won";
  return hasAnyMove(grid) ? "playing" : "lost";
}

/** Builds the reducer for a config. The config is validated once, here. */
export function createReducer(config: GameConfig): Reducer<GameState, Action> {
  validateConfig(config);
  return (state, action) => {
    switch (action.type) {
      case "newGame":
        return initialState(config, action.seed);

      case "keepPlaying":
        if (state.status !== "won") return state;
        return { ...state, status: hasAnyMove(state.grid) ? "playing" : "lost", keepPlaying: true };

      case "move": {
        if (state.status !== "playing") return state;
        const moved = applyMove(state.grid, action.direction, state.nextId);
        if (!moved.moved) return state; // blocked move: no spawn, same reference
        const s = spawnTile(moved.grid, state.rng, moved.nextId, config.fourChance);
        return {
          ...state,
          grid: s.grid,
          score: state.score + moved.gained,
          status: statusAfterMove(s.grid, config, state.keepPlaying),
          moves: state.moves + 1,
          rng: s.rng,
          nextId: s.nextId,
          turn: {
            direction: action.direction,
            slides: moved.slides,
            merges: moved.merges,
            spawns: s.spawn ? [s.spawn] : [],
            gained: moved.gained,
          },
        };
      }
    }
  };
}
