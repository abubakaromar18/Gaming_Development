package gamedev.minesweeper.sim;

import gamedev.minesweeper.core.BoardConfig;
import gamedev.minesweeper.core.Coord;
import gamedev.minesweeper.core.Game;
import gamedev.minesweeper.core.GameStatus;
import gamedev.minesweeper.solver.HintEngine;

import java.util.LinkedHashMap;
import java.util.Map;
import java.util.SplittableRandom;

/**
 * Command-line benchmark: auto-plays seeded games at each difficulty and prints the
 * win rate, guesses per game and time per game. Seeds are 1..N, so runs are repeatable.
 */
public final class Simulation {

    private Simulation() {
    }

    public static void main(String[] args) {
        int games = 300;
        if (args.length > 0) {
            try {
                games = Integer.parseInt(args[0]);
            } catch (NumberFormatException e) {
                System.err.println("usage: Simulation [games]   (got '" + args[0] + "')");
                System.exit(2);
            }
        }
        if (games < 1) {
            System.err.println("games must be positive");
            System.exit(2);
        }

        Map<String, BoardConfig> levels = new LinkedHashMap<>();
        levels.put("Beginner", BoardConfig.BEGINNER);
        levels.put("Intermediate", BoardConfig.INTERMEDIATE);
        levels.put("Expert", BoardConfig.EXPERT);

        AutoPlayer player = new AutoPlayer(new HintEngine());
        System.out.printf("%-13s %7s %8s %13s %10s%n", "level", "games", "win %", "guesses/game", "ms/game");
        for (Map.Entry<String, BoardConfig> level : levels.entrySet()) {
            Stats s = run(player, level.getValue(), games);
            System.out.printf("%-13s %7d %7.1f%% %13.2f %10.2f%n",
                    level.getKey(), games, 100.0 * s.wins / games, (double) s.guesses / games, s.millis / games);
        }
    }

    private record Stats(int wins, int guesses, double millis) { }

    private static Stats run(AutoPlayer player, BoardConfig config, int games) {
        Coord center = new Coord(config.width() / 2, config.height() / 2);
        int wins = 0;
        int guesses = 0;
        long start = System.nanoTime();
        for (long seed = 1; seed <= games; seed++) {
            AutoPlayer.Outcome o = player.play(new Game(config, new SplittableRandom(seed)), center);
            if (o.status() == GameStatus.WON) {
                wins++;
            }
            guesses += o.guesses();
        }
        return new Stats(wins, guesses, (System.nanoTime() - start) / 1e6);
    }
}
