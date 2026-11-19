package gamedev.minesweeper.core;

public enum GameStatus {
    /** No cell opened yet; mines are not placed until the first reveal. */
    READY,
    PLAYING,
    WON,
    LOST;

    public boolean isOver() {
        return this == WON || this == LOST;
    }
}
