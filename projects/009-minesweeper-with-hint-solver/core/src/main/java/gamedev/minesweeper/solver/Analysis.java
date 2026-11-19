package gamedev.minesweeper.solver;

import java.util.List;
import java.util.Optional;

/**
 * Everything the solver worked out about a position.
 *
 * @param deductions proven cells, simplest proofs first
 */
public record Analysis(List<Hint.Deduction> deductions, Optional<ProbabilityMap> probabilities) {

    public Analysis {
        deductions = List.copyOf(deductions);
    }
}
