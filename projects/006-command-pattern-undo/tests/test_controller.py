from sokoban.cli import check_report, main
from sokoban.core import parse_level
from sokoban.terminal.controller import Controller
from sokoban.terminal.keymap import DEFAULT_KEYMAP, MOVE_DIRECTIONS, Action, action_for
from helpers import CORRIDOR, ROOM, game


def test_every_action_has_a_key_and_unknown_keys_map_to_nothing():
    assert set(DEFAULT_KEYMAP.values()) == set(Action)
    assert action_for("KEY_UP") is Action.UP
    assert action_for("^R") is Action.REDO
    assert action_for("F") is None
    assert set(MOVE_DIRECTIONS) == {Action.UP, Action.DOWN, Action.LEFT, Action.RIGHT}


def test_win_message_and_quit():
    c = Controller(game(CORRIDOR))
    c.handle(Action.RIGHT)
    c.handle(Action.RIGHT)
    assert "Solved in 2 moves, 1 pushes" in c.message
    assert c.handle(Action.QUIT) is False


def test_messages_for_blocked_undo_redo_restart():
    c = Controller(game(CORRIDOR))
    c.handle(Action.LEFT)
    assert c.message == "Blocked."
    c.handle(Action.UNDO)
    assert c.message == "Nothing to undo."
    c.handle(Action.RESTART)
    assert c.message == "Already at the start."
    c.handle(Action.RIGHT)
    c.handle(Action.RESTART)
    assert "undo the restart" in c.message
    c.handle(Action.UNDO)
    c.handle(Action.REDO)
    assert c.message == "Redone."


def test_check_and_hint_messages():
    c = Controller(game(ROOM))
    c.handle(Action.CHECK)
    assert c.message.startswith("Still solvable in")
    c.handle(Action.HINT)
    assert c.message.startswith("Hint:")
    c.handle(Action.UNDO)
    c.handle(Action.UP)
    c.handle(Action.CHECK)
    assert c.message.startswith("Deadlocked")


def test_autoplay_runs_on_ticks_and_any_key_stops_it():
    c = Controller(game(ROOM), autoplay_step=0.1)
    c.handle(Action.AUTOPLAY)
    assert c.autoplaying
    c.tick(0.25)
    assert c.game.board.moves == 2
    c.handle(Action.AUTOPLAY)  # pressing p again just stops
    assert not c.autoplaying and c.game.board.moves == 2
    c.handle(Action.AUTOPLAY)
    c.tick(100.0)
    assert c.game.solved and not c.autoplaying
    assert c.message.startswith("Solved in")


def test_level_change_messages():
    c = Controller(game(CORRIDOR, ROOM))
    c.handle(Action.PREV_LEVEL)
    assert c.message == "This is the first level."
    c.handle(Action.NEXT_LEVEL)
    assert c.message == "L2."


def test_check_report_flags_unsolvable_levels():
    good = parse_level(CORRIDOR, "good")
    bad = parse_level("#######\n#.$$. #\n#   @ #\n#######", "bad")
    lines, ok = check_report([good, bad], max_states=1000)
    assert not ok
    assert "SOLVED" in lines[0] and "rR" in lines[0]
    assert "UNSOLVABLE" in lines[1]


def test_cli_check_mode_exit_codes(tmp_path, capsys):
    assert main(["--check"]) == 0
    assert "Warehouse" in capsys.readouterr().out
    broken = tmp_path / "broken.xsb"
    broken.write_text("#####\n#@ .#\n#####\n")
    assert main(["--check", "--levels", str(broken)]) == 2
    assert "no boxes" in capsys.readouterr().err
    assert main(["--check", "--levels", str(tmp_path / "missing.xsb")]) == 2


def test_cli_rejects_out_of_range_start_level(capsys):
    assert main(["--level", "99"]) == 2
    assert "collection has 6 levels" in capsys.readouterr().err
