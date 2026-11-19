package gamedev.minesweeper.core;

import java.util.random.RandomGenerator;

/**
 * Immutable mine layout plus the precomputed "adjacent mines" number for every cell.
 * This is the hidden truth of a game; only {@link Game} reads it, and the solver
 * never sees it (it works from {@link PlayerView}).
 */
public final class MineField {

    private final Grid grid;
    private final boolean[] mines;
    private final int[] adjacent;
    private final int mineCount;

    private MineField(Grid grid, boolean[] mines) {
        this.grid = grid;
        this.mines = mines;
        this.adjacent = new int[grid.size()];
        int count = 0;
        for (int i = 0; i < mines.length; i++) {
            if (mines[i]) {
                count++;
                for (int n : grid.neighbors(i)) {
                    adjacent[n]++;
                }
            }
        }
        this.mineCount = count;
    }

    /**
     * Places mines uniformly at random, keeping the first click safe.
     *
     * <p>Modern Minesweeper keeps the whole 3x3 block around the first click clear,
     * so the opening click always lands on a 0 and floods open a region. If the board
     * is too dense for that, only the clicked cell itself is protected.
     *
     * <p>Selection is a partial Fisher-Yates shuffle over the allowed cells: exactly
     * {@code mines} draws, no rejection sampling, and every layout is equally likely.
     */
    public static MineField generate(BoardConfig config, Coord firstClick, RandomGenerator rng) {
        Grid grid = new Grid(config.width(), config.height());
        int first = grid.index(firstClick);

        boolean[] protectedCell = new boolean[grid.size()];
        protectedCell[first] = true;
        int[] around = grid.neighbors(first);
        if (grid.size() - 1 - around.length >= config.mines()) {
            for (int n : around) {
                protectedCell[n] = true;
            }
        }

        int[] candidates = new int[grid.size()];
        int count = 0;
        for (int i = 0; i < grid.size(); i++) {
            if (!protectedCell[i]) {
                candidates[count++] = i;
            }
        }

        boolean[] mines = new boolean[grid.size()];
        for (int k = 0; k < config.mines(); k++) {
            int pick = k + rng.nextInt(count - k);
            int chosen = candidates[pick];
            candidates[pick] = candidates[k];
            candidates[k] = chosen;
            mines[chosen] = true;
        }
        return new MineField(grid, mines);
    }

    /**
     * Builds a field from text rows, {@code '*'} for a mine and {@code '.'} for a safe cell.
     * Used by tests and for hand-made puzzles.
     */
    public static MineField fromRows(String... rows) {
        if (rows.length == 0 || rows[0].isEmpty()) {
            throw new IllegalArgumentException("need at least one non-empty row");
        }
        int width = rows[0].length();
        Grid grid = new Grid(width, rows.length);
        boolean[] mines = new boolean[grid.size()];
        for (int y = 0; y < rows.length; y++) {
            if (rows[y].length() != width) {
                throw new IllegalArgumentException("row " + y + " has length " + rows[y].length() + ", expected " + width);
            }
            for (int x = 0; x < width; x++) {
                char ch = rows[y].charAt(x);
                if (ch != '*' && ch != '.') {
                    throw new IllegalArgumentException("unexpected '" + ch + "' at " + new Coord(x, y));
                }
                mines[grid.index(x, y)] = ch == '*';
            }
        }
        return new MineField(grid, mines);
    }

    public Grid grid() {
        return grid;
    }

    public int mineCount() {
        return mineCount;
    }

    public boolean isMine(int index) {
        return mines[index];
    }

    public boolean isMine(Coord c) {
        return mines[grid.index(c)];
    }

    /** Number of mines in the 8-neighbourhood of a cell. */
    public int adjacentMines(int index) {
        return adjacent[index];
    }
}
