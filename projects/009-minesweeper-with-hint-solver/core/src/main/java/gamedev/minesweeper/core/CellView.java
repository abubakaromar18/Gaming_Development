package gamedev.minesweeper.core;

/**
 * What the renderer should draw for one cell. Mines are only exposed once the game
 * is over, so a renderer built on this cannot leak the layout mid-game.
 *
 * @param adjacent the number to show when {@code kind == OPEN}, otherwise 0
 */
public record CellView(Kind kind, int adjacent) {

    public enum Kind {
        HIDDEN,
        FLAGGED,
        OPEN,
        /** An unflagged mine revealed at game over. */
        MINE,
        /** The mine the player clicked. */
        EXPLODED,
        /** A flag that turned out to be on a safe cell (shown at game over). */
        WRONG_FLAG
    }

    static final CellView HIDDEN = new CellView(Kind.HIDDEN, 0);
    static final CellView FLAGGED = new CellView(Kind.FLAGGED, 0);
    static final CellView MINE = new CellView(Kind.MINE, 0);
    static final CellView EXPLODED = new CellView(Kind.EXPLODED, 0);
    static final CellView WRONG_FLAG = new CellView(Kind.WRONG_FLAG, 0);

    static CellView open(int adjacent) {
        return new CellView(Kind.OPEN, adjacent);
    }
}
