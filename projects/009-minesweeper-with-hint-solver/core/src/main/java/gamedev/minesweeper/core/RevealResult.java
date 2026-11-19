package gamedev.minesweeper.core;

import java.util.List;

/**
 * Outcome of a reveal or chord: the cells that opened (in flood-fill order, which a
 * renderer can use to animate the ripple) and whether a mine went off.
 */
public record RevealResult(List<Coord> opened, boolean detonated) {

    public static final RevealResult NOTHING = new RevealResult(List.of(), false);

    public RevealResult {
        opened = List.copyOf(opened);
    }

    public boolean changedBoard() {
        return !opened.isEmpty();
    }
}
