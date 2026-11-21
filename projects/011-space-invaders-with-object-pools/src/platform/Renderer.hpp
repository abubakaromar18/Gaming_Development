#pragma once

// Draws a Game with SDL2. Reads core state only; never mutates it.

#include <SDL.h>

#include <array>
#include <cstddef>
#include <string_view>

#include "core/Game.hpp"

namespace invaders {

struct RenderOptions {
    bool showPoolOverlay = true;
    float hitFlash = 0.0f;  // 0..1, red tint after the player is hit
    float timeSeconds = 0.0f;  // drives blinking text; supplied by the shell
};

class Renderer {
public:
    explicit Renderer(SDL_Renderer* sdl) : sdl_(sdl) {}

    void draw(const Game& game, const RenderOptions& options);

private:
    template <std::size_t Rows>
    void drawBitmap(const std::array<std::string_view, Rows>& bitmap, float x, float y);
    void drawText(std::string_view text, float x, float y);
    void drawNumber(int value, float x, float y);
    void drawPoolBar(std::string_view label, std::size_t size, std::size_t highWater,
                     std::size_t capacity, std::size_t exhausted, float y);
    void setColor(Uint8 r, Uint8 g, Uint8 b, Uint8 a = 255);
    void fillRect(float x, float y, float w, float h);

    SDL_Renderer* sdl_;
};

}  // namespace invaders
