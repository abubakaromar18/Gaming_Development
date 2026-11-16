from sokoban.core import Game, parse_level

# Player, gap, box, goal: "rR" solves it.
CORRIDOR = """
######
#@ $.#
######
"""

# Two boxes, two goals, and a goal-less top wall that is all dead squares.
ROOM = """
######
#    #
# $$ #
#  @ #
#..  #
######
"""


def level(text: str = ROOM, name: str = "test"):
    return parse_level(text, name)


def game(*texts: str, **kwargs) -> Game:
    return Game([level(t, f"L{i + 1}") for i, t in enumerate(texts or (ROOM,))], **kwargs)
