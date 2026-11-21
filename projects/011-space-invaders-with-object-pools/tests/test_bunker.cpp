#include <catch2/catch_test_macros.hpp>

#include "core/Bunker.hpp"

using invaders::Bunker;
using invaders::Rect;
using invaders::Vec2;

namespace {

// A bullet-shaped rect through the middle of column `col`.
Rect shotThroughColumn(const Bunker& bunker, int col) {
    const Rect cell = bunker.cellRect(col, 0);
    const Rect box = bunker.bounds();
    return Rect{cell.x + cell.w * 0.25f, box.y - 1.0f, cell.w * 0.5f, box.h + 2.0f};
}

}  // namespace

TEST_CASE("a fresh bunker has the arch shape", "[bunker]") {
    const Bunker bunker(Vec2{10.0f, 20.0f}, 3.0f);
    CHECK(bunker.liveCells() == 4 + 6 + 8 + 8 + 6 + 4);
    CHECK_FALSE(bunker.cellAlive(0, 0));      // rounded corner
    CHECK(bunker.cellAlive(3, 0));
    CHECK_FALSE(bunker.cellAlive(3, Bunker::kRows - 1));  // notch underneath
    CHECK(bunker.bounds().w == 24.0f);
}

TEST_CASE("shots from below chip the lowest cell they overlap", "[bunker]") {
    Bunker bunker(Vec2{0.0f, 0.0f}, 3.0f);
    const Rect shot = shotThroughColumn(bunker, 0);
    REQUIRE(bunker.erodeFirstHit(shot, /*travellingUp=*/true));
    CHECK_FALSE(bunker.cellAlive(0, 5));
    CHECK(bunker.cellAlive(0, 4));
    CHECK(bunker.liveCells() == 35);
}

TEST_CASE("shots from above chip the highest cell they overlap", "[bunker]") {
    Bunker bunker(Vec2{0.0f, 0.0f}, 3.0f);
    const Rect shot = shotThroughColumn(bunker, 0);
    REQUIRE(bunker.erodeFirstHit(shot, /*travellingUp=*/false));
    CHECK_FALSE(bunker.cellAlive(0, 2));  // column 0 starts at row 2
    CHECK(bunker.cellAlive(0, 3));
}

TEST_CASE("repeated shots tunnel through and then pass", "[bunker]") {
    Bunker bunker(Vec2{0.0f, 0.0f}, 3.0f);
    const Rect shot = shotThroughColumn(bunker, 3);
    int hits = 0;
    while (bunker.erodeFirstHit(shot, true)) ++hits;
    CHECK(hits == 4);  // column 3 has four cells above the notch
    CHECK_FALSE(bunker.erodeFirstHit(shot, false));
}

TEST_CASE("shots that miss the bunker do nothing", "[bunker]") {
    Bunker bunker(Vec2{100.0f, 100.0f}, 3.0f);
    CHECK_FALSE(bunker.erodeFirstHit(Rect{0.0f, 0.0f, 1.0f, 5.0f}, true));
    CHECK(bunker.liveCells() == 36);
}

TEST_CASE("eraseOverlapping bulldozes an area", "[bunker]") {
    Bunker bunker(Vec2{0.0f, 0.0f}, 3.0f);
    const int erased = bunker.eraseOverlapping(Rect{0.0f, 0.0f, 24.0f, 6.0f});  // top two rows
    CHECK(erased == 4 + 6);
    CHECK(bunker.liveCells() == 36 - 10);
}
