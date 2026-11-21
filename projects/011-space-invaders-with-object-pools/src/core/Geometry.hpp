#pragma once

namespace invaders {

struct Vec2 {
    float x = 0.0f;
    float y = 0.0f;
};

// Axis-aligned rectangle; (x, y) is the top-left corner, y grows downwards.
struct Rect {
    float x = 0.0f;
    float y = 0.0f;
    float w = 0.0f;
    float h = 0.0f;

    float right() const { return x + w; }
    float bottom() const { return y + h; }
};

inline bool overlaps(const Rect& a, const Rect& b) {
    return a.x < b.right() && b.x < a.right() && a.y < b.bottom() && b.y < a.bottom();
}

// Rect centred on a point, the natural shape for bullets and the player.
inline Rect centeredRect(Vec2 center, float w, float h) {
    return Rect{center.x - w * 0.5f, center.y - h * 0.5f, w, h};
}

}  // namespace invaders
