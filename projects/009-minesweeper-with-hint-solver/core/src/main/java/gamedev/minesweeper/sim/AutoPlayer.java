package gamedev.minesweeper.sim;

import gamedev.minesweeper.core.Coord;
import gamedev.minesweeper.core.Game;
import gamedev.minesweeper.core.GameStatus;
import gamedev.minesweeper.solver.Hint;
import gamedev.minesweeper.solver.HintEngine;

/**
 * Plays a game by following the hint engine and nothing else. Used to measure solver
 * strength and, in tests, to check that every "proven" deduction is actually true.
 */
public final class AutoPlayer {

    /**
     * @param guesses      reveals made on a {@link Hint.Guess}
     * @param deadlyGuess  true if the game was lost on a guess (as opposed to a "proven" move)
     */
    public record Outcome(GameStatus status, int moves, int guesses, boolean deadlyGuess) { }

    private final HintEngine engine;

    public AutoPlayer(HintEngine engine) {
        this.engine = engine;
    }

    public Outcome play(Game game, Coord firstClick) {
        int moves = 0;
        int guesses = 0;
        boolean lastWasGuess = false;
        // Each move opens or flags at least one cell, or removes a flag the solver proved wrong.
        int limit = game.grid().size() * 3;
        while (!game.status().isOver()) {
            if (moves++ > limit) {
                throw new IllegalStateException("auto-player made no progress after " + limit + " moves");
            }
            Hint hint = engine.hint(game).orElseThrow();
            lastWasGuess = false;
            switch (hint) {
                case Hint.FirstClick f -> game.reveal(firstClick);
                case Hint.Deduction d when d.mine() -> game.toggleFlag(d.cell());
                case Hint.Deduction d -> game.reveal(d.cell());
                case Hint.WrongFlag w -> game.toggleFlag(w.cell());
                case Hint.Guess g -> {
                    guesses++;
                    lastWasGuess = true;
                    game.reveal(g.cell());
                }
            }
        }
        return new Outcome(game.status(), moves, guesses, game.status() == GameStatus.LOST && lastWasGuess);
    }
}
