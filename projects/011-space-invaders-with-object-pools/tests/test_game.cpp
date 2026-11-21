#include <catch2/catch_test_macros.hpp>

#include <algorithm>
#include <limits>
#include <stdexcept>

#include "TestSupport.hpp"
#include "core/Game.hpp"

using namespace invaders;
using invaders::test::kStep;
using invaders::test::sawEvent;
using invaders::test::stepUntil;

TEST_CASE("a new game starts wave 1 with a full formation", "[game]") {
    const Game game(GameConfig{}, Rng(1));
    CHECK(game.phase() == Phase::Playing);
    CHECK(game.wave() == 1);
    CHECK(game.lives() == 3);
    CHECK(game.score() == 0);
    CHECK(game.invaders().size() == 55);
    CHECK(game.bunkersEnd() - game.bunkersBegin() == 4);
}

TEST_CASE("the cannon moves with input and stops at the walls", "[game][player]") {
    Game game(test::quietConfig(), Rng(1));
    const float start = game.playerX();
    game.step(Input{1, false}, 0.5f);
    CHECK(game.playerX() > start);

    for (int i = 0; i < 600; ++i) game.step(Input{-1, false}, kStep);
    CHECK(game.playerRect().x == 0.0f);
}

TEST_CASE("the player bullet pool caps shots in flight", "[game][pool]") {
    GameConfig config = test::quietConfig();
    config.playerFireCooldown = 0.0f;  // only the pool limits fire
    Game game(config, Rng(1));

    std::size_t maxInFlight = 0;
    for (int i = 0; i < 60; ++i) {
        game.step(Input{0, true}, kStep);
        maxInFlight = std::max(maxInFlight, game.playerBullets().size());
    }
    CHECK(maxInFlight == pool_capacity::kPlayerBullets);
    CHECK(game.playerBullets().exhaustedCount() > 0);
}

TEST_CASE("the fire cooldown spaces shots out", "[game][player]") {
    Game game(test::quietConfig(), Rng(1));
    game.step(Input{0, true}, kStep);
    REQUIRE(game.playerBullets().size() == 1);
    REQUIRE(sawEvent(game, EventType::PlayerFired));
    game.step(Input{0, true}, kStep);
    CHECK(game.playerBullets().size() == 1);
}

TEST_CASE("a shot kills the invader above, scores, and recycles both objects", "[game]") {
    Game game(test::quietConfig(), Rng(1));
    game.step(Input{0, true}, kStep);

    bool killed = false;
    REQUIRE(stepUntil(game, Input{}, 2.0f, [&](const Game& g) {
        killed = killed || sawEvent(g, EventType::InvaderKilled);
        return killed;
    }));
    CHECK(game.score() == 10);  // bottom row is worth 10
    CHECK(game.invaders().size() == 54);
    CHECK(game.playerBullets().empty());
    CHECK(game.particles().size() == static_cast<std::size_t>(game.config().particlesPerExplosion));
}

TEST_CASE("explosion debris expires and returns to the pool", "[game][pool]") {
    Game game(test::loneSittingInvader(), Rng(1));
    game.step(Input{0, true}, kStep);
    REQUIRE(stepUntil(game, Input{}, 2.0f, [](const Game& g) { return !g.particles().empty(); }));
    REQUIRE(stepUntil(game, Input{}, 2.0f, [](const Game& g) { return g.particles().empty(); }));
}

TEST_CASE("the formation marches, drops at a wall, and reverses", "[game][formation]") {
    Game game(test::quietConfig(), Rng(1));
    const Vec2 start = game.formationOrigin();

    REQUIRE(stepUntil(game, Input{}, 2.0f, [&](const Game& g) { return g.formationOrigin().x > start.x; }));
    CHECK(game.formationOrigin().y == start.y);

    REQUIRE(stepUntil(game, Input{}, 30.0f, [&](const Game& g) { return g.formationOrigin().y > start.y; }));
    const float xAtDrop = game.formationOrigin().x;
    CHECK(game.formationOrigin().y == start.y + game.config().marchDropY);

    REQUIRE(stepUntil(game, Input{}, 2.0f, [&](const Game& g) { return g.formationOrigin().x != xAtDrop; }));
    CHECK(game.formationOrigin().x < xAtDrop);  // now heading left
}

TEST_CASE("the march alternates the animation frame", "[game][formation]") {
    Game game(test::quietConfig(), Rng(1));
    const int frame = game.marchFrame();
    REQUIRE(stepUntil(game, Input{}, 1.0f, [&](const Game& g) { return g.marchFrame() != frame; }));
}

TEST_CASE("fewer invaders march faster", "[game][formation]") {
    auto stepsInOneSecond = [](const GameConfig& config) {
        Game game(config, Rng(1));
        int steps = 0;
        int frame = game.marchFrame();
        for (int i = 0; i < 120; ++i) {
            game.step(Input{}, kStep);
            if (game.marchFrame() != frame) ++steps;
            frame = game.marchFrame();
        }
        return steps;
    };
    GameConfig lone = test::quietConfig();
    lone.invaderRows = 1;
    lone.invaderCols = 1;
    CHECK(stepsInOneSecond(lone) > 10 * stepsInOneSecond(test::quietConfig()));
}

