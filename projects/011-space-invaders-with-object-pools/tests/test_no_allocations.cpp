// Proves the central claim of the project: once a Game is constructed, the
// simulation never calls the heap allocator, however much is spawned and
// destroyed. Global operator new is replaced in this binary to count calls.

#include <catch2/catch_test_macros.hpp>

#include <atomic>
#include <algorithm>
#include <cstdlib>
#include <memory>
#include <new>

#include "TestSupport.hpp"
#include "core/Game.hpp"

namespace {
std::atomic<long> gAllocations{0};
}  // namespace

void* operator new(std::size_t size) {
    ++gAllocations;
    if (void* p = std::malloc(size ? size : 1)) return p;
    throw std::bad_alloc();
}
void* operator new[](std::size_t size) { return ::operator new(size); }
void operator delete(void* p) noexcept { std::free(p); }
void operator delete[](void* p) noexcept { std::free(p); }
void operator delete(void* p, std::size_t) noexcept { std::free(p); }
void operator delete[](void* p, std::size_t) noexcept { std::free(p); }

using namespace invaders;

TEST_CASE("a long, busy game performs zero heap allocations", "[game][pool][alloc]") {
    GameConfig config;
    config.enemyShotsPerSecond = 6.0f;  // keep every pool busy
    config.playerFireCooldown = 0.05f;
    auto game = std::make_unique<Game>(config, Rng(2024));
    Rng bot(7);

    long allocationsDuringPlay = 0;
    int waves = 0;
    {
        const long before = gAllocations.load();
        Input input;
        for (int tick = 0; tick < 120 * 120; ++tick) {  // two minutes of game time
            if (tick % 15 == 0) input = Input{static_cast<int>(bot.below(3)) - 1, true};
            game->step(input, test::kStep);
            if (game->phase() == Phase::GameOver) {
                game->restart();  // restarting must not allocate either
            }
            waves = std::max(waves, game->wave());
        }
        allocationsDuringPlay = gAllocations.load() - before;
    }

    CHECK(allocationsDuringPlay == 0);
    // And the run really exercised the pools.
    CHECK(game->particles().highWater() > 0);
    CHECK(game->enemyBullets().highWater() > 0);
    CHECK(waves >= 1);
}

TEST_CASE("the allocation counter really counts", "[alloc]") {
    const long before = gAllocations.load();
    // Call operator new directly: a new-expression may legally be elided.
    void* p = ::operator new(sizeof(int));
    ::operator delete(p);
    CHECK(gAllocations.load() - before == 1);
}
