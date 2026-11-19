package gamedev.minesweeper.sim;

import gamedev.minesweeper.core.BoardConfig;
import gamedev.minesweeper.core.Coord;
import gamedev.minesweeper.core.Game;
import gamedev.minesweeper.core.GameStatus;
import gamedev.minesweeper.core.MineField;
import gamedev.minesweeper.solver.Hint;
import gamedev.minesweeper.solver.HintEngine;
import org.junit.jupiter.api.Test;

import java.util.SplittableRandom;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertInstanceOf;
import static org.junit.jupiter.api.Assertions.assertTrue;

/** End-to-end: the hint engine playing full games. */
class AutoPlayerTest {

    private final HintEngine engine = new HintEngine();

    /**
     * Soundness stress test: play hundreds of full games and check every hint against
     * the hidden layout. A proven-safe cell must never be a mine, a proven mine must be
     * one, and a reported wrong flag must sit on a safe cell.
     */
    @Test
    void hintsNeverLieAcrossFullGames() {
        checkSoundness(BoardConfig.BEGINNER, 150);
        checkSoundness(BoardConfig.INTERMEDIATE, 40);
        checkSoundness(BoardConfig.EXPERT, 15);
    }

    private void checkSoundness(BoardConfig config, int games) {
        Coord center = new Coord(config.width() / 2, config.height() / 2);
        for (long seed = 1; seed <= games; seed++) {
            MineField field = MineField.generate(config, center, new SplittableRandom(seed));
            Game game = Game.withField(field);
            game.reveal(center);
            while (!game.status().isOver()) {
                Hint hint = engine.hint(game).orElseThrow();
                switch (hint) {
                    case Hint.FirstClick f -> throw new AssertionError("game already started");
                    case Hint.Deduction d -> {
                        assertEquals(field.isMine(d.cell()), d.mine(), "seed " + seed + ": " + d.explanation());
                        if (d.mine()) {
                            assertTrue(game.toggleFlag(d.cell()));
                        } else {
                            assertFalse(game.reveal(d.cell()).detonated());
                        }
                    }
                    case Hint.WrongFlag w -> {
                        assertFalse(field.isMine(w.cell()));
                        game.toggleFlag(w.cell());
                    }
                    case Hint.Guess g -> game.reveal(g.cell());
                }
            }
        }
    }

    @Test
    void solverWinsMostBeginnerGamesAndOnlyLosesOnGuesses() {
        AutoPlayer player = new AutoPlayer(engine);
        int wins = 0;
        for (long seed = 1; seed <= 100; seed++) {
            AutoPlayer.Outcome o = player.play(new Game(BoardConfig.BEGINNER, new SplittableRandom(seed)), new Coord(4, 4));
            if (o.status() == GameStatus.WON) {
                wins++;
            } else {
                assertTrue(o.deadlyGuess(), "seed " + seed + " was lost on a move the solver called safe");
            }
        }
        assertTrue(wins >= 85, "expected at least 85% wins on beginner, got " + wins);
    }

    @Test
    void autoPlayIsDeterministicPerSeed() {
        AutoPlayer player = new AutoPlayer(engine);
        Coord start = new Coord(8, 8);
        for (long seed = 1; seed <= 10; seed++) {
            AutoPlayer.Outcome a = player.play(new Game(BoardConfig.INTERMEDIATE, new SplittableRandom(seed)), start);
            AutoPlayer.Outcome b = player.play(new Game(BoardConfig.INTERMEDIATE, new SplittableRandom(seed)), start);
            assertEquals(a, b);
        }
    }

    @Test
    void wrongFlagsAreCorrectedDuringPlay() {
        // Mines in the top corners. The player wrongly flags the top-middle cell, then
        // opens the bottom: the flood fill stops at the flag, and the 0 below it proves it wrong.
        Game game = Game.withField(MineField.fromRows(
                "*...*",
                ".....",
                "....."));
        game.reveal(new Coord(0, 1));
        assertTrue(game.toggleFlag(new Coord(2, 0)));
        game.reveal(new Coord(2, 2));
        Hint.WrongFlag w = assertInstanceOf(Hint.WrongFlag.class, engine.hint(game).orElseThrow());
        assertEquals(new Coord(2, 0), w.cell());
        AutoPlayer.Outcome o = new AutoPlayer(engine).play(game, new Coord(2, 2));
        assertEquals(GameStatus.WON, o.status());
    }
}
