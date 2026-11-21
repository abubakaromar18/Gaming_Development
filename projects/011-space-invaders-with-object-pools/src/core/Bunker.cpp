#include "core/Bunker.hpp"

namespace invaders {

namespace {

// The classic arch: rounded top, notch underneath where the cannon hides.
constexpr const char* kShape[Bunker::kRows] = {
    "..XXXX..",
    ".XXXXXX.",
    "XXXXXXXX",
    "XXXXXXXX",
    "XXX..XXX",
    "XX....XX",
};

}  // namespace

Bunker::Bunker(Vec2 topLeft, float cellSize) : topLeft_(topLeft), cellSize_(cellSize) {
    for (int row = 0; row < kRows; ++row) {
        for (int col = 0; col < kCols; ++col) {
            cells_[index(col, row)] = kShape[row][col] == 'X';
        }
    }
}

Rect Bunker::cellRect(int col, int row) const {
    return Rect{topLeft_.x + static_cast<float>(col) * cellSize_,
                topLeft_.y + static_cast<float>(row) * cellSize_, cellSize_, cellSize_};
}

Rect Bunker::bounds() const {
    return Rect{topLeft_.x, topLeft_.y, cellSize_ * kCols, cellSize_ * kRows};
}

bool Bunker::erodeFirstHit(const Rect& bullet, bool travellingUp) {
    if (!overlaps(bullet, bounds())) return false;
    // Scan rows in the order the bullet meets them.
    for (int step = 0; step < kRows; ++step) {
        const int row = travellingUp ? kRows - 1 - step : step;
        for (int col = 0; col < kCols; ++col) {
            if (cells_[index(col, row)] && overlaps(bullet, cellRect(col, row))) {
                cells_[index(col, row)] = false;
                return true;
            }
        }
    }
    return false;
}

int Bunker::eraseOverlapping(const Rect& area) {
    if (!overlaps(area, bounds())) return 0;
    int erased = 0;
    for (int row = 0; row < kRows; ++row) {
        for (int col = 0; col < kCols; ++col) {
            if (cells_[index(col, row)] && overlaps(area, cellRect(col, row))) {
                cells_[index(col, row)] = false;
                ++erased;
            }
        }
    }
    return erased;
}

int Bunker::liveCells() const {
    int alive = 0;
    for (bool cell : cells_) alive += cell ? 1 : 0;
    return alive;
}

}  // namespace invaders
