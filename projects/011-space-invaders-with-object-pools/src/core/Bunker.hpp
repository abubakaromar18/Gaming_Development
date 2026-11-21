#pragma once

// A destructible shield: a small grid of cells that bullets chip away one at
// a time and that invaders bulldoze as they march through it.

#include <array>

#include "core/Geometry.hpp"

namespace invaders {

class Bunker {
public:
    static constexpr int kCols = 8;
    static constexpr int kRows = 6;

    Bunker() = default;
    Bunker(Vec2 topLeft, float cellSize);

    // Destroys the single cell a bullet travelling vertically would strike
    // first: the lowest overlapping cell for a shot coming from below, the
    // highest for a shot coming from above. Returns true if a cell was hit.
    bool erodeFirstHit(const Rect& bullet, bool travellingUp);

    // Destroys every cell overlapping the rect. Returns how many died.
    int eraseOverlapping(const Rect& area);

    bool cellAlive(int col, int row) const { return cells_[index(col, row)]; }
    Rect cellRect(int col, int row) const;
    Rect bounds() const;
    int liveCells() const;

private:
    static constexpr int index(int col, int row) { return row * kCols + col; }

    Vec2 topLeft_{};
    float cellSize_ = 1.0f;
    std::array<bool, kCols * kRows> cells_{};
};

}  // namespace invaders
