package gamedev.minesweeper.core;

/**
 * Everything a player can legitimately see: which cells are open, what numbers they
 * show, where flags are, and the total mine count. The hint solver depends only on
 * this interface, so it is impossible for it to "cheat" by peeking at the mine layout.
 */
public interface PlayerView {

    Grid grid();

    GameStatus status();

    /** Total mines on the board (shown on the counter in every Minesweeper). */
    int totalMines();

    boolean isRevealed(int index);

    boolean isFlagged(int index);

    /** The number on an open cell. Throws {@link IllegalStateException} if the cell is not open. */
    int number(int index);
}
