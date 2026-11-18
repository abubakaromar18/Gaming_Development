/**
 * Board generation and reshuffling. A playable board must start with no
 * matches (or the player gets free points before touching anything) and must
 * contain at least one legal move.
 */
import { createEmpty, type Board, type Cell, type Gem } from "./board";
import { validateConfig, type Match3Config } from "./config";
import { hasAnyMatch } from "./match";
import { hasMoves } from "./moves";
import { randomInt, type Rng } from "./rng";

/**
 * Fills cells left-to-right, top-to-bottom. For each cell it forbids the
 * color that would complete a run with the two cells to the left or the two
 * cells above. That rules out at most two colors, so with 3+ colors there is
 * always a choice and the board is match-free by construction.
 */
function fillWithoutMatches(cfg: Match3Config, rng: Rng): Board {
  const { width, height, colors } = cfg;
  const cells: Cell[] = createEmpty(width, height).cells.slice();
  const at = (x: number, y: number): Cell => (x < 0 || y < 0 ? null : (cells[y * width + x] ?? null));
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const forbidden = new Set<Gem>();
      const left = at(x - 1, y);
      if (left !== null && left === at(x - 2, y)) forbidden.add(left);
      const up = at(x, y - 1);
      if (up !== null && up === at(x, y - 2)) forbidden.add(up);
      const allowed: Gem[] = [];
      for (let g = 0; g < colors; g++) if (!forbidden.has(g)) allowed.push(g);
      cells[y * width + x] = allowed[randomInt(rng, allowed.length)] ?? 0;
    }
  }
  return { width, height, cells };
}

/** A fresh board with no matches and at least one move. */
export function generateBoard(cfg: Match3Config, rng: Rng): Board {
  validateConfig(cfg);
  for (let attempt = 0; attempt < cfg.maxGenerateAttempts; attempt++) {
    const board = fillWithoutMatches(cfg, rng);
    if (hasMoves(board)) return board;
  }
  throw new Error(`no playable board found in ${cfg.maxGenerateAttempts} attempts`);
}

/**
 * Rearranges the existing gems (Fisher–Yates) until the board has no
 * matches and at least one move. Keeping the same gems is what players expect
 * from a "no more moves" shuffle. If the gem mix can't form such a board in
 * the allotted attempts, it falls back to a freshly generated board.
 */
export function shuffleBoard(board: Board, rng: Rng, cfg: Match3Config): Board {
  for (let attempt = 0; attempt < cfg.maxGenerateAttempts; attempt++) {
    const cells = board.cells.slice();
    for (let i = cells.length - 1; i > 0; i--) {
      const j = randomInt(rng, i + 1);
      [cells[i], cells[j]] = [cells[j] ?? null, cells[i] ?? null];
    }
    const candidate = { ...board, cells };
    if (!hasAnyMatch(candidate) && hasMoves(candidate)) return candidate;
  }
  return generateBoard(cfg, rng);
}
