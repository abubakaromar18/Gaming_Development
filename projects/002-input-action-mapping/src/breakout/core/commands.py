"""The seam between the input layer and the game.

``Breakout`` never sees actions, keys, or sticks; it receives ``Commands``.
This one function is the only place that knows how gameplay actions combine
into game commands, so a replay system, an AI, or a network peer can drive the
game by producing ``Commands`` directly.
"""
from __future__ import annotations

from dataclasses import dataclass

from .actions import Action
from .resolver import ActionState


@dataclass(frozen=True)
class Commands:
    #: Paddle intent in [-1, 1]. Analog sticks give partial speeds.
    move: float = 0.0
    launch: bool = False
    toggle_pause: bool = False


IDLE = Commands()


def gameplay_commands(actions: ActionState) -> Commands:
    # Opposing directions cancel, which is also how a keyboard "left + right"
    # chord behaves in most action games.
    move = actions.value(Action.MOVE_RIGHT) - actions.value(Action.MOVE_LEFT)
    return Commands(
        move=max(-1.0, min(1.0, move)),
        launch=actions.just_pressed(Action.LAUNCH),
        toggle_pause=actions.just_pressed(Action.PAUSE),
    )
