import random

import pytest

from sokoban.core import Board, CommandHistory, Direction, IllegalMoveError, MoveCommand, Pos, RestartCommand
from helpers import CORRIDOR, ROOM, level

U, D, L, R = Direction.UP, Direction.DOWN, Direction.LEFT, Direction.RIGHT


def fresh(text=ROOM):
    board = Board(level(text))
    return board, CommandHistory(board)


def test_walk_and_undo():
    board, history = fresh()
    history.execute(MoveCommand(R))
    assert board.player == Pos(3, 4) and board.moves == 1 and board.pushes == 0
    history.undo()
    assert board.player == Pos(3, 3) and board.moves == 0


def test_push_moves_box_and_undo_pulls_it_back():
    board, history = fresh()
    cmd = MoveCommand(U)
    history.execute(cmd)
    assert cmd.pushed and cmd.notation == "U"
    assert board.boxes == {Pos(2, 2), Pos(1, 3)}
    assert board.pushes == 1
    history.undo()
    assert board.boxes == {Pos(2, 2), Pos(2, 3)}
    assert board.player == Pos(3, 3) and board.pushes == 0


def test_move_into_wall_raises_and_is_not_recorded():
    board, history = fresh(CORRIDOR)
    with pytest.raises(IllegalMoveError):
        history.execute(MoveCommand(L))
    assert history.undo_depth == 0 and board.moves == 0


def test_cannot_push_box_into_wall():
    board, history = fresh(ROOM)
    history.execute(MoveCommand(U))  # box now against the top wall
    assert not board.can_move(U)


def test_cannot_push_box_into_box():
    board, history = fresh(ROOM)
    history.execute(MoveCommand(R))
    history.execute(MoveCommand(U))  # player at (2,4), next to box at (2,3) with box (2,2) behind it
    assert not board.can_move(L)
    with pytest.raises(IllegalMoveError):
        history.execute(MoveCommand(L))


def test_redo_replays_and_new_command_clears_redo_branch():
    board, history = fresh(CORRIDOR)
    history.execute(MoveCommand(R))
    history.execute(MoveCommand(R))
    history.undo()
    history.undo()
    assert history.redo_depth == 2
    history.redo()
    assert board.player == Pos(1, 2) and history.redo_depth == 1
    history.execute(MoveCommand(L))
    assert not history.can_redo
    assert history.redo() is None


def test_undo_and_redo_on_empty_history_are_no_ops():
    board, history = fresh()
    before = board.snapshot()
    assert history.undo() is None and history.redo() is None
    assert board.snapshot() == before


def test_restart_is_itself_undoable():
    board, history = fresh(CORRIDOR)
    history.execute(MoveCommand(R))
    history.execute(MoveCommand(R))
    mid = board.snapshot()
    history.execute(RestartCommand())
    assert board.snapshot() == Board(board.level).snapshot()
    history.undo()
    assert board.snapshot() == mid
    history.redo()
    assert board.moves == 0
    assert history.notation() == ""


def test_notation_logs_moves_since_last_restart():
    board, history = fresh(CORRIDOR)
    history.execute(MoveCommand(R))
    history.execute(RestartCommand())
    history.execute(MoveCommand(R))
    history.execute(MoveCommand(R))
    assert history.notation() == "rR"


def test_history_limit_forgets_oldest_steps():
    board, history = fresh()
    history = CommandHistory(board, limit=2)
    for d in (L, L, R):
        history.execute(MoveCommand(d))
    assert history.undo_depth == 2
    history.undo()
    history.undo()
    assert history.undo() is None
    assert board.player == Pos(3, 2)  # the first L can no longer be undone


def test_invalid_limit_rejected():
    with pytest.raises(ValueError):
        CommandHistory(Board(level()), limit=0)


def test_restart_undo_before_execute_is_an_error():
    with pytest.raises(RuntimeError):
        RestartCommand().undo(Board(level()))


@pytest.mark.parametrize("seed", range(20))
def test_random_play_fully_unwinds_and_replays(seed):
    """Property: any legal sequence undone completely returns to the start, and
    redone completely reproduces the exact final position."""
    rng = random.Random(seed)
    board, history = fresh()
    start = board.snapshot()
    for _ in range(300):
        legal = [d for d in Direction if board.can_move(d)]
        if rng.random() < 0.05:
            history.execute(RestartCommand())
        elif legal:
            history.execute(MoveCommand(rng.choice(legal)))
        if rng.random() < 0.2:
            history.undo()
    end = board.snapshot()
    depth = history.undo_depth
    while history.undo():
        pass
    assert board.snapshot() == start
    for _ in range(depth):
        history.redo()
    assert board.snapshot() == end
