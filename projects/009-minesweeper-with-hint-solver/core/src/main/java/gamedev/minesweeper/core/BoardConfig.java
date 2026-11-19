package gamedev.minesweeper.core;

/**
 * Board dimensions and mine count. The constructor validates the combination so
 * that a {@link Game} can never be created in an unplayable configuration.
 */
public record BoardConfig(int width, int height, int mines) {

    /** Largest supported side length; keeps adjacency tables and the UI sane. */
    public static final int MAX_SIDE = 100;

    public static final BoardConfig BEGINNER = new BoardConfig(9, 9, 10);
    public static final BoardConfig INTERMEDIATE = new BoardConfig(16, 16, 40);
    public static final BoardConfig EXPERT = new BoardConfig(30, 16, 99);

    public BoardConfig {
        if (width < 1 || height < 1 || width > MAX_SIDE || height > MAX_SIDE) {
            throw new IllegalArgumentException(
                    "board must be between 1x1 and " + MAX_SIDE + "x" + MAX_SIDE + ", got " + width + "x" + height);
        }
        if (mines < 1 || mines > width * height - 1) {
            throw new IllegalArgumentException(
                    "mines must be in 1.." + (width * height - 1) + " so the first click can be safe, got " + mines);
        }
    }

    public int cells() {
        return width * height;
    }
}
