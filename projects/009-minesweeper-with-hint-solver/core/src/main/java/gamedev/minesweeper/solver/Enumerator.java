package gamedev.minesweeper.solver;

import gamedev.minesweeper.core.Grid;

import java.math.BigDecimal;
import java.math.BigInteger;
import java.math.MathContext;
import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Deque;
import java.util.List;

/**
 * Exact probabilities by counting mine layouts.
 *
 * <p>Undecided cells split into the <em>frontier</em> (next to at least one number) and
 * the <em>interior</em> (no information at all). Frontier cells are grouped into
 * independent components: two cells share a component only if some number touches
 * both. Each component is enumerated with backtracking, recording how many valid
 * assignments use k mines. Components are then combined by convolution, and the
 * interior is counted in closed form: with {@code r} mines left over for {@code I}
 * interior cells there are C(I, r) ways to place them. This turns one exponential
 * search over the whole frontier into several small ones.
 *
 * <p>All counting uses {@link BigInteger}, so "probability is exactly 0" and "exactly
 * 1" are integer comparisons, never floating-point guesses.
 */
final class Enumerator {

    /** Thrown internally to unwind the search once the node budget is spent. */
    private static final class BudgetExceeded extends RuntimeException {
        private static final long serialVersionUID = 1L;

        BudgetExceeded() {
            super(null, null, false, false);
        }
    }

    /** Result for the undecided cells; arrays are indexed by board cell. */
    record Result(ProbabilityMap map, boolean[] certainSafe, boolean[] certainMine, int[][] evidence) { }

    private final Grid grid;
    private final long nodeBudget;
    private long nodes;

    Enumerator(Grid grid, long nodeBudget) {
        this.grid = grid;
        this.nodeBudget = nodeBudget;
    }

    /**
     * @param undecided      cells whose content is not yet known
     * @param known          probability already fixed by earlier rules (0 or 1), NaN otherwise
     * @param constraints    reduced constraints over undecided cells only
     * @param minesRemaining mines not yet accounted for by earlier deductions
     * @return null if the search would exceed the node budget
     */
    Result solve(boolean[] undecided, double[] known, List<Constraint> constraints, int minesRemaining) {
        nodes = 0;
        int size = grid.size();

        List<Component> components = buildComponents(undecided, constraints);
        boolean[] frontier = new boolean[size];
        for (Component comp : components) {
            for (int c : comp.cells) {
                frontier[c] = true;
            }
        }
        int interior = 0;
        for (int i = 0; i < size; i++) {
            if (undecided[i] && !frontier[i]) {
                interior++;
            }
        }

        try {
            for (Component comp : components) {
                comp.enumerate(minesRemaining);
            }
        } catch (BudgetExceeded e) {
            return null;
        }

        BigInteger[] chooseInterior = binomialRow(interior);
        BigInteger[] chooseInteriorLessOne = interior > 0 ? binomialRow(interior - 1) : new BigInteger[0];

        BigInteger[] all = {BigInteger.ONE};
        for (Component comp : components) {
            all = convolve(all, comp.distribution());
        }
        BigInteger total = BigInteger.ZERO;
        for (int t = 0; t < all.length; t++) {
            total = total.add(all[t].multiply(at(chooseInterior, minesRemaining - t)));
        }
        if (total.signum() == 0) {
            throw new IllegalStateException("no mine layout matches the visible numbers");
        }

        double[] probability = known.clone();
        boolean[] certainSafe = new boolean[size];
        boolean[] certainMine = new boolean[size];
        int[][] evidence = new int[size][];

        for (int ci = 0; ci < components.size(); ci++) {
            Component comp = components.get(ci);
            BigInteger[] others = {BigInteger.ONE};
            for (int cj = 0; cj < components.size(); cj++) {
                if (cj != ci) {
                    others = convolve(others, components.get(cj).distribution());
                }
            }
            // weight[k]: number of ways to complete the board if this component uses k mines.
            BigInteger[] weight = new BigInteger[comp.cells.length + 1];
            for (int k = 0; k < weight.length; k++) {
                BigInteger w = BigInteger.ZERO;
                for (int t = 0; t < others.length; t++) {
                    w = w.add(others[t].multiply(at(chooseInterior, minesRemaining - k - t)));
                }
                weight[k] = w;
            }
            int[] sources = comp.sources();
            for (int local = 0; local < comp.cells.length; local++) {
                BigInteger mineWeight = BigInteger.ZERO;
                for (int k = 0; k < weight.length; k++) {
                    mineWeight = mineWeight.add(BigInteger.valueOf(comp.mineCounts[k][local]).multiply(weight[k]));
                }
                int cell = comp.cells[local];
                record(cell, mineWeight, total, probability, certainSafe, certainMine);
                evidence[cell] = sources;
            }
        }

        if (interior > 0) {
            BigInteger mineWeight = BigInteger.ZERO;
            for (int t = 0; t < all.length; t++) {
                mineWeight = mineWeight.add(all[t].multiply(at(chooseInteriorLessOne, minesRemaining - t - 1)));
            }
            for (int i = 0; i < size; i++) {
                if (undecided[i] && !frontier[i]) {
                    record(i, mineWeight, total, probability, certainSafe, certainMine);
                    evidence[i] = new int[0];
                }
            }
        }
        return new Result(new ProbabilityMap(probability, total), certainSafe, certainMine, evidence);
    }

