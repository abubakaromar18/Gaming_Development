// SDL2 shell: window, keyboard, fixed-timestep loop. All game rules live in
// src/core; this file only translates keys into Input, ticks the simulation
// at a constant rate, and hands the result to the Renderer.

#include <SDL.h>

#include <chrono>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <exception>
#include <stdexcept>
#include <string_view>

#include "core/Game.hpp"
#include "platform/Renderer.hpp"

namespace {

constexpr double kStepSeconds = 1.0 / 120.0;
constexpr double kMaxFrameSeconds = 0.25;  // avoids the spiral of death after a stall
constexpr int kWindowScale = 3;
constexpr float kHitFlashSeconds = 0.35f;

struct Options {
    std::uint64_t seed = 0;
    bool seedGiven = false;
};

Options parseArgs(int argc, char** argv) {
    Options options;
    for (int i = 1; i < argc; ++i) {
        const std::string_view arg = argv[i];
        if (arg == "--seed" && i + 1 < argc) {
            char* end = nullptr;
            options.seed = std::strtoull(argv[++i], &end, 10);
            if (end == nullptr || *end != '\0') throw std::invalid_argument("--seed expects an integer");
            options.seedGiven = true;
        } else {
            throw std::invalid_argument("usage: invaders [--seed N]");
        }
    }
    return options;
}

invaders::Input readInput() {
    const Uint8* keys = SDL_GetKeyboardState(nullptr);
    invaders::Input input;
    if (keys[SDL_SCANCODE_LEFT] || keys[SDL_SCANCODE_A]) input.move -= 1;
    if (keys[SDL_SCANCODE_RIGHT] || keys[SDL_SCANCODE_D]) input.move += 1;
    input.fire = keys[SDL_SCANCODE_SPACE] || keys[SDL_SCANCODE_UP] || keys[SDL_SCANCODE_W];
    return input;
}

int run(const Options& options) {
    // The shell is the one place allowed to look at the wall clock, and only
    // to choose a seed when the player didn't supply one.
    const std::uint64_t seed =
        options.seedGiven ? options.seed
                          : static_cast<std::uint64_t>(std::chrono::steady_clock::now().time_since_epoch().count());
    std::printf("invaders: seed %llu\n", static_cast<unsigned long long>(seed));

    const invaders::GameConfig config;
    invaders::Game game(config, invaders::Rng(seed));

    if (SDL_Init(SDL_INIT_VIDEO) != 0) {
        std::fprintf(stderr, "SDL_Init failed: %s\n", SDL_GetError());
        return EXIT_FAILURE;
    }
    SDL_Window* window = SDL_CreateWindow("Invaders - object pools", SDL_WINDOWPOS_CENTERED, SDL_WINDOWPOS_CENTERED,
                                          static_cast<int>(config.worldWidth) * kWindowScale,
                                          static_cast<int>(config.worldHeight) * kWindowScale, SDL_WINDOW_RESIZABLE);
    SDL_Renderer* sdlRenderer =
        window ? SDL_CreateRenderer(window, -1, SDL_RENDERER_ACCELERATED | SDL_RENDERER_PRESENTVSYNC) : nullptr;
    if (!window || !sdlRenderer) {
        std::fprintf(stderr, "SDL window/renderer failed: %s\n", SDL_GetError());
        if (window) SDL_DestroyWindow(window);
        SDL_Quit();
        return EXIT_FAILURE;
    }
    // Draw in world units; SDL scales and letterboxes to the window.
    SDL_RenderSetLogicalSize(sdlRenderer, static_cast<int>(config.worldWidth), static_cast<int>(config.worldHeight));
    SDL_SetRenderDrawBlendMode(sdlRenderer, SDL_BLENDMODE_BLEND);

    invaders::Renderer renderer(sdlRenderer);
    invaders::RenderOptions renderOptions;
    float hitFlashTimer = 0.0f;
    double accumulator = 0.0;
    double elapsed = 0.0;
    Uint64 previous = SDL_GetPerformanceCounter();
    const double ticksPerSecond = static_cast<double>(SDL_GetPerformanceFrequency());

    bool running = true;
    while (running) {
        SDL_Event event;
        while (SDL_PollEvent(&event)) {
            if (event.type == SDL_QUIT) running = false;
            if (event.type == SDL_KEYDOWN && !event.key.repeat) {
                switch (event.key.keysym.scancode) {
                    case SDL_SCANCODE_ESCAPE: running = false; break;
                    case SDL_SCANCODE_F1: renderOptions.showPoolOverlay = !renderOptions.showPoolOverlay; break;
                    case SDL_SCANCODE_R:
                        if (game.phase() == invaders::Phase::GameOver) game.restart();
                        break;
                    default: break;
                }
            }
        }

        const Uint64 now = SDL_GetPerformanceCounter();
        double frameSeconds = static_cast<double>(now - previous) / ticksPerSecond;
        previous = now;
        if (frameSeconds > kMaxFrameSeconds) frameSeconds = kMaxFrameSeconds;
        accumulator += frameSeconds;
        elapsed += frameSeconds;

        const invaders::Input input = readInput();
        while (accumulator >= kStepSeconds) {
            game.step(input, static_cast<float>(kStepSeconds));
            for (const invaders::Event& e : game.events()) {
                if (e.type == invaders::EventType::PlayerHit) hitFlashTimer = kHitFlashSeconds;
            }
            accumulator -= kStepSeconds;
        }

        hitFlashTimer = hitFlashTimer > static_cast<float>(frameSeconds) ? hitFlashTimer - static_cast<float>(frameSeconds) : 0.0f;
        renderOptions.hitFlash = hitFlashTimer / kHitFlashSeconds;
        renderOptions.timeSeconds = static_cast<float>(elapsed);
        renderer.draw(game, renderOptions);
    }

    SDL_DestroyRenderer(sdlRenderer);
    SDL_DestroyWindow(window);
    SDL_Quit();
    return EXIT_SUCCESS;
}

}  // namespace

int main(int argc, char** argv) {
    try {
        return run(parseArgs(argc, argv));
    } catch (const std::exception& e) {
        std::fprintf(stderr, "invaders: %s\n", e.what());
        return EXIT_FAILURE;
    }
}
