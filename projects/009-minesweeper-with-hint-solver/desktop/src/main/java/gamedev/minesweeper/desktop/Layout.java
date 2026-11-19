package gamedev.minesweeper.desktop;

import gamedev.minesweeper.core.BoardConfig;
import gamedev.minesweeper.core.Coord;

/**
 * Pixel geometry of the window in a y-down coordinate system (origin top-left),
 * matching both the camera and mouse coordinates. Pure arithmetic, no libGDX types.
 */
final class Layout {

    static final int CELL = 24;
    static final int MARGIN = 12;
    static final int HEADER = 48;
    static final int FOOTER = 64;
    static final int FACE = 34;
    static final int DIGIT_W = 16;
    static final int DIGIT_H = 30;
    /** Narrowest window, so the header and hint text fit even on a 9-wide board. */
    private static final int MIN_BOARD_WIDTH = 9 * CELL + 120;

    private final BoardConfig config;
    private final int boardWidth;

    Layout(BoardConfig config) {
        this.config = config;
        this.boardWidth = config.width() * CELL;
    }

    int windowWidth() {
        return Math.max(boardWidth, MIN_BOARD_WIDTH) + 2 * MARGIN;
    }

    int windowHeight() {
        return MARGIN + HEADER + MARGIN + config.height() * CELL + MARGIN + FOOTER;
    }

    /** Left edge of the board; centred when the window is wider than the board. */
    int boardX() {
        return (windowWidth() - boardWidth) / 2;
    }

    int boardY() {
        return MARGIN + HEADER + MARGIN;
    }

    int cellX(int x) {
        return boardX() + x * CELL;
    }

    int cellY(int y) {
        return boardY() + y * CELL;
    }

    /** The cell under a point, or null if the point is not on the board. */
    Coord cellAt(float px, float py) {
        float bx = px - boardX();
        float by = py - boardY();
        if (bx < 0 || by < 0) {
            return null;
        }
        int x = (int) (bx / CELL);
        int y = (int) (by / CELL);
        return x < config.width() && y < config.height() ? new Coord(x, y) : null;
    }

    int faceX() {
        return (windowWidth() - FACE) / 2;
    }

    int faceY() {
        return MARGIN + (HEADER - FACE) / 2;
    }

    boolean onFace(float px, float py) {
        return px >= faceX() && px < faceX() + FACE && py >= faceY() && py < faceY() + FACE;
    }

    int counterY() {
        return MARGIN + (HEADER - DIGIT_H) / 2;
    }

    /** Left edge of the three-digit mine counter. */
    int mineCounterX() {
        return MARGIN + 8;
    }

    /** Left edge of the three-digit timer. */
    int timerX() {
        return windowWidth() - MARGIN - 8 - 3 * (DIGIT_W + 3);
    }

    int footerY() {
        return windowHeight() - FOOTER;
    }
}