    private static void record(int cell, BigInteger mineWeight, BigInteger total,
                               double[] probability, boolean[] certainSafe, boolean[] certainMine) {
        probability[cell] = new BigDecimal(mineWeight).divide(new BigDecimal(total), MathContext.DECIMAL64).doubleValue();
        certainSafe[cell] = mineWeight.signum() == 0;
        certainMine[cell] = mineWeight.equals(total);
    }

    // ---------------------------------------------------------------- components

    private List<Component> buildComponents(boolean[] undecided, List<Constraint> constraints) {
        int size = grid.size();
        // Which constraints touch each cell.
        List<List<Integer>> byCell = new ArrayList<>(size);
        for (int i = 0; i < size; i++) {
            byCell.add(new ArrayList<>(2));
        }
        for (int ci = 0; ci < constraints.size(); ci++) {
            for (int c : constraints.get(ci).cells()) {
                if (!undecided[c]) {
                    throw new IllegalArgumentException("constraint mentions decided cell " + grid.coord(c));
                }
                byCell.get(c).add(ci);
            }
        }

        // Flood over the cell <-> constraint graph. BFS order also makes a good
        // backtracking order: neighbouring cells are assigned together, so each
        // constraint becomes fully assigned (and prunable) early.
        int[] componentOf = new int[size];
        Arrays.fill(componentOf, -1);
        boolean[] constraintSeen = new boolean[constraints.size()];
        List<Component> components = new ArrayList<>();
        for (int seed = 0; seed < size; seed++) {
            if (byCell.get(seed).isEmpty() || componentOf[seed] >= 0) {
                continue;
            }
            List<Integer> cells = new ArrayList<>();
            List<Integer> compConstraints = new ArrayList<>();
            Deque<Integer> queue = new ArrayDeque<>();
            queue.add(seed);
            componentOf[seed] = components.size();
            while (!queue.isEmpty()) {
                int cell = queue.poll();
                cells.add(cell);
                for (int ci : byCell.get(cell)) {
                    if (constraintSeen[ci]) {
                        continue;
                    }
                    constraintSeen[ci] = true;
                    compConstraints.add(ci);
                    for (int other : constraints.get(ci).cells()) {
                        if (componentOf[other] < 0) {
                            componentOf[other] = components.size();
                            queue.add(other);
                        }
                    }
                }
            }
            List<Constraint> picked = new ArrayList<>(compConstraints.size());
            for (int ci : compConstraints) {
                picked.add(constraints.get(ci));
            }
            components.add(new Component(cells.stream().mapToInt(Integer::intValue).toArray(), picked));
        }
        return components;
    }

    /** One independent group of frontier cells and the constraints over them. */
    private final class Component {
        final int[] cells;
        final List<Constraint> constraints;
        /** For each local cell, the local indices of the constraints that mention it. */
        final int[][] cellConstraints;
        final int[] required;
        final int[] assignedMines;
        final int[] unassigned;
        final boolean[] value;
        /** solutions[k]: valid assignments that use exactly k mines. */
        final long[] solutions;
        /** mineCounts[k][cell]: of those, how many put a mine on that cell. */
        final long[][] mineCounts;

