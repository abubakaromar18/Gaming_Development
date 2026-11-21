#include "platform/Renderer.hpp"

#include <algorithm>
#include <cmath>
#include <cstdio>

#include "platform/Sprites.hpp"

namespace invaders {

namespace {

constexpr float kGlyphAdvance = 4.0f;

}  // namespace

void Renderer::setColor(Uint8 r, Uint8 g, Uint8 b, Uint8 a) { SDL_SetRenderDrawColor(sdl_, r, g, b, a); }

void Renderer::fillRect(float x, float y, float w, float h) {
    const SDL_FRect rect{x, y, w, h};
    SDL_RenderFillRectF(sdl_, &rect);
}

template <std::size_t Rows>
void Renderer::drawBitmap(const std::array<std::string_view, Rows>& bitmap, float x, float y) {
    for (std::size_t row = 0; row < Rows; ++row) {
        const std::string_view line = bitmap[row];
        for (std::size_t col = 0; col < line.size(); ++col) {
            if (line[col] == 'X') fillRect(x + static_cast<float>(col), y + static_cast<float>(row), 1.0f, 1.0f);
        }
    }
}

void Renderer::drawText(std::string_view text, float x, float y) {
    for (char c : text) {
        drawBitmap(sprites::glyph(c), x, y);
        x += kGlyphAdvance;
    }
}

void Renderer::drawNumber(int value, float x, float y) {
    char buffer[16];
    std::snprintf(buffer, sizeof buffer, "%d", value);
    drawText(buffer, x, y);
}

void Renderer::drawPoolBar(std::string_view label, std::size_t size, std::size_t highWater,
                           std::size_t capacity, std::size_t exhausted, float y) {
    constexpr float kBarX = 18.0f;
    constexpr float kBarWidth = 60.0f;
    const auto fraction = [capacity](std::size_t n) {
        return static_cast<float>(n) / static_cast<float>(capacity);
    };

    setColor(150, 150, 170);
    drawText(label, 4.0f, y);
    setColor(40, 40, 60);
    fillRect(kBarX, y, kBarWidth, 5.0f);
    // Bar colour warns as the pool fills up.
    const float used = fraction(size);
    if (used < 0.75f) setColor(80, 200, 120); else setColor(230, 90, 70);
    fillRect(kBarX, y, kBarWidth * used, 5.0f);
    // High-water mark: the number that tells you how big the pool must be.
    setColor(255, 230, 90);
    fillRect(kBarX + kBarWidth * fraction(highWater) - 1.0f, y, 1.0f, 5.0f);

    char buffer[32];
    std::snprintf(buffer, sizeof buffer, "%zu/%zu", size, capacity);
    setColor(150, 150, 170);
    drawText(buffer, kBarX + kBarWidth + 4.0f, y);
    if (exhausted > 0) {
        setColor(230, 90, 70);
        std::snprintf(buffer, sizeof buffer, "X%zu", exhausted);
        drawText(buffer, kBarX + kBarWidth + 40.0f, y);
    }
}

void Renderer::draw(const Game& game, const RenderOptions& options) {
    const GameConfig& cfg = game.config();

    setColor(8, 8, 20);
    SDL_RenderClear(sdl_);

    // Ground line.
    setColor(60, 200, 90);
    fillRect(0.0f, cfg.playerY + cfg.playerHeight * 0.5f + 3.0f, cfg.worldWidth, 1.0f);

    // Bunkers, one rectangle per surviving cell.
    for (const Bunker* bunker = game.bunkersBegin(); bunker != game.bunkersEnd(); ++bunker) {
        for (int row = 0; row < Bunker::kRows; ++row) {
            for (int col = 0; col < Bunker::kCols; ++col) {
                if (!bunker->cellAlive(col, row)) continue;
                const Rect r = bunker->cellRect(col, row);
                fillRect(r.x, r.y, r.w, r.h);
            }
        }
    }

    // Invaders: the sprite depends on row, the frame on the march.
    const int frame = game.marchFrame();
    game.invaders().forEach([&](const Invader& inv) {
        const Rect r = game.invaderRect(inv);
        if (inv.row == 0) {
            setColor(240, 120, 220);
            drawBitmap(sprites::kSquid[frame], r.x, r.y);
        } else if (inv.row <= 2) {
            setColor(120, 200, 255);
            drawBitmap(sprites::kCrab[frame], r.x, r.y);
        } else {
            setColor(255, 240, 140);
            drawBitmap(sprites::kBlob[frame], r.x, r.y);
        }
    });

    // Bullets.
    setColor(255, 255, 255);
    game.playerBullets().forEach([&](const Bullet& b) {
        const Rect r = game.bulletRect(b);
        fillRect(r.x, r.y, r.w, r.h);
    });
    setColor(255, 140, 80);
    game.enemyBullets().forEach([&](const Bullet& b) {
        const Rect r = game.bulletRect(b);
        // A wiggle makes enemy shots readable at a glance.
        const float wiggle = std::fmod(b.pos.y, 4.0f) < 2.0f ? -1.0f : 1.0f;
        fillRect(r.x + wiggle * 0.5f, r.y, r.w + 1.0f, r.h);
    });

    // Explosion debris fades out over its lifetime.
    game.particles().forEach([&](const Particle& p) {
        const float life = 1.0f - std::clamp(p.age / p.lifetime, 0.0f, 1.0f);
        setColor(255, static_cast<Uint8>(120 + 135 * life), 80, static_cast<Uint8>(255 * life));
        fillRect(p.pos.x, p.pos.y, 1.0f, 1.0f);
    });

    // The cannon blinks while its respawn shield is up.
    const bool blinkOff = game.playerInvulnerable() && std::fmod(options.timeSeconds, 0.2f) < 0.1f;
    if (game.phase() != Phase::GameOver && !blinkOff) {
        const Rect p = game.playerRect();
        setColor(60, 200, 90);
        drawBitmap(sprites::kCannon, p.x, p.y);
    }

    // HUD.
    setColor(230, 230, 240);
    drawText("SCORE", 4.0f, 3.0f);
    drawNumber(game.score(), 26.0f, 3.0f);
    drawText("WAVE", 90.0f, 3.0f);
    drawNumber(game.wave(), 108.0f, 3.0f);
    drawText("LIVES", 160.0f, 3.0f);
    drawNumber(game.lives(), 182.0f, 3.0f);

    if (options.showPoolOverlay) {
        drawPoolBar("PB", game.playerBullets().size(), game.playerBullets().highWater(),
                    game.playerBullets().capacity(), game.playerBullets().exhaustedCount(), 11.0f);
        drawPoolBar("EB", game.enemyBullets().size(), game.enemyBullets().highWater(),
                    game.enemyBullets().capacity(), game.enemyBullets().exhaustedCount(), 17.0f);
        drawPoolBar("IN", game.invaders().size(), game.invaders().highWater(),
                    game.invaders().capacity(), game.invaders().exhaustedCount(), 23.0f);
        drawPoolBar("FX", game.particles().size(), game.particles().highWater(),
                    game.particles().capacity(), game.particles().exhaustedCount(), 29.0f);
    }

    if (game.phase() == Phase::WaveClear) {
        setColor(255, 240, 140);
        drawText("WAVE CLEAR", cfg.worldWidth * 0.5f - 20.0f, cfg.worldHeight * 0.5f);
    } else if (game.phase() == Phase::GameOver) {
        setColor(230, 90, 70);
        drawText("GAME OVER", cfg.worldWidth * 0.5f - 18.0f, cfg.worldHeight * 0.5f - 6.0f);
        if (std::fmod(options.timeSeconds, 1.0f) < 0.6f) {
            setColor(230, 230, 240);
            drawText("PRESS R", cfg.worldWidth * 0.5f - 14.0f, cfg.worldHeight * 0.5f + 4.0f);
        }
    }

    if (options.hitFlash > 0.0f) {
        setColor(255, 40, 40, static_cast<Uint8>(90.0f * options.hitFlash));
        fillRect(0.0f, 0.0f, cfg.worldWidth, cfg.worldHeight);
    }

    SDL_RenderPresent(sdl_);
}

}  // namespace invaders
