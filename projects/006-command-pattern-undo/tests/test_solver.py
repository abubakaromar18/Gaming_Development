import time

import pytest

from sokoban.core import Pos, SolveStatus, check_solution, dead_squares, parse_level, solve
from sokoban.resources import bundled_levels
from helpers import CORRIDOR, ROOM, level


def test_solves_trivial_corridor():
    result = solve(level(CORRIDOR))
    assert result.status is SolveStatus.SOLVED
    assert result.solution == "rR"
    assert result.pushes == 1


def test_solution_is_push_optimal():
    # Two pushes are enough; BFS over pushes must find exactly that, however
    # much walking it takes between them.
    text = """
#######
#@    #
# $ $ #
# . . #
#######
"""
    result = solve(parse_level(text))
    assert result.pushes == 2
    assert check_solution(parse_level(text), result.solution).solved


def test_dead_squares_are_corners_and_goal_less_walls():
    lv = level(ROOM)
    dead = dead_squares(lv)
    assert Pos(1, 1) in dead and Pos(1, 4) in dead  # top corners
    assert Pos(4, 4) in dead  # bottom-right corner, no goal
    assert Pos(1, 2) in dead  # top wall has no goal: a box there can never come back down
    assert not (dead & lv.goals)
    assert Pos(2, 2) not in dead


def test_box_on_dead_square_is_unsolvable_without_search():
    lv = level(ROOM)
    result = solve(lv, Pos(3, 3), frozenset({Pos(1, 1), Pos(2, 3)}))
    assert result.status is SolveStatus.UNSOLVABLE and result.explored == 0


def test_detects_freeze_deadlock_by_exhausting_search():
    # Two boxes side by side against a wall with goals elsewhere along that
    # wall: no single box is on a dead square, but neither can be moved along.
    text = """
#######
#.$$. #
#   @ #
#######
"""
    lv = parse_level(text)
    result = solve(lv)
    assert result.status is SolveStatus.UNSOLVABLE
    assert result.explored > 0


def test_budget_exhaustion_reports_gave_up():
    result = solve(bundled_levels()[-1], max_states=10)
    assert result.status is SolveStatus.GAVE_UP and result.solution is None


def test_already_solved_position():
    lv = level(CORRIDOR)
    result = solve(lv, Pos(1, 3), lv.goals)
    assert result.status is SolveStatus.SOLVED and result.solution == ""


def test_rejects_bad_budget():
    with pytest.raises(ValueError):
        solve(level(), max_states=0)


@pytest.mark.parametrize("text, expected", [("rR", True), ("RR", True), ("r R", True), ("r", False)])
def test_check_solution_outcomes(text, expected):
    assert check_solution(level(CORRIDOR), text).solved is expected


def test_check_solution_reports_blocked_and_invalid_steps():
    blocked = check_solution(level(CORRIDOR), "rRRr")
    assert blocked.solved  # stops at the winning push, ignoring the rest
    wall = check_solution(level(CORRIDOR), "l")
    assert not wall.solved and "blocked move 'l' at step 1" in wall.reason
    junk = check_solution(level(CORRIDOR), "rx")
    assert not junk.solved and "invalid move 'x' at step 2" in junk.reason and junk.moves_applied == 1


def test_partial_solution_reports_progress():
    result = check_solution(level(CORRIDOR), "r")
    assert result.reason == "0/1 boxes on goals after all moves"


@pytest.mark.parametrize("lv", bundled_levels(), ids=lambda lv: lv.name)
def test_every_bundled_level_is_solvable_and_solution_replays(lv):
    result = solve(lv)
    assert result.status is SolveStatus.SOLVED
    check = check_solution(lv, result.solution)
    assert check.solved, check.reason
    assert check.moves_applied == len(result.solution)


def test_solver_is_fast_enough_for_interactive_hints():
    """Stress check: solving the whole bundled set should stay well under a second or two."""
    start = time.perf_counter()
    for lv in bundled_levels():
        solve(lv)
    assert time.perf_counter() - start < 5.0