TEST_CASE("clearing the board starts the next wave lower down", "[game][wave]") {
    Game game(test::loneSittingInvader(), Rng(1));
    const float wave1Y = game.formationOrigin().y;
    game.step(Input{0, true}, kStep);

    REQUIRE(stepUntil(game, Input{}, 2.0f, [](const Game& g) { return g.phase() == Phase::WaveClear; }));
    CHECK(sawEvent(game, EventType::WaveCleared));

    REQUIRE(stepUntil(game, Input{}, 3.0f, [](const Game& g) { return g.phase() == Phase::Playing; }));
    CHECK(game.wave() == 2);
    CHECK(game.invaders().size() == 1);
    CHECK(game.formationOrigin().y == wave1Y + game.config().waveDropPerWave);
    CHECK(game.score() == 30);  // a single-row formation is the top row
}

TEST_CASE("enemy fire costs lives, grants a shield, and ends the game", "[game][player]") {
    GameConfig config = test::loneSittingInvader();
    config.enemyShotsPerSecond = 30.0f;
    Game game(config, Rng(7));

    REQUIRE(stepUntil(game, Input{}, 10.0f, [](const Game& g) { return g.lives() == 2; }));
    CHECK(sawEvent(game, EventType::PlayerHit));
    CHECK(game.playerInvulnerable());

    // During the shield, shots are absorbed and no life is lost.
    const bool lostLifeDuringShield =
        stepUntil(game, Input{}, config.respawnInvulnerability * 0.9f, [](const Game& g) { return g.lives() < 2; });
    CHECK_FALSE(lostLifeDuringShield);

    REQUIRE(stepUntil(game, Input{}, 20.0f, [](const Game& g) { return g.phase() == Phase::GameOver; }));
    CHECK(game.lives() == 0);
}

TEST_CASE("enemy bullets in flight never exceed their pool", "[game][pool]") {
    GameConfig config = test::quietConfig();
    config.enemyShotsPerSecond = 120.0f;
    Game game(config, Rng(3));
    std::size_t peak = 0;
    for (int i = 0; i < 240; ++i) {
        game.step(Input{}, kStep);
        peak = std::max(peak, game.enemyBullets().size());
    }
    CHECK(peak == pool_capacity::kEnemyBullets);
    CHECK(game.enemyBullets().exhaustedCount() > 0);
}

TEST_CASE("invaders that reach the cannon end the game", "[game][formation]") {
    GameConfig config = test::quietConfig();
    config.marchIntervalFull = 0.01f;
    config.marchIntervalLast = 0.01f;
    Game game(config, Rng(1));
    REQUIRE(stepUntil(game, Input{}, 60.0f, [](const Game& g) { return g.phase() == Phase::GameOver; }));
    CHECK(sawEvent(game, EventType::GameOver));
    CHECK(game.lives() == 0);
}

TEST_CASE("bunkers absorb the player's shots", "[game][bunker]") {
    GameConfig config = test::quietConfig();
    config.bunkerCount = 4;
    Game game(config, Rng(1));
    const Bunker& first = *game.bunkersBegin();
    const Rect box = first.bounds();
    const float target = box.x + box.w * 0.25f;

    REQUIRE(stepUntil(game, Input{-1, false}, 3.0f, [&](const Game& g) { return g.playerX() <= target; }));
    const int cellsBefore = first.liveCells();
    game.step(Input{0, true}, kStep);

    bool blocked = false;
    REQUIRE(stepUntil(game, Input{}, 1.0f, [&](const Game& g) {
        blocked = blocked || sawEvent(g, EventType::BunkerHit);
        return blocked;
    }));
    CHECK(first.liveCells() == cellsBefore - 1);
    CHECK(game.score() == 0);
}

TEST_CASE("player and enemy shots cancel out in mid-air", "[game]") {
    GameConfig config = test::loneSittingInvader();
    config.enemyShotsPerSecond = 1000.0f;  // fires on the first tick
    Game game(config, Rng(5));
    game.step(Input{}, kStep);
    REQUIRE(game.enemyBullets().size() == 1);

    // The lone invader sits right above the cannon, so the shots are aligned.
    game.step(Input{0, true}, kStep);
    bool collided = false;
    REQUIRE(stepUntil(game, Input{}, 2.0f, [&](const Game& g) {
        collided = collided || sawEvent(g, EventType::BulletsCollided);
        return collided;
    }));
    CHECK(game.playerBullets().empty());
    CHECK(game.score() == 0);
}

TEST_CASE("the same seed and inputs replay to the same state", "[game][determinism]") {
    auto play = [](std::uint64_t seed) {
        Game game(GameConfig{}, Rng(seed));
        Rng bot(99);  // scripted pseudo-player, independent of the game RNG
        Input input;
        for (int tick = 0; tick < 120 * 30; ++tick) {
            if (tick % 20 == 0) input = Input{static_cast<int>(bot.below(3)) - 1, bot.below(2) == 0};
            game.step(input, kStep);
        }
        return game.stateHash();
    };
    CHECK(play(42) == play(42));
    CHECK(play(42) != play(43));
}

TEST_CASE("restart resets score, lives and wave", "[game]") {
    Game game(test::loneSittingInvader(), Rng(1));
    game.step(Input{0, true}, kStep);
    REQUIRE(stepUntil(game, Input{}, 5.0f, [](const Game& g) { return g.wave() == 2; }));
    game.restart();
    CHECK(game.score() == 0);
    CHECK(game.wave() == 1);
    CHECK(game.lives() == game.config().startingLives);
    CHECK(game.phase() == Phase::Playing);
}

TEST_CASE("negative or NaN time steps are rejected", "[game]") {
    Game game(GameConfig{}, Rng(1));
    CHECK_THROWS_AS(game.step(Input{}, -0.1f), std::invalid_argument);
    CHECK_THROWS_AS(game.step(Input{}, std::numeric_limits<float>::quiet_NaN()), std::invalid_argument);
}
