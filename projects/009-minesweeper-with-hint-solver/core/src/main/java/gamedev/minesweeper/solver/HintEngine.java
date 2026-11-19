package gamedev.minesweeper.solver;

import gamedev.minesweeper.core.Coord;
import gamedev.minesweeper.core.GameStatus;
import gamedev.minesweeper.core.Grid;
import gamedev.minesweeper.core.PlayerView;

import java.math.BigInteger;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Optional;

/**
 * Constraint-based Minesweeper hints.
 *
 * <p>Every open number is a constraint: "exactly n of these hidden neighbours are mines".
 * The engine works through increasingly expensive techniques and stops at the first
 * one that yields something the player can act on:
 *
 * <ol>
 *   <li>{@link Rule#SINGLE}: one number on its own, applied until nothing changes.</li>
 *   <li>{@link Rule#PAIR}: two overlapping numbers bound their shared cells.</li>
 *   <li>{@link Rule#ENUMERATION}: count every consistent layout (see {@link Enumerator}).</li>
 *   <li>Otherwise, a {@link Hint.Guess} at the cell with the lowest mine probability.</li>
 * </ol>
 *
 * <p>Player flags are deliberately <em>not</em> trusted. The solver reasons from numbers
 * alone, which means a flag on a provably safe cell can be reported as a mistake
 * instead of silently poisoning every later deduction.
 */
public final class HintEngine {

    /** Search nodes allowed per enumeration; keeps a hint well under a frame on any board. */
    public static final long DEFAULT_NODE_BUDGET = 500_000;

    private static final byte OPEN = 0;
    private static final byte UNKNOWN = 1;
    private static final byte SAFE = 2;
    private static final byte MINE = 3;

    private final long nodeBudget;

    public HintEngine() {
        this(DEFAULT_NODE_BUDGET);
    }

    public HintEngine(long nodeBudget) {
        if (nodeBudget < 1) {
            throw new IllegalArgumentException("node budget must be positive, got " + nodeBudget);
        }
        this.nodeBudget = nodeBudget;
    }

    /** The single most useful hint, or empty once the game is over. */
    public Optional<Hint> hint(PlayerView view) {
        if (view.status().isOver()) {
            return Optional.empty();
        }
        if (view.status() == GameStatus.READY) {
            return Optional.of(new Hint.FirstClick());
        }
        Solve solve = new Solve(view);
        solve.runRules();
        Optional<Hint> fromRules = solve.actionable();
        if (fromRules.isPresent()) {
            return fromRules;
        }
        solve.runEnumeration();
        return Optional.of(solve.actionable().orElseGet(solve::bestGuess));
    }

    /** Full analysis, including exact probabilities when affordable (used by the probability overlay). */
    public Analysis analyze(PlayerView view) {
        Solve solve = new Solve(view);
        if (view.status() == GameStatus.PLAYING) {
            solve.runRules();
            solve.runEnumeration();
        }
        return new Analysis(solve.deductions, Optional.ofNullable(solve.probabilities));
    }

    /** State for one analysis: what is known about each cell and how it was proven. */
    private final class Solve {
        private final PlayerView view;
        private final Grid grid;
        private final byte[] state;
        private final List<Hint.Deduction> deductions = new ArrayList<>();
        private int knownMines;
        private ProbabilityMap probabilities;
        private boolean enumerationExact;

        Solve(PlayerView view) {
            this.view = view;
            this.grid = view.grid();
            this.state = new byte[grid.size()];
            for (int i = 0; i < state.length; i++) {
                state[i] = view.isRevealed(i) ? OPEN : UNKNOWN;
            }
        }

        // ------------------------------------------------------------ rules

        void runRules() {
            boolean progress = true;
            while (progress) {
                progress = applySingle() || applyPair();
            }
        }

        /** One pass of the single-number rule over the whole board. */
        private boolean applySingle() {
            boolean progress = false;
            for (Constraint c : constraints()) {
                int number = c.number();
                Coord at = grid.coord(c.source());
                if (c.mines() == 0) {
                    String why = "The " + number + " at " + at.label() + " already touches " + number
                            + " proven mine" + plural(number) + ", so its other hidden neighbors are safe.";
                    progress |= decide(c.cells(), false, Rule.SINGLE, new int[]{c.source()}, why);
                } else if (c.mines() == c.cells().length) {
                    String why = c.mines() == number
                            ? "The " + number + " at " + at.label() + " has exactly " + number + " hidden neighbor"
                              + plural(number) + ", so " + (number == 1 ? "it is a mine." : "they are all mines.")
                            : "The " + number + " at " + at.label() + " needs " + c.mines() + " more mine"
                              + plural(c.mines()) + " and has exactly " + c.mines() + " undecided neighbor"
                              + plural(c.mines()) + " left, so " + (c.mines() == 1 ? "it is a mine." : "they are all mines.");
                    progress |= decide(c.cells(), true, Rule.SINGLE, new int[]{c.source()}, why);
                }
            }
            return progress;
        }

