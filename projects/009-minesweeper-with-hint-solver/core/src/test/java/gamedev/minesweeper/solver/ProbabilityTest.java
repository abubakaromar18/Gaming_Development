package gamedev.minesweeper.solver;

import gamedev.minesweeper.core.Grid;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.List;
import java.util.SplittableRandom;

import static org.junit.jupiter.api.Assertions.assertArrayEquals;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertInstanceOf;

/**
 * Checks the component enumeration against the dumbest possible oracle: try every
 * mine layout over the hidden cells and keep the ones that match all numbers.
 */
class ProbabilityTest {

    private static final int MAX_BRUTE_FORCE_CELLS = 18;

    @Test
    void probabilitiesMatchBruteForceOnRandomPositions() {
        SplittableRandom rng = new SplittableRandom(2024);
        HintEngine engine = new HintEngine();
        int checked = 0;
        while (checked < 120) {
            Position p = RandomPositions.next(rng, 6, 5, 6);
            List<Integer> hidden = hiddenCells(p);
            if (hidden.size() > MAX_BRUTE_FORCE_CELLS || hidden.isEmpty()) {
                continue;
            }
            Oracle truth = bruteForce(p, hidden);
            ProbabilityMap odds = engine.analyze(p).probabilities().orElseThrow();

            assertEquals(BigInteger.valueOf(truth.layouts), odds.layouts(), "layout count");
            for (int k = 0; k < hidden.size(); k++) {
                double expected = (double) truth.mineCounts[k] / truth.layouts;
                assertEquals(expected, odds.mineProbability(hidden.get(k)), 1e-12, "cell " + hidden.get(k));
            }
            Hint h = engine.hint(p).orElseThrow();
            if (h instanceof Hint.Guess g) {
                double min = Double.POSITIVE_INFINITY;
                for (int k = 0; k < hidden.size(); k++) {
                    min = Math.min(min, (double) truth.mineCounts[k] / truth.layouts);
                }
                assertEquals(min, g.mineProbability(), 1e-12, "guess should take the safest cell");
            }
            checked++;
        }
    }

    @Test
    void binomialRowAndConvolution() {
        assertArrayEquals(
                new BigInteger[]{BigInteger.ONE, BigInteger.valueOf(4), BigInteger.valueOf(6),
                        BigInteger.valueOf(4), BigInteger.ONE},
                Enumerator.binomialRow(4));
        // (1 + 2x)(3 + x) = 3 + 7x + 2x^2
        BigInteger[] product = Enumerator.convolve(
                new BigInteger[]{BigInteger.ONE, BigInteger.TWO},
                new BigInteger[]{BigInteger.valueOf(3), BigInteger.ONE});
        assertArrayEquals(new BigInteger[]{BigInteger.valueOf(3), BigInteger.valueOf(7), BigInteger.TWO}, product);
        // C(480, 240) is far beyond a long; BigInteger keeps it exact.
        assertEquals(144, Enumerator.binomialRow(480)[240].toString().length());
    }

    @Test
    void expertSizedInteriorStaysExact() {
        // One open cell on an expert-sized board with 79 mines: the interior term is C(475, 78).
        StringBuilder top = new StringBuilder("o");
        top.append(".".repeat(29));
        String[] rows = new String[16];
        rows[0] = top.toString();
        for (int y = 1; y < 16; y++) {
            rows[y] = y <= 4 ? "*".repeat(20) + ".".repeat(10) : ".".repeat(30);
        }
        rows[1] = "." + rows[1].substring(1);
        Position p = Position.of(rows);
        HintEngine engine = new HintEngine();
        ProbabilityMap odds = engine.analyze(p).probabilities().orElseThrow();
        double sum = 0;
        for (int i = 0; i < p.grid().size(); i++) {
            if (!p.isRevealed(i)) {
                sum += odds.mineProbability(i);
            }
        }
        assertEquals(p.totalMines(), sum, 1e-6, "probabilities must add up to the mine count");
        assertInstanceOf(Hint.Guess.class, engine.hint(p).orElseThrow());
    }

    private static List<Integer> hiddenCells(Position p) {
        List<Integer> hidden = new ArrayList<>();
        for (int i = 0; i < p.grid().size(); i++) {
            if (!p.isRevealed(i)) {
                hidden.add(i);
            }
        }
        return hidden;
    }

    private record Oracle(long layouts, long[] mineCounts) { }

    private static Oracle bruteForce(Position p, List<Integer> hidden) {
        Grid g = p.grid();
        boolean[] mine = new boolean[g.size()];
        long layouts = 0;
        long[] counts = new long[hidden.size()];
        for (int mask = 0; mask < (1 << hidden.size()); mask++) {
            if (Integer.bitCount(mask) != p.totalMines()) {
                continue;
            }
            for (int k = 0; k < hidden.size(); k++) {
                mine[hidden.get(k)] = (mask & (1 << k)) != 0;
            }
            if (consistent(p, mine)) {
                layouts++;
                for (int k = 0; k < hidden.size(); k++) {
                    if (mine[hidden.get(k)]) {
                        counts[k]++;
                    }
                }
            }
        }
        return new Oracle(layouts, counts);
    }

    private static boolean consistent(Position p, boolean[] mine) {
        Grid g = p.grid();
        for (int i = 0; i < g.size(); i++) {
            if (!p.isRevealed(i)) {
                continue;
            }
            int n = 0;
            for (int nb : g.neighbors(i)) {
                if (mine[nb]) {
                    n++;
                }
            }
            if (n != p.number(i)) {
                return false;
            }
        }
        return true;
    }
}
