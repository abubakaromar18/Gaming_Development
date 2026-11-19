package gamedev.minesweeper.core;

/**
 * A cell position on the board. {@code x} is the column and {@code y} the row,
 * both zero-based with (0, 0) at the top-left.
 */
public record Coord(int x, int y) {

    /** Human-friendly, one-based label used in hint explanations, e.g. "(col 4, row 2)". */
    public String label() {
        return "(col " + (x + 1) + ", row " + (y + 1) + ")";
    }
}
