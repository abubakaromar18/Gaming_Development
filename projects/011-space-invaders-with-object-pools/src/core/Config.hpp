#pragma once

// Tunable game parameters. Pool capacities are compile-time constants because
// they size fixed arrays inside the pools; everything else is a runtime value
// that is checked once by validate() so the simulation can trust it.

#include <cstddef>

namespace invaders {

namespace pool_capacity {
// The player may only have this many shots in flight, as in the arcade
// original, where it was famously one. Pool exhaustion *is* the fire limit.
inline constexpr std::size_t kPlayerBullets = 2;
inline constexpr std::size_t kEnemyBullets = 6;
inline constexpr std::size_t kInvaders = 66;  // up to 6 rows x 11 columns
inline constexpr std::size_t kParticles = 384;
}  // namespace pool_capacity

struct GameConfig {
    // World, in logical pixels.
    float worldWidth = 224.0f;
    float worldHeight = 256.0f;

    // Player cannon.
    float playerY = 228.0f;
    float playerWidth = 13.0f;
    float playerHeight = 8.0f;
    float playerSpeed = 90.0f;      // px/s
    float playerFireCooldown = 0.30f;  // s between shots
    float playerBulletSpeed = 260.0f;  // px/s, upwards
    float respawnInvulnerability = 1.5f;  // s
    int startingLives = 3;

    // Invader formation.
    int invaderRows = 5;
    int invaderCols = 11;
    float invaderWidth = 12.0f;
    float invaderHeight = 8.0f;
    float invaderSpacingX = 16.0f;
    float invaderSpacingY = 14.0f;
    float formationStartY = 40.0f;
    float formationSideMargin = 6.0f;
    float marchStepX = 2.0f;
    float marchDropY = 8.0f;
    // Seconds between formation steps with a full and a nearly empty board.
    // The interval shrinks as invaders die, which produces the classic
    // speed-up without any extra state.
    float marchIntervalFull = 0.50f;
    float marchIntervalLast = 0.02f;
    // Each wave starts this much lower, up to waveDropLimit waves deep.
    float waveDropPerWave = 8.0f;
    int waveDropLimit = 4;
    float waveClearPause = 1.5f;  // s

    // Invader fire.
    float enemyShotsPerSecond = 1.1f;
    float enemyBulletSpeed = 85.0f;  // px/s, downwards
    float enemyFireRateGrowthPerWave = 0.15f;

    // Bullet hit boxes.
    float bulletWidth = 1.0f;
    float bulletHeight = 5.0f;

    // Bunkers: a row of destructible blocks made of small cells.
    int bunkerCount = 4;
    float bunkerY = 196.0f;
    float bunkerCellSize = 3.0f;

    // Cosmetic explosion debris.
    int particlesPerExplosion = 14;
    float particleSpeed = 60.0f;
    float particleLifetime = 0.55f;

    // Throws std::invalid_argument naming the first invalid field.
    void validate() const;
};

}  // namespace invaders