        /**
         * Two numbers A and B that share undecided cells. From A alone, the shared cells
         * hold between {@code lo} and {@code hi} mines. If B still needs no more than
         * {@code lo}, B's private cells are safe; if B needs {@code hi} plus all of its
         * private cells, they are mines. The subset rule is the case where A has no
         * private cells. Stops after the first productive pair so the next deduction is
         * made from fresh constraints.
         */
        private boolean applyPair() {
            List<Constraint> cs = constraints();
            for (Constraint a : cs) {
                for (Constraint b : cs) {
                    if (a == b || grid.distance(a.source(), b.source()) > 2) {
                        continue;
                    }
                    int shared = a.sharedWith(b);
                    if (shared == 0) {
                        continue;
                    }
                    int[] onlyB = b.minus(a);
                    if (onlyB.length == 0) {
                        continue;
                    }
                    int onlyA = a.cells().length - shared;
                    int lo = Math.max(0, a.mines() - onlyA);
                    int hi = Math.min(shared, a.mines());
                    Coord atA = grid.coord(a.source());
                    Coord atB = grid.coord(b.source());
                    int[] evidence = {a.source(), b.source()};
                    if (b.mines() - lo == 0) {
                        String why = "The " + a.number() + " at " + atA.label() + " puts at least " + lo + " mine"
                                + plural(lo) + " in the cells it shares with the " + b.number() + " at " + atB.label()
                                + ". That is all the " + b.number() + " still needs, so its other undecided neighbor"
                                + plural(onlyB.length) + (onlyB.length == 1 ? " is" : " are") + " safe.";
                        if (decide(onlyB, false, Rule.PAIR, evidence, why)) {
                            return true;
                        }
                    } else if (b.mines() - hi == onlyB.length) {
                        String why = "The " + a.number() + " at " + atA.label() + " allows at most " + hi + " mine"
                                + plural(hi) + " in the cells it shares with the " + b.number() + " at " + atB.label()
                                + ". The " + b.number() + " still needs " + b.mines() + ", so its other "
                                + onlyB.length + " undecided neighbor" + plural(onlyB.length)
                                + (onlyB.length == 1 ? " must be a mine." : " must all be mines.");
                        if (decide(onlyB, true, Rule.PAIR, evidence, why)) {
                            return true;
                        }
                    }
                }
            }
            return false;
        }

        /** Current constraints: every open number with undecided neighbours, reduced by known cells. */
        private List<Constraint> constraints() {
            List<Constraint> out = new ArrayList<>();
            int[] scratch = new int[8];
            for (int i = 0; i < state.length; i++) {
                if (state[i] != OPEN) {
                    continue;
                }
                int number = view.number(i);
                int undecided = 0;
                int mines = 0;
                for (int n : grid.neighbors(i)) {
                    if (state[n] == UNKNOWN) {
                        scratch[undecided++] = n;
                    } else if (state[n] == MINE) {
                        mines++;
                    }
                }
                if (undecided == 0) {
                    continue;
                }
                int remaining = number - mines;
                if (remaining < 0 || remaining > undecided) {
                    throw new IllegalStateException("the " + number + " at " + grid.coord(i).label()
                            + " cannot be satisfied: the position is inconsistent");
                }
                int[] cells = Arrays.copyOf(scratch, undecided);
                Arrays.sort(cells);
                out.add(new Constraint(i, number, cells, remaining));
            }
            return out;
        }

        private boolean decide(int[] cells, boolean mine, Rule rule, int[] evidence, String why) {
            boolean changed = false;
            List<Coord> proof = new ArrayList<>(evidence.length);
            for (int e : evidence) {
                proof.add(grid.coord(e));
            }
            for (int c : cells) {
                if (state[c] != UNKNOWN) {
                    continue;
                }
                state[c] = mine ? MINE : SAFE;
                if (mine) {
                    knownMines++;
                }
                deductions.add(new Hint.Deduction(grid.coord(c), mine, rule, proof, why));
                changed = true;
            }
            return changed;
        }

