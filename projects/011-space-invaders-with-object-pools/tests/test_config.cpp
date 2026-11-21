#include <catch2/catch_test_macros.hpp>
#include <catch2/matchers/catch_matchers_string.hpp>

#include <stdexcept>

#include "core/Config.hpp"
#include "core/Game.hpp"

using invaders::GameConfig;
using Catch::Matchers::ContainsSubstring;

TEST_CASE("the default config is valid", "[config]") { CHECK_NOTHROW(GameConfig{}.validate()); }

TEST_CASE("a formation larger than the invader pool is rejected", "[config]") {
    GameConfig config;
    config.invaderRows = 7;
    config.invaderCols = 11;
    CHECK_THROWS_WITH(config.validate(), ContainsSubstring("kInvaders"));
}

TEST_CASE("invalid values name the offending field", "[config]") {
    GameConfig config;
    config.playerSpeed = -1.0f;
    CHECK_THROWS_WITH(config.validate(), ContainsSubstring("playerSpeed"));

    config = GameConfig{};
    config.marchIntervalLast = 1.0f;  // slower than a full board
    CHECK_THROWS_WITH(config.validate(), ContainsSubstring("marchInterval"));

    config = GameConfig{};
    config.invaderRows = 1;
    config.invaderCols = 14;  // fits the pool but not the screen
    CHECK_THROWS_WITH(config.validate(), ContainsSubstring("wider than the world"));
}

TEST_CASE("the game refuses an invalid config", "[config]") {
    GameConfig config;
    config.startingLives = 0;
    CHECK_THROWS_AS(invaders::Game(config, invaders::Rng(1)), std::invalid_argument);
}
