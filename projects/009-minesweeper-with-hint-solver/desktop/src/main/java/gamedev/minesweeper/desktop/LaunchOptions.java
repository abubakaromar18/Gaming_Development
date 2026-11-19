package gamedev.minesweeper.desktop;

import gamedev.minesweeper.core.BoardConfig;

import java.util.Locale;
import java.util.OptionalLong;

/**
 * Command-line options: {@code --level=beginner|intermediate|expert} and {@code --seed=N}.
 * Without a seed every new game gets a fresh one.
 */
record LaunchOptions(BoardConfig config, OptionalLong seed) {

    static final String USAGE = "usage: desktop [--level=beginner|intermediate|expert] [--seed=N]";

    static LaunchOptions parse(String[] args) {
        BoardConfig config = BoardConfig.BEGINNER;
        OptionalLong seed = OptionalLong.empty();
        for (String arg : args) {
            if (arg.startsWith("--level=")) {
                config = level(arg.substring("--level=".length()));
            } else if (arg.startsWith("--seed=")) {
                String value = arg.substring("--seed=".length());
                try {
                    seed = OptionalLong.of(Long.parseLong(value));
                } catch (NumberFormatException e) {
                    throw new IllegalArgumentException("seed must be an integer, got '" + value + "'", e);
                }
            } else {
                throw new IllegalArgumentException("unknown argument '" + arg + "'");
            }
        }
        return new LaunchOptions(config, seed);
    }

    static BoardConfig level(String name) {
        return switch (name.toLowerCase(Locale.ROOT)) {
            case "beginner" -> BoardConfig.BEGINNER;
            case "intermediate" -> BoardConfig.INTERMEDIATE;
            case "expert" -> BoardConfig.EXPERT;
            default -> throw new IllegalArgumentException("unknown level '" + name + "'");
        };
    }
}
