import { describe, expect, it } from "vitest";
import { DEFAULT_CONFIG, type GameConfig } from "../src/core/config";
import { fromRows, maxValue, tiles, toRows, type Grid } from "../src/core/grid";
import { createHistory, undoable } from "../src/core/history";
import { DIRECTIONS, type Direction } from "../src/core/moves";
import { createReducer, initialState, spawnTile, type Action, type GameState } from "../src/core/reducer";
import { nextInt, seedRng } from "../src/core/rng";

const reduce = createReducer(DEFAULT_CONFIG);
const move = (direction: Direction): Action => ({ type: "move", direction });

/** A playing state with a hand-built grid. Ids above any fixture id. */
function stateWith(grid: Grid, overrides: Partial<GameState> = {}): GameState {
  return { ...initialState(DEFAULT_CONFIG, 1), grid, nextId: 1000, ...overrides };
}

function deepFreeze<T>(value: T): T {
  if (value && typeof value === "object" && !Object.isFrozen(value)) {
    Object.freeze(value);
    for (const v of Object.values(value)) deepFreeze(v);
  }
  return value;
}

describe("initial state", () => {
  it("places startTiles tiles of value 2 or 4", () => {
    const s = initialState(DEFAULT_CONFIG, 123);
    const placed = tiles(s.grid);
    expect(placed).toHaveLength(2);
    expect(placed.every((t) => t.tile.value === 2 || t.tile.value === 4)).toBe(true);
    expect(s.turn.spawns.map((sp) => sp.id)).toEqual(placed.map((t) => t.tile.id).sort());
    expect(s).toMatchObject({ score: 0, moves: 0, status: "playing", seed: 123 });
  });

  it("is reproducible from the seed", () => {
    expect(initialState(DEFAULT_CONFIG, 99)).toEqual(initialState(DEFAULT_CONFIG, 99));
    expect(toRows(initialState(DEFAULT_CONFIG, 99).grid)).not.toEqual(toRows(initialState(DEFAULT_CONFIG, 100).grid));
  });

  it("rejects an invalid config", () => {
    expect(() => createReducer({ ...DEFAULT_CONFIG, size: 0 })).toThrow(RangeError);
  });
});

describe("spawnTile", () => {
  it("picks an empty cell with the rng and respects fourChance", () => {
    const g = fromRows(["2 .", ". 2"]);
    const rng = seedRng(5);
    const [index] = nextInt(rng, 2);
    const always4 = spawnTile(g, rng, 10, 1);
    expect(always4.spawn).toEqual({ id: 10, value: 4, at: [{ x: 1, y: 0 }, { x: 0, y: 1 }][index] });
    expect(spawnTile(g, rng, 10, 0).spawn?.value).toBe(2);
    expect(always4.nextId).toBe(11);
  });

  it("does nothing on a full board", () => {
    const g = fromRows(["2 4", "8 16"]);
    const res = spawnTile(g, seedRng(1), 3, 0.1);
    expect(res.spawn).toBeNull();
    expect(res.grid).toBe(g);
  });
});

describe("move action", () => {
  it("slides, scores, counts the move, and spawns exactly one tile", () => {
    const s = stateWith(fromRows(["2 2 . .", ". . . .", ". . . .", ". . . ."]));
    const next = reduce(s, move("left"));
    expect(next.score).toBe(4);
    expect(next.moves).toBe(1);
    expect(tiles(next.grid)).toHaveLength(2);
    expect(next.grid[0]![0]!.value).toBe(4);
    expect(next.turn.spawns).toHaveLength(1);
    expect(next.turn.direction).toBe("left");
    expect(next.rng).not.toBe(s.rng);
  });

  it("returns the same reference for a blocked move: no spawn, no rng advance", () => {
    const s = stateWith(fromRows(["2 4 . .", ". . . .", ". . . .", ". . . ."]));
    expect(reduce(s, move("left"))).toBe(s);
    expect(reduce(s, move("up"))).toBe(s);
  });

  it("never mutates its input", () => {
    const s = deepFreeze(stateWith(fromRows(["2 2 4 4", "2 . . 2", ". 8 8 .", "4 . . 4"])));
    const snapshot = structuredClone(s);
    for (const d of DIRECTIONS) reduce(s, move(d));
    expect(s).toEqual(snapshot);
  });

  it("is deterministic: same state and action give equal results", () => {
    const s = initialState(DEFAULT_CONFIG, 2024);
    expect(reduce(s, move("down"))).toEqual(reduce(s, move("down")));
  });
});

