"""Draws game and menu state. Reads state, never changes it.

All visuals are procedural: rectangles, a colour ramp for brick rows, and
pygame's built-in default font. Prompts are generated from the live action
map, so after a rebind the on-screen hints update automatically.
"""
from __future__ import annotations

import colorsys

import pygame

from ..core.actions import Action
from ..core.bindings import ActionMap, describe_source
from ..core.breakout import Breakout, Phase
from ..core.menu import BindingsMenu

BG = (14, 16, 24)
FG = (230, 232, 240)
DIM = (130, 136, 156)
ACCENT = (255, 196, 64)
PANEL = (28, 32, 48)


def row_colour(row: int, rows: int) -> tuple[int, int, int]:
    """Warm at the top (high value), cool at the bottom."""
    hue = 0.0 + 0.6 * row / max(1, rows - 1)
    r, g, b = colorsys.hsv_to_rgb(hue, 0.65, 0.95)
    return int(r * 255), int(g * 255), int(b * 255)


def binding_hint(action_map: ActionMap, action: Action, limit: int = 2) -> str:
    sources = action_map.sources_for(action)[:limit]
    return " / ".join(describe_source(s) for s in sources) or "unbound"


class Renderer:
    def __init__(self, surface: pygame.Surface) -> None:
        self.surface = surface
        self.font = pygame.font.Font(None, 24)
        self.big = pygame.font.Font(None, 48)

    def _text(self, text: str, pos: tuple[float, float], colour=FG, font=None, centre=False) -> None:
        img = (font or self.font).render(text, True, colour)
        rect = img.get_rect()
        if centre:
            rect.center = (int(pos[0]), int(pos[1]))
        else:
            rect.topleft = (int(pos[0]), int(pos[1]))
        self.surface.blit(img, rect)

    def draw_game(self, game: Breakout, action_map: ActionMap, flash: float) -> None:
        cfg = game.cfg
        s = self.surface
        s.fill(BG)
        if flash > 0:
            overlay = pygame.Surface(s.get_size(), pygame.SRCALPHA)
            overlay.fill((200, 40, 40, int(120 * min(1.0, flash))))
            s.blit(overlay, (0, 0))

        for b in game.bricks:
            pygame.draw.rect(s, row_colour(b.row, cfg.brick_rows), pygame.Rect(b.x, b.y, b.w, b.h), border_radius=3)
        pygame.draw.rect(
            s, FG, pygame.Rect(game.paddle_x, game.paddle_top, cfg.paddle_width, cfg.paddle_height), border_radius=4
        )
        r = cfg.ball_half
        pygame.draw.rect(s, ACCENT, pygame.Rect(game.ball.x - r, game.ball.y - r, 2 * r, 2 * r))

        self._text(f"SCORE {game.score}", (16, 16))
        self._text(f"LIVES {game.lives}", (cfg.width - 100, 16))

        mid = (cfg.width / 2, cfg.height * 0.62)
        if game.paused:
            self._text("PAUSED", mid, ACCENT, self.big, centre=True)
            self._text(f"{binding_hint(action_map, Action.PAUSE)} to resume", (mid[0], mid[1] + 36), DIM, centre=True)
        elif game.phase is Phase.SERVING:
            self._text(f"{binding_hint(action_map, Action.LAUNCH)} to launch", mid, DIM, centre=True)
        elif game.phase in (Phase.WON, Phase.LOST):
            title = "CLEARED!" if game.phase is Phase.WON else "GAME OVER"
            self._text(title, mid, ACCENT, self.big, centre=True)
            self._text(f"{binding_hint(action_map, Action.LAUNCH)} to play again", (mid[0], mid[1] + 36), DIM, centre=True)

        footer = (
            f"Move {binding_hint(action_map, Action.MOVE_LEFT, 1)}/{binding_hint(action_map, Action.MOVE_RIGHT, 1)}"
            f"   Pause {binding_hint(action_map, Action.PAUSE, 1)}"
            f"   Controls {binding_hint(action_map, Action.OPEN_BINDINGS, 1)}"
        )
        self._text(footer, (cfg.width / 2, cfg.height - 12), DIM, centre=True)

    def draw_menu(self, menu: BindingsMenu, pads: list[str]) -> None:
        s = self.surface
        w, h = s.get_size()
        s.fill(BG)
        self._text("CONTROLS", (w / 2, 40), ACCENT, self.big, centre=True)
        pad_line = "Gamepad: " + (", ".join(pads) if pads else "none connected")
        self._text(pad_line, (w / 2, 76), DIM, centre=True)

        y = 104
        for row in menu.rows():
            if row.selected:
                pygame.draw.rect(s, PANEL, pygame.Rect(24, y - 4, w - 48, 28), border_radius=4)
            colour = ACCENT if row.selected else FG
            self._text(row.label, (40, y), colour)
            self._text(row.bindings, (240, y), DIM if not row.selected else FG)
            y += 32

        if menu.capture is not None:
            self._text(f"Waiting for input... {menu.capture.seconds_left:.0f}s", (w / 2, h - 64), ACCENT, centre=True)
        self._text(menu.message, (w / 2, h - 36), FG, centre=True)