        // ------------------------------------------------------------ enumeration

        void runEnumeration() {
            boolean[] undecided = new boolean[state.length];
            double[] known = new double[state.length];
            for (int i = 0; i < state.length; i++) {
                undecided[i] = state[i] == UNKNOWN;
                known[i] = switch (state[i]) {
                    case SAFE -> 0.0;
                    case MINE -> 1.0;
                    default -> Double.NaN;
                };
            }
            int minesRemaining = view.totalMines() - knownMines;
            Enumerator.Result result = new Enumerator(grid, nodeBudget)
                    .solve(undecided, known, constraints(), minesRemaining);
            if (result == null) {
                return;
            }
            probabilities = result.map();
            enumerationExact = true;
            String layouts = String.format("%,d", result.map().layouts());
            for (int i = 0; i < state.length; i++) {
                if (!undecided[i] || !(result.certainSafe()[i] || result.certainMine()[i])) {
                    continue;
                }
                boolean mine = result.certainMine()[i];
                Coord at = grid.coord(i);
                int[] sources = result.evidence()[i];
                String why;
                if (sources.length == 0) {
                    why = mine
                            ? "The mine total of " + view.totalMines() + " only works out if every cell away from the"
                              + " numbers, including " + at.label() + ", is a mine."
                            : "All " + minesRemaining + " remaining mine" + plural(minesRemaining)
                              + " must sit next to the numbers, so cells away from them, like " + at.label() + ", are safe.";
                } else {
                    why = "Of the " + layouts + " mine layout" + (result.map().layouts().equals(BigInteger.ONE) ? "" : "s")
                            + " that fit every visible number and the mine total, "
                            + (mine ? "every one puts a mine at " : "none puts a mine at ") + at.label() + ".";
                }
                decide(new int[]{i}, mine, Rule.ENUMERATION, sources, why);
            }
        }

        // ------------------------------------------------------------ choosing

        /**
         * Priority: a wrong flag first (it can make chording explode), then a safe
         * cell (progress), then a mine to flag. Within each group, the simplest proof
         * comes first because deductions are recorded in the order they were found.
         */
        Optional<Hint> actionable() {
            for (Hint.Deduction d : deductions) {
                if (!d.mine() && view.isFlagged(grid.index(d.cell()))) {
                    return Optional.of(new Hint.WrongFlag(d.cell(), d));
                }
            }
            for (Hint.Deduction d : deductions) {
                if (!d.mine()) {
                    return Optional.of(d);
                }
            }
            for (Hint.Deduction d : deductions) {
                if (d.mine() && !view.isFlagged(grid.index(d.cell()))) {
                    return Optional.of(d);
                }
            }
            return Optional.empty();
        }

        /** Lowest-probability undecided cell; unflagged cells first, row-major on ties. */
        Hint bestGuess() {
            double[] p = probabilities != null ? null : estimate();
            int best = -1;
            double bestP = Double.POSITIVE_INFINITY;
            boolean bestFlagged = true;
            for (int i = 0; i < state.length; i++) {
                if (state[i] != UNKNOWN) {
                    continue;
                }
                double pi = p != null ? p[i] : probabilities.mineProbability(i);
                boolean flagged = view.isFlagged(i);
                boolean better = best < 0
                        || (bestFlagged && !flagged)
                        || (bestFlagged == flagged && pi < bestP);
                if (better) {
                    best = i;
                    bestP = pi;
                    bestFlagged = flagged;
                }
            }
            if (best < 0) {
                throw new IllegalStateException("game is in progress but no undecided cell remains");
            }
            return new Hint.Guess(grid.coord(best), bestP, enumerationExact);
        }

        /**
         * Cheap fallback when enumeration is over budget: a frontier cell takes the worst
         * local density among its numbers, an interior cell the global density.
         */
        private double[] estimate() {
            double[] p = new double[state.length];
            int undecided = 0;
            for (byte s : state) {
                if (s == UNKNOWN) {
                    undecided++;
                }
            }
            double global = undecided == 0 ? 0 : (double) (view.totalMines() - knownMines) / undecided;
            Arrays.fill(p, global);
            boolean[] touched = new boolean[state.length];
            for (Constraint c : constraints()) {
                double local = (double) c.mines() / c.cells().length;
                for (int cell : c.cells()) {
                    p[cell] = touched[cell] ? Math.max(p[cell], local) : local;
                    touched[cell] = true;
                }
            }
            return p;
        }
    }

    private static String plural(int n) {
        return n == 1 ? "" : "s";
    }
}
