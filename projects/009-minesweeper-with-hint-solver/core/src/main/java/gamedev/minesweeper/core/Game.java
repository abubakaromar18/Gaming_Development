package gamedev.minesweeper.core;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.random.RandomGenerator;

/**
 * One game of Minesweeper: the rules, the player's knowledge, and the clock.
 *
 * <p>Mines are placed lazily on the first reveal ({@link MineField#generate}), which is
 * how first-click safety works without ever moving a mine after the fact. Time only
 * advances through {@link #tick(double)}, so the game never reads a wall clock.
 */
public final class Game implements PlayerView {

    private enum CellState { HIDDEN, FLAGGED, REVEALED }

    private final BoardConfig config;
    private final Grid grid;
    private final RandomGenerator rng;
    private final CellState[] cells;

    private MineField field;
    private GameStatus status = GameStatus.READY;
    private int revealedCount;
    private int flagCount;
    private int explodedIndex = -1;
    private double elapsedSeconds;

    /** A new game whose mines are placed on the first reveal using {@code rng}. */
    public Game(BoardConfig config, RandomGenerator rng) {
        this.config = config;
        this.grid = new Grid(config.width(), config.height());
        this.rng = rng;
        this.cells = new CellState[grid.size()];
        Arrays.fill(cells, CellState.HIDDEN);
    }

    /**
     * A game on a fixed layout. There is no first-click protection, which is exactly
     * what tests and hand-made puzzles need.
     */
    public static Game withField(MineField field) {
        Grid g = field.grid();
        Game game = new Game(new BoardConfig(g.width(), g.height(), field.mineCount()), null);
        game.field = field;
        return game;
    }

    // ---------------------------------------------------------------- actions

    /** Opens a hidden cell. Flagged and already-open cells are ignored. */
    public RevealResult reveal(Coord c) {
        int start = grid.index(c);
        if (status.isOver() || cells[start] != CellState.HIDDEN) {
            return RevealResult.NOTHING;
        }
        if (field == null) {
            field = MineField.generate(config, c, rng);
        }
        status = GameStatus.PLAYING;

        if (field.isMine(start)) {
            cells[start] = CellState.REVEALED;
            explodedIndex = start;
            status = GameStatus.LOST;
            return new RevealResult(List.of(c), true);
        }

        List<Coord> opened = floodOpen(start);
        if (revealedCount == grid.size() - field.mineCount()) {
            win();
        }
        return new RevealResult(opened, false);
    }

    /**
     * "Chording": clicking an open number whose flag count already matches it opens all
     * of its other hidden neighbours at once. If a flag is wrong, this can hit a mine.
     */
    public RevealResult chord(Coord c) {
        int center = grid.index(c);
        if (status != GameStatus.PLAYING || cells[center] != CellState.REVEALED) {
            return RevealResult.NOTHING;
        }
        int[] around = grid.neighbors(center);
        int flags = 0;
        for (int n : around) {
            if (cells[n] == CellState.FLAGGED) {
                flags++;
            }
        }
        if (flags != field.adjacentMines(center)) {
            return RevealResult.NOTHING;
        }

        List<Coord> opened = new ArrayList<>();
        boolean detonated = false;
        for (int n : around) {
            if (cells[n] == CellState.HIDDEN) {
                RevealResult r = reveal(grid.coord(n));
                opened.addAll(r.opened());
                detonated |= r.detonated();
            }
        }
        return new RevealResult(opened, detonated);
    }

    /** Toggles a flag on a hidden cell. Returns true if anything changed. */
    public boolean toggleFlag(Coord c) {
        int i = grid.index(c);
        if (status != GameStatus.PLAYING || cells[i] == CellState.REVEALED) {
            return false;
        }
        if (cells[i] == CellState.FLAGGED) {
            cells[i] = CellState.HIDDEN;
            flagCount--;
        } else {
            cells[i] = CellState.FLAGGED;
            flagCount++;
        }
        return true;
    }

    /** Advances the game clock. The clock only runs between the first click and game over. */
    public void tick(double dtSeconds) {
        if (!(dtSeconds >= 0) || Double.isInfinite(dtSeconds)) {
            throw new IllegalArgumentException("dt must be a finite, non-negative number of seconds, got " + dtSeconds);
        }
        if (status == GameStatus.PLAYING) {
            elapsedSeconds += dtSeconds;
        }
    }

    // ---------------------------------------------------------------- queries

    public BoardConfig config() {
        return config;
    }

    public double elapsedSeconds() {
        return elapsedSeconds;
    }

    /** Mines minus flags: the classic counter. Goes negative if the player over-flags. */
    public int remainingMines() {
        return config.mines() - flagCount;
    }

    /** What to draw at {@code c}. Mine positions are only revealed once the game is over. */
    public CellView view(Coord c) {
        int i = grid.index(c);
        CellState state = cells[i];
        if (status == GameStatus.LOST) {
            if (i == explodedIndex) {
                return CellView.EXPLODED;
            }
            boolean mine = field.isMine(i);
            if (state == CellState.FLAGGED) {
                return mine ? CellView.FLAGGED : CellView.WRONG_FLAG;
            }
            if (mine) {
                return CellView.MINE;
            }
        }
        return switch (state) {
            case HIDDEN -> CellView.HIDDEN;
            case FLAGGED -> CellView.FLAGGED;
            case REVEALED -> CellView.open(field.adjacentMines(i));
        };
    }

    @Override
    public Grid grid() {
        return grid;
    }

    @Override
    public GameStatus status() {
        return status;
    }

    @Override
    public int totalMines() {
        return config.mines();
    }

    @Override
    public boolean isRevealed(int index) {
        return cells[index] == CellState.REVEALED;
    }

    @Override
    public boolean isFlagged(int index) {
        return cells[index] == CellState.FLAGGED;
    }

    @Override
    public int number(int index) {
        if (cells[index] != CellState.REVEALED) {
            throw new IllegalStateException(grid.coord(index) + " is not open");
        }
        return field.adjacentMines(index);
    }

    // ---------------------------------------------------------------- internals

    /**
     * Breadth-first flood fill. Opening a 0 opens all its neighbours; any neighbour that
     * is also a 0 continues the fill. An explicit queue (not recursion) keeps a 100x100
     * empty board from overflowing the stack, and each cell is enqueued at most once
     * because it is marked open before it is queued.
     */
    private List<Coord> floodOpen(int start) {
        List<Coord> opened = new ArrayList<>();
        int[] queue = new int[grid.size()];
        int head = 0;
        int tail = 0;
        open(start, opened);
        queue[tail++] = start;
        while (head < tail) {
            int cell = queue[head++];
            if (field.adjacentMines(cell) != 0) {
                continue;
            }
            for (int n : grid.neighbors(cell)) {
                if (cells[n] == CellState.HIDDEN) {
                    open(n, opened);
                    queue[tail++] = n;
                }
            }
        }
        return opened;
    }

    private void open(int index, List<Coord> opened) {
        cells[index] = CellState.REVEALED;
        revealedCount++;
        opened.add(grid.coord(index));
    }

    /** On a win every remaining mine is flagged, as in the original game. */
    private void win() {
        status = GameStatus.WON;
        for (int i = 0; i < cells.length; i++) {
            if (cells[i] == CellState.HIDDEN) {
                cells[i] = CellState.FLAGGED;
            }
        }
        flagCount = config.mines();
    }
}
