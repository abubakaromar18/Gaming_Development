package gamedev.minesweeper.solver;

import gamedev.minesweeper.core.Coord;

import java.util.List;

/**
 * Advice for the player. A sealed hierarchy lets the UI handle every case with an
 * exhaustive {@code switch}, and the compiler flags any case it forgets.
 */
public sealed interface Hint {

    /** A sentence a player can read. */
    String explanation();

    /** Before the first click: every cell is safe. */
    record FirstClick() implements Hint {
        @Override
        public String explanation() {
            return "Click anywhere: the first reveal never hits a mine.";
        }
    }

    /**
     * A cell whose content is proven from visible information.
     *
     * @param evidence the open numbers the proof relies on, for highlighting
     */
    record Deduction(Coord cell, boolean mine, Rule rule, List<Coord> evidence, String explanation) implements Hint {
        public Deduction {
            evidence = List.copyOf(evidence);
        }
    }

    /** The player flagged a cell that is provably safe. */
    record WrongFlag(Coord cell, Deduction proof) implements Hint {
        @Override
        public String explanation() {
            return "The flag at " + cell.label() + " is wrong. " + proof.explanation();
        }
    }

    /**
     * Nothing is provable, so this is the safest cell to try.
     *
     * @param exact true if the probability comes from full enumeration, false if it is a local estimate
     */
    record Guess(Coord cell, double mineProbability, boolean exact) implements Hint {
        @Override
        public String explanation() {
            long pct = Math.round(mineProbability * 100);
            return "No safe cell can be proven, so this needs a guess. Best odds: " + cell.label()
                    + (exact ? " has a " : " has roughly a ") + pct + "% chance of hiding a mine.";
        }
    }
}
