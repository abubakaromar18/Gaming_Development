package gamedev.minesweeper.core;

import org.junit.jupiter.api.Test;

import java.util.SplittableRandom;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertSame;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

class GameTest {

    private static Coord at(int x, int y) {
        return new Coord(x, y);
    }

    @Test
    void firstRevealIsAlwaysSafeAndOpensARegion() {
        for (long seed = 0; seed < 300; seed++) {
            Game game = new Game(BoardConfig.EXPERT, new SplittableRandom(seed));
            assertEquals(GameStatus.READY, game.status());
            RevealResult r = game.reveal(at(15, 8));
            assertFalse(r.detonated(), "seed " + seed);
            assertTrue(r.opened().size() >= 9, "the 3x3 around the click is clear, so at least 9 cells open");
            assertEquals(GameStatus.PLAYING, game.status());
        }
    }

    @Test
    void floodFillStopsAtNumbersAndSkipsFlags() {
        Game game = Game.withField(MineField.fromRows(
                ".....",
                ".....",
                "...**",
                "....."));
        game.reveal(at(4, 0)); // a 0, so the fill spreads
        assertEquals(CellView.Kind.OPEN, game.view(at(4, 0)).kind());
        assertEquals(CellView.Kind.OPEN, game.view(at(0, 3)).kind(), "the fill wraps around the mines");
        assertEquals(2, game.view(at(3, 1)).adjacent());
        assertEquals(CellView.Kind.HIDDEN, game.view(at(3, 2)).kind());

        Game flagged = Game.withField(MineField.fromRows(
                "....*",
                "....."));
        flagged.reveal(at(3, 0)); // a 1: opens only itself and starts the game
        assertTrue(flagged.toggleFlag(at(0, 0)));
        flagged.reveal(at(0, 1));
        assertEquals(CellView.Kind.FLAGGED, flagged.view(at(0, 0)).kind(), "flood fill never opens a flag");
        assertEquals(CellView.Kind.OPEN, flagged.view(at(1, 0)).kind());
    }

    @Test
    void openingEveryHugeEmptyBoardDoesNotOverflowTheStack() {
        String[] rows = new String[100];
        for (int y = 0; y < 100; y++) {
            rows[y] = y == 99 ? ".".repeat(99) + "*" : ".".repeat(100);
        }
        Game game = Game.withField(MineField.fromRows(rows));
        RevealResult r = game.reveal(at(0, 0));
        assertEquals(9999, r.opened().size());
        assertEquals(GameStatus.WON, game.status());
    }

    @Test
    void winningFlagsEveryMine() {
        Game game = Game.withField(MineField.fromRows("*.", ".."));
        game.reveal(at(1, 0));
        game.reveal(at(0, 1));
        assertEquals(GameStatus.PLAYING, game.status());
        game.reveal(at(1, 1));
        assertEquals(GameStatus.WON, game.status());
        assertEquals(CellView.Kind.FLAGGED, game.view(at(0, 0)).kind());
        assertEquals(0, game.remainingMines());
    }

    @Test
    void losingRevealsMinesAndMarksWrongFlags() {
        Game game = Game.withField(MineField.fromRows(
                "*.*",
                "..."));
        game.reveal(at(1, 1));
        game.toggleFlag(at(0, 1)); // wrong
        game.toggleFlag(at(2, 0)); // right
        RevealResult r = game.reveal(at(0, 0));
        assertTrue(r.detonated());
        assertEquals(GameStatus.LOST, game.status());
        assertEquals(CellView.Kind.EXPLODED, game.view(at(0, 0)).kind());
        assertEquals(CellView.Kind.WRONG_FLAG, game.view(at(0, 1)).kind());
        assertEquals(CellView.Kind.FLAGGED, game.view(at(2, 0)).kind());
        assertEquals(CellView.Kind.HIDDEN, game.view(at(1, 0)).kind());
        assertSame(RevealResult.NOTHING, game.reveal(at(1, 0)), "no moves after game over");
        assertFalse(game.toggleFlag(at(1, 0)));
    }

    @Test
    void minesStayHiddenWhilePlaying() {
        Game game = Game.withField(MineField.fromRows("*..", "..."));
        game.reveal(at(1, 1));
        assertEquals(CellView.Kind.HIDDEN, game.view(at(0, 0)).kind());
        assertThrows(IllegalStateException.class, () -> game.number(0));
    }

    @Test
    void chordOpensNeighborsOnlyWhenFlagsMatchTheNumber() {
        Game game = Game.withField(MineField.fromRows(
                "*..",
                "...",
                "..."));
        game.reveal(at(1, 0)); // shows 1
        assertFalse(game.chord(at(1, 0)).changedBoard(), "no flags yet");
        game.toggleFlag(at(0, 0));
        RevealResult r = game.chord(at(1, 0));
        assertTrue(r.changedBoard());
        assertEquals(GameStatus.WON, game.status());
    }

    @Test
    void chordWithAWrongFlagDetonates() {
        Game game = Game.withField(MineField.fromRows(
                "*..",
                "...",
                "..."));
        game.reveal(at(1, 0));
        game.toggleFlag(at(2, 0)); // wrong: the mine is at (0,0)
        assertTrue(game.chord(at(1, 0)).detonated());
        assertEquals(GameStatus.LOST, game.status());
    }

    @Test
    void flagsOnlyGoOnHiddenCellsDuringPlay() {
        Game game = new Game(BoardConfig.BEGINNER, new SplittableRandom(3));
        assertFalse(game.toggleFlag(at(0, 0)), "no flags before the first click");
        game.reveal(at(4, 4));
        assertFalse(game.toggleFlag(at(4, 4)), "open cell");
        Coord hidden = null;
        for (int i = 0; i < 81 && hidden == null; i++) {
            if (!game.isRevealed(i)) {
                hidden = game.grid().coord(i);
            }
        }
        assertTrue(game.toggleFlag(hidden));
        assertEquals(9, game.remainingMines());
        assertSame(RevealResult.NOTHING, game.reveal(hidden), "flags protect cells from clicks");
        assertTrue(game.toggleFlag(hidden));
        assertEquals(10, game.remainingMines());
    }

    @Test
    void clockRunsOnlyWhilePlaying() {
        Game game = Game.withField(MineField.fromRows("*.", ".."));
        game.tick(5);
        assertEquals(0, game.elapsedSeconds());
        game.reveal(at(1, 0));
        game.tick(0.5);
        game.tick(0.25);
        assertEquals(0.75, game.elapsedSeconds(), 1e-12);
        game.reveal(at(0, 0));
        game.tick(10);
        assertEquals(0.75, game.elapsedSeconds(), 1e-12);
    }

    @Test
    void clockRejectsInvalidTimeSteps() {
        Game game = new Game(BoardConfig.BEGINNER, new SplittableRandom(1));
        assertThrows(IllegalArgumentException.class, () -> game.tick(-0.1));
        assertThrows(IllegalArgumentException.class, () -> game.tick(Double.NaN));
        assertThrows(IllegalArgumentException.class, () -> game.tick(Double.POSITIVE_INFINITY));
    }

    @Test
    void offBoardCoordinatesAreProgrammerErrors() {
        Game game = new Game(BoardConfig.BEGINNER, new SplittableRandom(1));
        assertThrows(IllegalArgumentException.class, () -> game.reveal(at(9, 0)));
        assertThrows(IllegalArgumentException.class, () -> game.view(at(-1, 0)));
    }
}