        Component(int[] cells, List<Constraint> constraints) {
            this.cells = cells;
            this.constraints = constraints;
            int[] localOf = new int[grid.size()];
            for (int i = 0; i < cells.length; i++) {
                localOf[cells[i]] = i;
            }
            List<List<Integer>> touching = new ArrayList<>(cells.length);
            for (int i = 0; i < cells.length; i++) {
                touching.add(new ArrayList<>(4));
            }
            required = new int[constraints.size()];
            unassigned = new int[constraints.size()];
            assignedMines = new int[constraints.size()];
            for (int ci = 0; ci < constraints.size(); ci++) {
                Constraint c = constraints.get(ci);
                required[ci] = c.mines();
                unassigned[ci] = c.cells().length;
                for (int cell : c.cells()) {
                    touching.get(localOf[cell]).add(ci);
                }
            }
            cellConstraints = new int[cells.length][];
            for (int i = 0; i < cells.length; i++) {
                cellConstraints[i] = touching.get(i).stream().mapToInt(Integer::intValue).toArray();
            }
            value = new boolean[cells.length];
            solutions = new long[cells.length + 1];
            mineCounts = new long[cells.length + 1][cells.length];
        }

        void enumerate(int maxMines) {
            search(0, 0, maxMines);
        }

        private void search(int depth, int minesUsed, int maxMines) {
            if (++nodes > nodeBudget) {
                throw new BudgetExceeded();
            }
            if (depth == cells.length) {
                solutions[minesUsed]++;
                long[] counts = mineCounts[minesUsed];
                for (int i = 0; i < cells.length; i++) {
                    if (value[i]) {
                        counts[i]++;
                    }
                }
                return;
            }
            if (assign(depth, false)) {
                search(depth + 1, minesUsed, maxMines);
            }
            unassign(depth, false);
            if (minesUsed < maxMines) {
                if (assign(depth, true)) {
                    search(depth + 1, minesUsed + 1, maxMines);
                }
                unassign(depth, true);
            }
        }

        /** Sets a cell and reports whether every constraint it touches can still be satisfied. */
        private boolean assign(int local, boolean mine) {
            value[local] = mine;
            boolean ok = true;
            for (int ci : cellConstraints[local]) {
                unassigned[ci]--;
                if (mine) {
                    assignedMines[ci]++;
                }
                if (assignedMines[ci] > required[ci] || assignedMines[ci] + unassigned[ci] < required[ci]) {
                    ok = false;
                }
            }
            return ok;
        }

        private void unassign(int local, boolean mine) {
            for (int ci : cellConstraints[local]) {
                unassigned[ci]++;
                if (mine) {
                    assignedMines[ci]--;
                }
            }
            value[local] = false;
        }

        BigInteger[] distribution() {
            BigInteger[] d = new BigInteger[solutions.length];
            for (int k = 0; k < d.length; k++) {
                d[k] = BigInteger.valueOf(solutions[k]);
            }
            return d;
        }

        int[] sources() {
            return constraints.stream().mapToInt(Constraint::source).sorted().toArray();
        }
    }

    // ---------------------------------------------------------------- combinatorics

    /** C(n, 0..n), built with the multiplicative recurrence C(n, r+1) = C(n, r)(n-r)/(r+1). */
    static BigInteger[] binomialRow(int n) {
        BigInteger[] row = new BigInteger[n + 1];
        row[0] = BigInteger.ONE;
        for (int r = 0; r < n; r++) {
            row[r + 1] = row[r].multiply(BigInteger.valueOf(n - r)).divide(BigInteger.valueOf(r + 1));
        }
        return row;
    }

    private static BigInteger at(BigInteger[] row, int r) {
        return r >= 0 && r < row.length ? row[r] : BigInteger.ZERO;
    }

    /** Polynomial product: ways to split a mine total across two independent groups. */
    static BigInteger[] convolve(BigInteger[] a, BigInteger[] b) {
        BigInteger[] out = new BigInteger[a.length + b.length - 1];
        Arrays.fill(out, BigInteger.ZERO);
        for (int i = 0; i < a.length; i++) {
            if (a[i].signum() == 0) {
                continue;
            }
            for (int j = 0; j < b.length; j++) {
                out[i + j] = out[i + j].add(a[i].multiply(b[j]));
            }
        }
        return out;
    }
}
