#include "core/Config.hpp"

#include <stdexcept>
#include <string>

namespace invaders {

namespace {

void require(bool ok, const char* field, const char* rule) {
    if (!ok) throw std::invalid_argument(std::string("GameConfig.") + field + " " + rule);
}

}  // namespace

void GameConfig::validate() const {
    require(worldWidth > 0.0f && worldHeight > 0.0f, "worldWidth/worldHeight", "must be positive");
    require(playerY > 0.0f && playerY < worldHeight, "playerY", "must lie inside the world");
    require(playerWidth > 0.0f && playerWidth < worldWidth, "playerWidth", "must fit in the world");
    require(playerHeight > 0.0f, "playerHeight", "must be positive");
    require(playerSpeed > 0.0f, "playerSpeed", "must be positive");
    require(playerFireCooldown >= 0.0f, "playerFireCooldown", "must not be negative");
    require(playerBulletSpeed > 0.0f, "playerBulletSpeed", "must be positive");
    require(respawnInvulnerability >= 0.0f, "respawnInvulnerability", "must not be negative");
    require(startingLives > 0, "startingLives", "must be at least 1");

    require(invaderRows > 0 && invaderCols > 0, "invaderRows/invaderCols", "must be positive");
    require(static_cast<std::size_t>(invaderRows) * static_cast<std::size_t>(invaderCols) <=
                pool_capacity::kInvaders,
            "invaderRows*invaderCols", "exceeds pool_capacity::kInvaders");
    require(invaderWidth > 0.0f && invaderHeight > 0.0f, "invaderWidth/invaderHeight",
            "must be positive");
    require(invaderSpacingX >= invaderWidth, "invaderSpacingX", "must be >= invaderWidth");
    require(invaderSpacingY >= invaderHeight, "invaderSpacingY", "must be >= invaderHeight");
    const float formationWidth = invaderSpacingX * static_cast<float>(invaderCols - 1) + invaderWidth;
    require(formationWidth + 2.0f * formationSideMargin < worldWidth, "invaderCols",
            "formation is wider than the world");
    require(formationStartY >= 0.0f && formationStartY < playerY, "formationStartY",
            "must lie above the player");
    require(marchStepX > 0.0f && marchDropY > 0.0f, "marchStepX/marchDropY", "must be positive");
    require(marchIntervalFull >= marchIntervalLast && marchIntervalLast > 0.0f,
            "marchIntervalFull/marchIntervalLast", "need full >= last > 0");
    require(waveDropPerWave >= 0.0f && waveDropLimit >= 0, "waveDropPerWave/waveDropLimit",
            "must not be negative");
    require(waveClearPause >= 0.0f, "waveClearPause", "must not be negative");

    require(enemyShotsPerSecond >= 0.0f, "enemyShotsPerSecond", "must not be negative");
    require(enemyBulletSpeed > 0.0f, "enemyBulletSpeed", "must be positive");
    require(enemyFireRateGrowthPerWave >= 0.0f, "enemyFireRateGrowthPerWave",
            "must not be negative");
    require(bulletWidth > 0.0f && bulletHeight > 0.0f, "bulletWidth/bulletHeight",
            "must be positive");

    require(bunkerCount >= 0 && bunkerCount <= 8, "bunkerCount", "must be in [0, 8]");
    require(bunkerCellSize > 0.0f, "bunkerCellSize", "must be positive");
    require(bunkerY > formationStartY && bunkerY < playerY, "bunkerY",
            "must lie between the formation and the player");

    require(particlesPerExplosion >= 0, "particlesPerExplosion", "must not be negative");
    require(particleSpeed >= 0.0f && particleLifetime > 0.0f, "particleSpeed/particleLifetime",
            "need speed >= 0 and lifetime > 0");
}

}  // namespace invaders
