#pragma once

#include <algorithm>
#include <functional>

#include "core/Game.hpp"

namespace invaders::test {

inline constexpr float kStep = 1.0f / 120.0f;

// A calm battlefield: no enemy fire and no bunkers, so a test controls
// exactly what happens.
inline GameConfig quietConfig() {
    GameConfig config;
    config.enemyShotsPerSecond = 0.0f;
    config.bunkerCount = 0;
    return config;
}

// One invader that never moves, parked right above the cannon's start.
inline GameConfig loneSittingInvader() {
    GameConfig config = quietConfig();
    config.invaderRows = 1;
    config.invaderCols = 1;
    config.marchIntervalFull = 1000.0f;
    config.marchIntervalLast = 1000.0f;
    return config;
}

// Steps until pred(game) is true or maxSeconds of game time pass.
// Returns true if the predicate was met.
inline bool stepUntil(Game& game, const Input& input, float maxSeconds, const std::function<bool(const Game&)>& pred) {
    for (float t = 0.0f; t < maxSeconds; t += kStep) {
        game.step(input, kStep);
        if (pred(game)) return true;
    }
    return false;
}

inline bool sawEvent(const Game& game, EventType type) {
    return std::any_of(game.events().begin(), game.events().end(), [type](const Event& e) { return e.type == type; });
}

}  // namespace invaders::test
