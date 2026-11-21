#pragma once

// Small, fast, seedable PRNG (xorshift64*). The game never touches global
// randomness: a seeded Rng is passed in, so two games with the same seed and
// the same inputs play out identically, which is what the tests rely on.

#include <cstdint>

namespace invaders {

class Rng {
public:
    explicit Rng(std::uint64_t seed) : state_(seed ? seed : 0x9E3779B97F4A7C15ull) {}

    std::uint32_t nextU32() {
        state_ ^= state_ >> 12;
        state_ ^= state_ << 25;
        state_ ^= state_ >> 27;
        return static_cast<std::uint32_t>((state_ * 0x2545F4914F6CDD1Dull) >> 32);
    }

    // Uniform float in [0, 1).
    float nextFloat() { return static_cast<float>(nextU32() >> 8) * (1.0f / 16777216.0f); }

    // Uniform float in [lo, hi).
    float range(float lo, float hi) { return lo + (hi - lo) * nextFloat(); }

    // Uniform integer in [0, n). n must be > 0.
    std::uint32_t below(std::uint32_t n) {
        return static_cast<std::uint32_t>((static_cast<std::uint64_t>(nextU32()) * n) >> 32);
    }

private:
    std::uint64_t state_;
};

}  // namespace invaders
