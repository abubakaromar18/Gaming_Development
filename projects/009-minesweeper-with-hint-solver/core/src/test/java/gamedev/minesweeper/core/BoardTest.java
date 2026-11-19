package gamedev.minesweeper.core;

import org.junit.jupiter.api.Test;

import java.util.SplittableRandom;

import static org.junit.jupiter.api.Assertions.assertArrayEquals;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

/** BoardConfig, Grid and MineField. */
class BoardTest {

    @Test
    void presetsMatchTheClassicDifficulties() {
        assertEquals(81, BoardConfig.BEGINNER.cells());
        assertEquals(40, BoardConfig.INTERMEDIATE.mines());
        assertEquals(30, BoardConfig.EXPERT.width());
        assertEquals(99, BoardConfig.EXPERT.mines());
    }

    @Test
    void configRejectsUnplayableBoards() {
        assertThrows(IllegalArgumentException.class, () -> new BoardConfig(0, 5, 1));
        assertThrows(IllegalArgumentException.class, () -> new BoardConfig(5, 101, 1));
        assertThrows(IllegalArgumentException.class, () -> new BoardConfig(3, 3, 0));
        assertThrows(IllegalArgumentException.class, () -> new BoardConfig(3, 3, 9)); // no safe first click
        assertEquals(8, new BoardConfig(3, 3, 8).mines());
    }

    @Test
    void neighborsRespectEdgesAndCorners() {
        Grid g = new Grid(4, 3);
        assertEquals(3, g.neighbors(g.index(0, 0)).length);
        assertEquals(5, g.neighbors(g.index(1, 0)).length);
        assertEquals(8, g.neighbors(g.index(1, 1)).length);
        assertArrayEquals(new int[]{1, 4, 5}, g.neighbors(0));
        assertEquals(new Coord(3, 2), g.coord(g.index(3, 2)));
        assertEquals(2, g.distance(g.index(0, 0), g.index(2, 1)));
        assertThrows(IllegalArgumentException.class, () -> g.index(new Coord(4, 0)));
    }

    @Test
    void fromRowsCountsMinesAndNumbers() {
        MineField f = MineField.fromRows(
                "*..",
                "..*",
                "...");
        assertEquals(2, f.mineCount());
        assertTrue(f.isMine(new Coord(0, 0)));
        Grid g = f.grid();
        assertEquals(2, f.adjacentMines(g.index(1, 1)));
        assertEquals(2, f.adjacentMines(g.index(1, 0)), "touches both mines");
        assertEquals(0, f.adjacentMines(g.index(0, 2)));
    }

    @Test
    void fromRowsRejectsBadInput() {
        assertThrows(IllegalArgumentException.class, () -> MineField.fromRows("*.", "."));
        assertThrows(IllegalArgumentException.class, () -> MineField.fromRows("*x"));
        assertThrows(IllegalArgumentException.class, MineField::fromRows);
    }

    @Test
    void generationKeepsTheWholeOpeningBlockClear() {
        BoardConfig config = BoardConfig.EXPERT;
        Coord click = new Coord(10, 7);
        for (long seed = 0; seed < 200; seed++) {
            MineField f = MineField.generate(config, click, new SplittableRandom(seed));
            assertEquals(config.mines(), f.mineCount());
            Grid g = f.grid();
            int c = g.index(click);
            assertFalse(f.isMine(c));
            assertEquals(0, f.adjacentMines(c), "seed " + seed);
        }
    }

    @Test
    void denseBoardsProtectOnlyTheClickedCell() {
        // 3x3 with 8 mines: there is no room for a clear block, but the click is still safe.
        MineField f = MineField.generate(new BoardConfig(3, 3, 8), new Coord(1, 1), new SplittableRandom(5));
        assertFalse(f.isMine(new Coord(1, 1)));
        assertEquals(8, f.mineCount());
    }

    @Test
    void generationIsDeterministicPerSeedAndCoversEveryAllowedCell() {
        BoardConfig config = new BoardConfig(6, 6, 5);
        Coord click = new Coord(0, 0);
        MineField a = MineField.generate(config, click, new SplittableRandom(77));
        MineField b = MineField.generate(config, click, new SplittableRandom(77));
        boolean[] everMine = new boolean[36];
        for (int i = 0; i < 36; i++) {
            assertEquals(a.isMine(i), b.isMine(i));
        }
        for (long seed = 0; seed < 300; seed++) {
            MineField f = MineField.generate(config, click, new SplittableRandom(seed));
            for (int i = 0; i < 36; i++) {
                everMine[i] |= f.isMine(i);
            }
        }
        Grid g = a.grid();
        for (int i = 0; i < 36; i++) {
            boolean protectedCell = g.distance(i, g.index(click)) <= 1;
            assertEquals(!protectedCell, everMine[i], "cell " + g.coord(i));
        }
    }
}
