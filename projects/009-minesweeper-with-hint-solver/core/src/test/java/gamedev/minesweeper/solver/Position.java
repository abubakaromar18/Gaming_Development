package gamedev.minesweeper.solver;

import gamedev.minesweeper.core.GameStatus;
import gamedev.minesweeper.core.Grid;
import gamedev.minesweeper.core.MineField;
import gamedev.minesweeper.core.PlayerView;

/**
 * A hand-drawn mid-game position for solver tests. Because the solver depends only on
 * {@link PlayerView}, a position can be described directly instead of played into.
 *
 * <pre>
 *   o  open safe cell (its number is computed from the mines)
 *   .  hidden safe cell
 *   *  hidden mine
 *   F  flagged mine
 *   f  flagged safe cell (a wrong flag)
 * </pre>
 *
 * Numbers are derived from the mines, so every position is consistent by construction,
 * and {@link #isMine(int)} gives tests the ground truth to check deductions against.
 */
final class Position implements PlayerView {

    private final Grid grid;
    private final MineField field;
    private final boolean[] open;
    private final boolean[] flagged;

    private Position(String[] rows) {
        String[] mineRows = new String[rows.length];
        for (int y = 0; y < rows.length; y++) {
            StringBuilder sb = new StringBuilder();
            for (char ch : rows[y].toCharArray()) {
                switch (ch) {
                    case '*', 'F' -> sb.append('*');
                    case 'o', '.', 'f' -> sb.append('.');
                    default -> throw new IllegalArgumentException("bad cell '" + ch + "'");
                }
            }
            mineRows[y] = sb.toString();
        }
        this.field = MineField.fromRows(mineRows);
        this.grid = field.grid();
        this.open = new boolean[grid.size()];
        this.flagged = new boolean[grid.size()];
        for (int y = 0; y < rows.length; y++) {
            for (int x = 0; x < rows[y].length(); x++) {
                char ch = rows[y].charAt(x);
                open[grid.index(x, y)] = ch == 'o';
                flagged[grid.index(x, y)] = ch == 'F' || ch == 'f';
            }
        }
    }

    static Position of(String... rows) {
        return new Position(rows);
    }

    /** A position built from a real layout with an arbitrary set of safe cells opened. */
    static Position of(MineField field, boolean[] opened) {
        StringBuilder all = new StringBuilder();
        Grid g = field.grid();
        String[] rows = new String[g.height()];
        for (int y = 0; y < g.height(); y++) {
            all.setLength(0);
            for (int x = 0; x < g.width(); x++) {
                int i = g.index(x, y);
                all.append(field.isMine(i) ? '*' : opened[i] ? 'o' : '.');
            }
            rows[y] = all.toString();
        }
        return new Position(rows);
    }

    boolean isMine(int index) {
        return field.isMine(index);
    }

    @Override
    public Grid grid() {
        return grid;
    }

    @Override
    public GameStatus status() {
        return GameStatus.PLAYING;
    }

    @Override
    public int totalMines() {
        return field.mineCount();
    }

    @Override
    public boolean isRevealed(int index) {
        return open[index];
    }

    @Override
    public boolean isFlagged(int index) {
        return flagged[index];
    }

    @Override
    public int number(int index) {
        if (!open[index]) {
            throw new IllegalStateException("cell " + index + " is not open");
        }
        return field.adjacentMines(index);
    }
}
