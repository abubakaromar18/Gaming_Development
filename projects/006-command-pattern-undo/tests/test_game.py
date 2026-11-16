import pytest

from sokoban.core import Direction, Game, GameConfig, Pos, SolutionPlayer, SolveStatus
from helpers import CORRIDOR, ROOM, game

R, L, U = Direction.RIGHT, Direction.LEFT, Direction.UP


def test_counts_moves_and_pushes_and_detects_win():
    g = game(CORRIDOR)
    assert g.move(R) and not g.solved
    assert g.move(R) and g.solved
    assert (g.board.moves, g.board.pushes) == (2, 1)


def test_moves_are_ignored_after_winning_but_undo_reopens_the_level():
    g = game(CORRIDOR)
    g.move(R), g.move(R)
    assert not g.move(L)
    assert g.undo() and not g.solved
    assert g.move(L)


def test_blocked_move_returns_false_and_leaves_history_alone():
    g = game(CORRIDOR)
    assert not g.move(L)
    assert g.history.undo_depth == 0


def test_restart_only_when_something_changed_and_is_undoable():
    g = game(CORRIDOR)
    assert not g.restart()
    g.move(R)
    assert g.restart() and g.board.player == Pos(1, 1)
    assert g.undo() and g.board.player == Pos(1, 2)


def test_level_navigation_resets_board_and_history():
    g = game(CORRIDOR, ROOM)
    g.move(R)
    assert g.next_level() and g.index == 1
    assert g.history.undo_depth == 0 and g.board.moves == 0
    assert not g.next_level()
    assert g.prev_level() and g.index == 0 and g.board.player == Pos(1, 1)
    assert not g.prev_level()


def test_constructor_validation():
    with pytest.raises(ValueError):
        Game([])
    with pytest.raises(IndexError):
        game(CORRIDOR).select(3)
    with pytest.raises(ValueError):
        GameConfig(history_limit=0)
    with pytest.raises(ValueError):
        GameConfig(solver_max_states=0)


def test_history_limit_comes_from_config():
    g = game(ROOM, config=GameConfig(history_limit=1))
    g.move(L), g.move(L)
    assert g.undo() and not g.undo()


def test_hints_alone_solve_the_level_and_are_undoable():
    g = game(ROOM)
    hints = 0
    while not g.solved:
        result = g.hint()
        assert result.status is SolveStatus.SOLVED and result.moved is not None
        hints += 1
        assert hints < 100
    assert g.board.moves == hints
    while g.undo():
        pass
    assert g.board.snapshot() == type(g.board)(g.level).snapshot()


def test_hint_reuses_cached_plan_while_player_follows_it():
    g = game(ROOM)
    plan = g.solution().solution
    g.hint()
    assert g.solution().solution == plan[1:]
    assert g.solution().explored == 0  # cache hit, no new search


def test_hint_replans_after_player_deviates():
    g = game(ROOM)
    g.hint()
    g.undo()
    g.move(L)
    result = g.hint()
    assert result.status is SolveStatus.SOLVED


def test_check_and_hint_report_deadlock():
    g = game(ROOM)
    g.move(U)  # pushes a box against the goal-less top wall
    assert g.check().status is SolveStatus.UNSOLVABLE
    hint = g.hint()
    assert hint.status is SolveStatus.UNSOLVABLE and hint.moved is None
    g.undo()
    assert g.check().status is SolveStatus.SOLVED


def test_hint_on_solved_level_does_nothing():
    g = game(CORRIDOR)
    g.move(R), g.move(R)
    result = g.hint()
    assert result.moved is None and g.board.moves == 2


class TestSolutionPlayer:
    def test_plays_one_step_per_interval_using_accumulated_dt(self):
        g = game(CORRIDOR)
        player = SolutionPlayer(g, "rR", step_interval=0.1)
        assert player.update(0.05) == 0
        assert player.update(0.05) == 1 and g.board.player == Pos(1, 2)
        assert player.update(0.5) == 1 and player.finished and g.solved

    def test_large_dt_plays_several_steps(self):
        g = game(ROOM)
        solution = g.solution().solution
        # 0.125 is exact in binary, so the accumulator has no rounding drift.
        player = SolutionPlayer(g, solution, step_interval=0.125)
        assert player.update(0.125 * len(solution)) == len(solution)
        assert g.solved

    def test_stops_when_board_no_longer_matches(self):
        g = game(CORRIDOR)
        player = SolutionPlayer(g, "lrR", step_interval=0.1)
        assert player.update(1.0) == 0
        assert player.finished and g.board.moves == 0

    def test_autoplayed_moves_are_undoable(self):
        g = game(CORRIDOR)
        SolutionPlayer(g, "rR", 0.1).update(1.0)
        assert g.undo() and g.undo() and not g.undo()

    @pytest.mark.parametrize("interval, dt", [(0, 0.1), (-1, 0.1), (0.1, -0.1)])
    def test_rejects_bad_timing(self, interval, dt):
        with pytest.raises(ValueError):
            SolutionPlayer(game(CORRIDOR), "r", interval).update(dt)
