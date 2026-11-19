package gamedev.minesweeper.solver;

import java.util.Arrays;

/**
 * One open number, reduced by what is already known: {@code cells} are its still
 * undecided neighbours (sorted) and {@code mines} is how many of them are mines.
 *
 * @param source  the open cell showing the number
 * @param number  the number as displayed
 */
record Constraint(int source, int number, int[] cells, int mines) {

    /** Number of entries of {@code cells} that also appear in {@code other.cells}. Both are sorted. */
    int sharedWith(Constraint other) {
        int i = 0;
        int j = 0;
        int shared = 0;
        while (i < cells.length && j < other.cells.length) {
            if (cells[i] == other.cells[j]) {
                shared++;
                i++;
                j++;
            } else if (cells[i] < other.cells[j]) {
                i++;
            } else {
                j++;
            }
        }
        return shared;
    }

    /** Entries of {@code cells} that are not in {@code other.cells}. */
    int[] minus(Constraint other) {
        int[] out = new int[cells.length];
        int n = 0;
        int j = 0;
        for (int c : cells) {
            while (j < other.cells.length && other.cells[j] < c) {
                j++;
            }
            if (j >= other.cells.length || other.cells[j] != c) {
                out[n++] = c;
            }
        }
        return Arrays.copyOf(out, n);
    }
}
