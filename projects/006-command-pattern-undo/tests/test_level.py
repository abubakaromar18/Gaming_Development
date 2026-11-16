import pytest

from sokoban.core import LevelError, Pos, parse_collection, parse_level
from sokoban.core.board import Board
from sokoban.core.render import to_xsb
from helpers import ROOM


def test_parses_every_symbol():
    level = parse_level("#####\n#+*$#\n#   #\n#####")
    assert level.start_player == Pos(1, 1)
    assert level.goals == {Pos(1, 1), Pos(1, 2)}
    assert level.start_boxes == {Pos(1, 2), Pos(1, 3)}
    assert (level.width, level.height) == (5, 4)


def test_ragged_lines_and_alternative_floor_chars():
    level = parse_level("######\n#@-$.#\n#_####\n###")
    assert (level.width, level.height) == (6, 4)
    assert {Pos(1, 2), Pos(2, 1)} <= level.interior


@pytest.mark.parametrize(
    "text, message",
    [
        ("", "empty"),
        ("#####\n#$ .#\n#####", "exactly one player"),
        ("#####\n#@@$.\n#####", "exactly one player"),
        ("#####\n#@  #\n#####", "no boxes"),
        ("######\n#@$$.#\n######", "2 boxes but 1 goals"),
        ("####\n#@*#\n####", "already starts on a goal"),
        ("#####\n#@$x.#\n#####", "unknown symbol 'x' at line 2, column 4"),
        ("#####\n#@$.\n#####", "not enclosed"),
        ("#######\n#@ #$.#\n#######", "outside the playable area"),
    ],
)
def test_rejects_unplayable_levels(text, message):
    with pytest.raises(LevelError, match=message):
        parse_level(text)


def test_collection_uses_comments_as_names():
    text = """
; A file header that is not a level name

; Tiny
#####
#@$.#
#####

#####
#.$@#
#####
"""
    levels = parse_collection(text)
    assert [lv.name for lv in levels] == ["Tiny", "Level 2"]


def test_empty_collection_is_an_error():
    with pytest.raises(LevelError):
        parse_collection("; only comments\n\n")


def test_render_round_trips():
    level = parse_level(ROOM)
    assert parse_level("\n".join(to_xsb(Board(level)))) == level
