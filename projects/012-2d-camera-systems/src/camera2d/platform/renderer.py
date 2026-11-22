"""Procedural pygame drawing. Reads core state; never mutates it.

Everything in the world layer goes through ``Camera.world_to_screen`` and is
drawn as polygons, so shake rotation applies to rocks and the grid as well as
their positions. Background stars use ``parallax_scroll`` instead.
"""
from __future__ import annotations

import math

import pygame

from ..core import FEATURES, STAR_TILE, Camera, Rect, Vec2, World

BG = (12, 14, 24)
GRID = (28, 32, 50)
BOUNDS = (90, 110, 170)
ROCK_FILL = (70, 64, 84)
ROCK_EDGE = (130, 120, 150)
SHIP = (255, 210, 90)
RING = (255, 140, 80)
DEADZONE = (80, 220, 160)
GOAL = (240, 90, 200)
FOCUS = (90, 180, 255)
TEXT = (220, 225, 240)
DIM = (110, 115, 135)
GRID_STEP = 200.0


class Renderer:
    def __init__(self, surface: pygame.Surface) -> None:
        self.surface = surface
        self.font = pygame.font.Font(None, 22)

    def draw(self, world: World, camera: Camera, show_debug: bool, fps: float) -> None:
        self.surface.fill(BG)
        self._stars(world, camera)
        self._grid(camera)
        self._rect_outline(camera, world.bounds, BOUNDS, 3)
        # Cull against the logical view, padded so shake can't pop rocks in at the edge.
        pad = camera.config.shake_max_offset + 64
        view = camera.view_rect()
        visible = Rect(view.left - pad, view.top - pad, view.width + 2 * pad, view.height + 2 * pad)
        for rock in world.rocks:
            if rock.intersects(visible):
                pts = self._corners(camera, rock)
                pygame.draw.polygon(self.surface, ROCK_FILL, pts)
                pygame.draw.polygon(self.surface, ROCK_EDGE, pts, 2)
        for ring in world.rings:
            t = ring.age / world.config.shockwave_lifetime
            radius = int((40 + 260 * t) * camera.zoom)
            center = camera.world_to_screen(ring.position)
            width = max(1, int(6 * (1 - t)))
            pygame.draw.circle(self.surface, RING, (center.x, center.y), radius, width)
        self._ship(world, camera)
        if show_debug:
            self._debug(camera)
        self._hud(camera, show_debug, fps)

    def _stars(self, world: World, camera: Camera) -> None:
        w, h = self.surface.get_size()
        for star in world.stars:
            scroll = camera.parallax_scroll(star.depth)
            # Wrap into one tile, then repeat the tile across the screen.
            x0 = (star.x - scroll.x) % STAR_TILE
            y0 = (star.y - scroll.y) % STAR_TILE
            shade = int(80 + 170 * star.depth)
            size = 1 if star.depth < 0.3 else 2
            x = x0
            while x < w:
                y = y0
                while y < h:
                    self.surface.fill((shade, shade, min(255, shade + 30)), (int(x), int(y), size, size))
                    y += STAR_TILE
                x += STAR_TILE

    def _grid(self, camera: Camera) -> None:
        view = camera.view_rect()
        margin = GRID_STEP * 2
        x = math.floor((view.left - margin) / GRID_STEP) * GRID_STEP
        while x <= view.right + margin:
            a = camera.world_to_screen(Vec2(x, view.top - margin))
            b = camera.world_to_screen(Vec2(x, view.bottom + margin))
            pygame.draw.line(self.surface, GRID, (a.x, a.y), (b.x, b.y))
            x += GRID_STEP
        y = math.floor((view.top - margin) / GRID_STEP) * GRID_STEP
        while y <= view.bottom + margin:
            a = camera.world_to_screen(Vec2(view.left - margin, y))
            b = camera.world_to_screen(Vec2(view.right + margin, y))
            pygame.draw.line(self.surface, GRID, (a.x, a.y), (b.x, b.y))
            y += GRID_STEP

    def _ship(self, world: World, camera: Camera) -> None:
        v = world.velocity
        heading = math.atan2(v.y, v.x) if v.length() > 1.0 else -math.pi / 2
        s = world.config.ship_size * 0.7
        local = [Vec2(s, 0), Vec2(-s * 0.7, s * 0.6), Vec2(-s * 0.35, 0), Vec2(-s * 0.7, -s * 0.6)]
        pts = [camera.world_to_screen(world.position + p.rotate(heading)) for p in local]
        pygame.draw.polygon(self.surface, SHIP, [(p.x, p.y) for p in pts])

    def _debug(self, camera: Camera) -> None:
        dbg = camera.debug
        if dbg.deadzone is not None:
            self._rect_outline(camera, dbg.deadzone, DEADZONE, 1, shaken=False)
        g = camera.world_to_screen(dbg.goal, shaken=False)
        pygame.draw.line(self.surface, GOAL, (g.x - 8, g.y), (g.x + 8, g.y), 2)
        pygame.draw.line(self.surface, GOAL, (g.x, g.y - 8), (g.x, g.y + 8), 2)
        f = camera.world_to_screen(dbg.focus, shaken=False)
        pygame.draw.circle(self.surface, FOCUS, (f.x, f.y), 5, 2)
        c = camera.viewport * 0.5
        pygame.draw.circle(self.surface, TEXT, (c.x, c.y), 2)

    def _hud(self, camera: Camera, show_debug: bool, fps: float) -> None:
        y = 10
        for i, (field, label) in enumerate(FEATURES.items(), start=1):
            on = getattr(camera.config, field)
            self._text(f"{i}  {label}: {'ON' if on else 'off'}", (10, y), TEXT if on else DIM)
            y += 20
        self._text(f"Tab  Debug overlay: {'ON' if show_debug else 'off'}", (10, y), TEXT if show_debug else DIM)
        y += 28
        self._text(f"Trauma {camera.trauma:.2f}  Shake {camera.shake:.2f}", (10, y), TEXT)
        pygame.draw.rect(self.surface, DIM, (10, y + 20, 160, 8), 1)
        pygame.draw.rect(self.surface, RING, (10, y + 20, int(160 * camera.trauma), 8))
        w, h = self.surface.get_size()
        self._text(f"zoom {camera.zoom:.2f}   {fps:4.0f} fps", (w - 170, 10), DIM)
        self._text(
            "WASD/arrows fly · Space shockwave · T trauma · wheel zoom · R reset · Esc quit", (10, h - 24), DIM
        )

    def _text(self, text: str, pos: tuple[int, int], color: tuple[int, int, int]) -> None:
        self.surface.blit(self.font.render(text, True, color), pos)

    @staticmethod
    def _corners(camera: Camera, r: Rect, shaken: bool = True) -> list[tuple[float, float]]:
        corners = (Vec2(r.left, r.top), Vec2(r.right, r.top), Vec2(r.right, r.bottom), Vec2(r.left, r.bottom))
        return [(p.x, p.y) for p in (camera.world_to_screen(c, shaken) for c in corners)]

    def _rect_outline(
        self, camera: Camera, r: Rect, color: tuple[int, int, int], width: int, shaken: bool = True
    ) -> None:
        pygame.draw.polygon(self.surface, color, self._corners(camera, r, shaken), width)
