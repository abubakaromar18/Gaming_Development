package gamedev.minesweeper.core;

import java.util.Arrays;

/**
 * Board geometry. Cells are addressed by a flat row-major index for speed; the
 * 8-neighbourhood of every cell is precomputed once because flood fill, chording
 * and the solver all ask for it constantly.
 */
public final class Grid {

    private final int width;
    private final int height;
    private final int[][] neighbors;

    public Grid(int width, int height) {
        if (width < 1 || height < 1) {
            throw new IllegalArgumentException("grid must be at least 1x1, got " + width + "x" + height);
        }
        this.width = width;
        this.height = height;
        this.neighbors = new int[width * height][];
        int[] scratch = new int[8];
        for (int y = 0; y < height; y++) {
            for (int x = 0; x < width; x++) {
                int n = 0;
                for (int dy = -1; dy <= 1; dy++) {
                    for (int dx = -1; dx <= 1; dx++) {
                        if ((dx != 0 || dy != 0) && contains(x + dx, y + dy)) {
                            scratch[n++] = index(x + dx, y + dy);
                        }
                    }
                }
                neighbors[index(x, y)] = Arrays.copyOf(scratch, n);
            }
        }
    }

    public int width() {
        return width;
    }

    public int height() {
        return height;
    }

    public int size() {
        return width * height;
    }

    public boolean contains(int x, int y) {
        return x >= 0 && y >= 0 && x < width && y < height;
    }

    public boolean contains(Coord c) {
        return contains(c.x(), c.y());
    }

    public int index(int x, int y) {
        return y * width + x;
    }

    /** Index of {@code c}; throws if it is off the board. */
    public int index(Coord c) {
        if (!contains(c)) {
            throw new IllegalArgumentException(c + " is outside the " + width + "x" + height + " board");
        }
        return index(c.x(), c.y());
    }

    public Coord coord(int index) {
        return new Coord(index % width, index / width);
    }

    /** The up-to-eight neighbours of a cell. The returned array is shared: do not modify it. */
    public int[] neighbors(int index) {
        return neighbors[index];
    }

    /** Chebyshev distance between two cells. */
    public int distance(int a, int b) {
        return Math.max(Math.abs(a % width - b % width), Math.abs(a / width - b / width));
    }
}
