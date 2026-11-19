package gamedev.minesweeper.solver;

/** The reasoning technique behind a deduction, from simplest to most expensive. */
public enum Rule {
    /** One number on its own: it needs exactly as many mines as it has undecided neighbours, or none. */
    SINGLE,
    /** Two overlapping numbers bound how many mines their shared cells hold (covers the subset rule). */
    PAIR,
    /** Every mine layout consistent with all visible numbers and the mine total agrees on this cell. */
    ENUMERATION
}
