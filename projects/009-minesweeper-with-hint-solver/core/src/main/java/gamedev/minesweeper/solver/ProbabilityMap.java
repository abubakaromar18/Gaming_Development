package gamedev.minesweeper.solver;

import java.math.BigInteger;

/**
 * Exact mine probability for every unopened cell, derived by counting every full-board
 * mine layout that agrees with the visible numbers and the total mine count.
 */
public final class ProbabilityMap {

    private final double[] probability;
    private final BigInteger layouts;

    ProbabilityMap(double[] probability, BigInteger layouts) {
        this.probability = probability;
        this.layouts = layouts;
    }

    /** Chance in [0, 1] that the cell holds a mine, or NaN for an open cell. */
    public double mineProbability(int index) {
        return probability[index];
    }

    /** How many distinct mine layouts are consistent with what the player can see. */
    public BigInteger layouts() {
        return layouts;
    }
}
