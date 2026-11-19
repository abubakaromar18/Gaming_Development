package gamedev.minesweeper.solver;

import gamedev.minesweeper.core.BoardConfig;
import gamedev.minesweeper.core.Coord;
import gamedev.minesweeper.core.Game;
import gamedev.minesweeper.core.MineField;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.SplittableRandom;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertInstanceOf;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

class HintEngineTest {

    private final HintEngine engine = new HintEngine();

    private Hint hint(Position p) {
        return engine.hint(p).orElseThrow();
    }

    @Test
    void beforeTheFirstClickAnyCellIsSafe() {
        Game game = new Game(BoardConfig.BEGINNER, new SplittableRandom(1));
        assertInstanceOf(Hint.FirstClick.class, engine.hint(game).orElseThrow());
    }

    @Test
    void noHintOnceTheGameIsOver() {
        Game game = Game.withField(MineField.fromRows("*."));
        game.reveal(new Coord(0, 0));
        assertTrue(engine.hint(game).isEmpty());
    }

    @Test
    void singleNumberWithAsManyHiddenNeighborsAsItsValueMarksThemAllMines() {
        Hint.Deduction d = assertInstanceOf(Hint.Deduction.class, hint(Position.of(
                "o*",
                "oo")));
        assertEquals(new Coord(1, 0), d.cell());
        assertTrue(d.mine());
        assertEquals(Rule.SINGLE, d.rule());
        assertEquals(List.of(new Coord(0, 0)), d.evidence());
    }

    @Test
    void numberSatisfiedByProvenMinesMakesItsOtherNeighborsSafe() {
        // The 1 at (0,0) proves (1,0) is a mine; the 1 at (1,1) is then satisfied.
        Hint.Deduction d = assertInstanceOf(Hint.Deduction.class, hint(Position.of(
                "o*.",
                "ooo")));
        assertEquals(new Coord(2, 0), d.cell());
        assertFalse(d.mine());
        assertEquals(Rule.SINGLE, d.rule());
        assertEquals(List.of(new Coord(1, 1)), d.evidence());
    }

    @Test
    void subsetOfAnotherNumberProvesTheDifferenceSafe() {
        // Both 1s need exactly one mine; the left 1's cells are a subset of the right 1's,
        // so the right 1's extra cells in column 2 hold no mine.
        Hint.Deduction d = assertInstanceOf(Hint.Deduction.class, hint(Position.of(
                "*...",
                "oo..")));
        assertEquals(new Coord(2, 0), d.cell());
        assertFalse(d.mine());
        assertEquals(Rule.PAIR, d.rule());
        assertEquals(List.of(new Coord(0, 1), new Coord(1, 1)), d.evidence());
    }

    @Test
    void oneTwoPatternProvesAMineThenAChainOfSafeCells() {
        Position p = Position.of(
                "*.*",
                "ooo");
        Analysis a = engine.analyze(p);
        Hint.Deduction first = a.deductions().get(0);
        assertEquals(new Coord(2, 0), first.cell());
        assertTrue(first.mine());
        assertEquals(Rule.PAIR, first.rule());

        // The hint prefers the safe cell that the mine unlocks.
        Hint.Deduction h = assertInstanceOf(Hint.Deduction.class, hint(p));
        assertEquals(new Coord(1, 0), h.cell());
        assertFalse(h.mine());
    }

    @Test
    void mineTotalProvesInteriorCellsSafe() {
        // The only mine must be next to the 1s, so the right-hand cells are safe.
        Position p = Position.of(
                "o*..",
                "o...");
        Hint.Deduction d = assertInstanceOf(Hint.Deduction.class, hint(p));
        assertEquals(Rule.ENUMERATION, d.rule());
        assertFalse(d.mine());
        assertEquals(new Coord(2, 0), d.cell());
        assertTrue(d.evidence().isEmpty());

        ProbabilityMap odds = engine.analyze(p).probabilities().orElseThrow();
        assertEquals(0.5, odds.mineProbability(p.grid().index(1, 0)), 1e-12);
        assertEquals(0.0, odds.mineProbability(p.grid().index(3, 1)), 1e-12);
        assertTrue(Double.isNaN(odds.mineProbability(p.grid().index(0, 0))));
        assertEquals(2, odds.layouts().intValueExact());
    }

    @Test
    void flagOnAProvablySafeCellIsReportedAsWrong() {
        Hint.WrongFlag w = assertInstanceOf(Hint.WrongFlag.class, hint(Position.of(
                "o*f",
                "ooo")));
        assertEquals(new Coord(2, 0), w.cell());
        assertTrue(w.explanation().startsWith("The flag at (col 3, row 1) is wrong."));
    }

    @Test
    void correctFlagsAreNotSuggestedAgain() {
        // (1,0) is already flagged, so the next useful step is the safe cell.
        Hint.Deduction d = assertInstanceOf(Hint.Deduction.class, hint(Position.of(
                "oF.",
                "ooo")));
        assertEquals(new Coord(2, 0), d.cell());
        assertFalse(d.mine());
    }

    @Test
    void trueFiftyFiftyGivesAnExactGuess() {
        Hint.Guess g = assertInstanceOf(Hint.Guess.class, hint(Position.of(
                "*.",
                "oo")));
        assertEquals(new Coord(0, 0), g.cell());
        assertEquals(0.5, g.mineProbability(), 1e-12);
        assertTrue(g.exact());
    }

    @Test
    void overBudgetEnumerationFallsBackToAnEstimate() {
        Hint.Guess g = assertInstanceOf(Hint.Guess.class, new HintEngine(1).hint(Position.of(
                "*.",
                "oo")).orElseThrow());
        assertFalse(g.exact());
        assertEquals(0.5, g.mineProbability(), 1e-12);
        assertTrue(new HintEngine(1).analyze(Position.of("*.", "oo")).probabilities().isEmpty());
    }

    @Test
    void rejectsNonPositiveBudget() {
        assertThrows(IllegalArgumentException.class, () -> new HintEngine(0));
    }

    @Test
    void everyDeductionOnRandomPositionsIsTrue() {
        SplittableRandom rng = new SplittableRandom(99);
        for (int trial = 0; trial < 300; trial++) {
            Position p = RandomPositions.next(rng, 7, 6, 8);
            for (Hint.Deduction d : engine.analyze(p).deductions()) {
                assertEquals(p.isMine(p.grid().index(d.cell())), d.mine(), "trial " + trial + ": " + d.explanation());
            }
        }
    }
}
