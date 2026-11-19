package gamedev.minesweeper.solver;

import gamedev.minesweeper.core.BoardConfig;
import gamedev.minesweeper.core.Coord;
import gamedev.minesweeper.core.Game;
import gamedev.minesweeper.core.Grid;
import gamedev.minesweeper.core.MineField;

import java.util.SplittableRandom;

/** Realistic mid-game positions: a real opening plus a few extra random safe reveals. */
final class RandomPositions {

    private RandomPositions() {
    }

    static Position next(SplittableRandom rng, int width, int height, int mines) {
        BoardConfig config = new BoardConfig(width, height, mines);
        Coord first = new Coord(rng.nextInt(width), rng.nextInt(height));
        MineField field = MineField.generate(config, first, rng);
        Game game = Game.withField(field);
        game.reveal(first);
        Grid g = field.grid();
        int extra = rng.nextInt(4);
        for (int k = 0; k < extra && !game.status().isOver(); k++) {
            int i = rng.nextInt(g.size());
            if (!field.isMine(i)) {
                game.reveal(g.coord(i));
            }
        }
        boolean[] opened = new boolean[g.size()];
        for (int i = 0; i < g.size(); i++) {
            opened[i] = game.isRevealed(i);
        }
        return Position.of(field, opened);
    }
}