describe("win and loss", () => {
  const small: GameConfig = { ...DEFAULT_CONFIG, size: 2, target: 8, startTiles: 1, fourChance: 0 };
  const reduceSmall = createReducer(small);

  it("wins on reaching the target and freezes until the player continues", () => {
    const s = { ...initialState(small, 1), grid: fromRows(["4 4", ". ."]), nextId: 100 };
    const won = reduceSmall(s, move("left"));
    expect(won.status).toBe("won");
    expect(maxValue(won.grid)).toBe(8);
    expect(reduceSmall(won, move("right"))).toBe(won);

    const cont = reduceSmall(won, { type: "keepPlaying" });
    expect(cont).toMatchObject({ status: "playing", keepPlaying: true });
    // Reaching the target again doesn't re-trigger the win screen.
    const after = reduceSmall({ ...cont, grid: fromRows(["4 4", ". 8"]) }, move("left"));
    expect(after.status).not.toBe("won");
  });

  it("keepPlaying is ignored unless the game is won", () => {
    const s = initialState(small, 1);
    expect(reduceSmall(s, { type: "keepPlaying" })).toBe(s);
  });

  it("reports lost for a full board with no merges after the spawn", () => {
    // 3x3 with one gap: sliding right opens (0, 0) and the spawn (always a 2 here) fills it.
    const cfg: GameConfig = { ...DEFAULT_CONFIG, size: 3, target: 2048, startTiles: 1, fourChance: 0 };
    const s = { ...initialState(cfg, 1), grid: fromRows(["4 8 .", "8 4 2", "4 8 4"]), nextId: 100 };
    const next = createReducer(cfg)(s, move("right"));
    expect(toRows(next.grid)).toEqual(["2 4 8", "8 4 2", "4 8 4"]);
    expect(next.status).toBe("playing"); // the 4s at (1,0) and (1,1) still merge vertically

    const s2 = { ...s, grid: fromRows(["16 8 .", "8 4 2", "4 8 4"]) };
    const lost = createReducer(cfg)(s2, move("right"));
    expect(toRows(lost.grid)).toEqual(["2 16 8", "8 4 2", "4 8 4"]);
    expect(lost.status).toBe("lost");
  });
});

/** Independent oracle for loss detection used above. */
function hasAnyMoveRows(grid: Grid): boolean {
  const rows = toRows(grid).map((r) => r.split(" "));
  for (let y = 0; y < rows.length; y++) {
    for (let x = 0; x < rows.length; x++) {
      const v = rows[y]![x];
      if (v === "." || v === rows[y]![x + 1] || v === rows[y + 1]?.[x]) return true;
    }
  }
  return false;
}

describe("newGame action", () => {
  it("resets everything from the given seed", () => {
    const played = reduce(initialState(DEFAULT_CONFIG, 1), move("left"));
    expect(reduce(played, { type: "newGame", seed: 77 })).toEqual(initialState(DEFAULT_CONFIG, 77));
  });
});

describe("undoable", () => {
  const reduceH = undoable(reduce, { limit: 3, resets: (a: Action) => a.type === "newGame" });
  const start = () => createHistory(stateWith(fromRows(["2 2 . .", ". . . .", ". . . .", "4 . . ."])));

  it("undo restores the exact previous state and redo replays it", () => {
    const h0 = start();
    const h1 = reduceH(h0, move("left"));
    const undone = reduceH(h1, { type: "undo" });
    expect(undone.present).toBe(h0.present);
    expect(reduceH(undone, { type: "redo" }).present).toBe(h1.present);
  });

  it("repeating an undone move spawns the same tile (no rerolling)", () => {
    const h1 = reduceH(start(), move("left"));
    const again = reduceH(reduceH(h1, { type: "undo" }), move("left"));
    expect(again.present).toEqual(h1.present);
  });

  it("does not record blocked moves and caps history at the limit", () => {
    const h0 = start();
    expect(reduceH(h0, move("up")).past).toHaveLength(1);
    let h = h0;
    for (const d of ["left", "right", "left", "right", "down", "up"] as const) h = reduceH(h, move(d));
    expect(h.past.length).toBeLessThanOrEqual(3);
    const blocked = reduceH(createHistory(stateWith(fromRows(["2 4 . .", ". . . .", ". . . .", ". . . ."]))), move("left"));
    expect(blocked.past).toHaveLength(0);
  });

  it("a new game clears history; undo with no past is a no-op", () => {
    const h = reduceH(reduceH(start(), move("left")), { type: "newGame", seed: 5 });
    expect(h.past).toHaveLength(0);
    expect(reduceH(h, { type: "undo" })).toBe(h);
    expect(reduceH(h, { type: "redo" })).toBe(h);
  });

  it("a new move after undo discards the redo branch", () => {
    const h = reduceH(reduceH(reduceH(start(), move("left")), { type: "undo" }), move("down"));
    expect(h.future).toHaveLength(0);
  });

  it("rejects a bad limit", () => {
    expect(() => undoable(reduce, { limit: 0 })).toThrow(RangeError);
  });
});

describe("soak: random games keep their invariants", () => {
  it("holds for 200 seeded games played to the end", () => {
    for (let seed = 1; seed <= 200; seed++) {
      let s = initialState(DEFAULT_CONFIG, seed);
      let pick = seedRng(seed * 31);
      let spawnedSum = tiles(s.grid).reduce((sum, t) => sum + t.tile.value, 0);
      let mergedSum = 0;
      for (let guard = 0; s.status === "playing" && guard < 5000; guard++) {
        const [i, nextPick] = nextInt(pick, 4);
        pick = nextPick;
        const next = reduce(s, move(DIRECTIONS[i]!));
        if (next === s) continue;
        spawnedSum += next.turn.spawns.reduce((sum, sp) => sum + sp.value, 0);
        mergedSum += next.turn.gained;
        s = next;

        const ids = tiles(s.grid).map((t) => t.tile.id);
        expect(new Set(ids).size).toBe(ids.length);
        expect(ids.every((id) => id < s.nextId)).toBe(true);
      }
      expect(s.status).not.toBe("playing");
      // Merges conserve value, so the board total equals the total ever spawned.
      expect(tiles(s.grid).reduce((sum, t) => sum + t.tile.value, 0)).toBe(spawnedSum);
      expect(s.score).toBe(mergedSum);
      if (s.status === "lost") expect(hasAnyMoveRows(s.grid)).toBe(false);
    }
  });
});
