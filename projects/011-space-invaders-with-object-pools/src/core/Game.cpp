#include "core/Game.hpp"

#include <algorithm>
#include <cmath>
#include <cstring>
#include <stdexcept>

namespace invaders {

namespace {

constexpr float kTwoPi = 6.28318530718f;
constexpr float kParticleDrag = 2.5f;  // 1/s, exponential slow-down of debris

int pointsForRow(int row) {
    if (row == 0) return 30;
    if (row <= 2) return 20;
    return 10;
}

}  // namespace

Game::Game(const GameConfig& config, Rng rng) : config_(config), rng_(rng) {
    config_.validate();
    // The only allocation the game ever makes: room for one step's events.
    events_.reserve(kMaxEventsPerStep);
    restart();
}

void Game::restart() {
    score_ = 0;
    lives_ = config_.startingLives;
    playerX_ = config_.worldWidth * 0.5f;
    fireCooldown_ = 0.0f;
    invulnerableFor_ = 0.0f;
    particles_.clear();
    events_.clear();
    startWave(1);
}

void Game::startWave(int wave) {
    wave_ = wave;
    phase_ = Phase::Playing;
    playerBullets_.clear();
    enemyBullets_.clear();
    invaders_.clear();

    const float formationWidth =
        config_.invaderSpacingX * static_cast<float>(config_.invaderCols - 1) + config_.invaderWidth;
    const int depth = std::min(wave - 1, config_.waveDropLimit);
    formationOrigin_ = Vec2{(config_.worldWidth - formationWidth) * 0.5f,
                            config_.formationStartY + config_.waveDropPerWave * static_cast<float>(depth)};
    marchDirection_ = 1.0f;
    marchTimer_ = 0.0f;
    marchFrame_ = 0;

    // Validation guarantees the grid fits, so every acquire succeeds.
    for (int row = 0; row < config_.invaderRows; ++row) {
        for (int col = 0; col < config_.invaderCols; ++col) {
            invaders_.acquire(Invader{row, col, pointsForRow(row)});
        }
    }
    invadersPerWave_ = static_cast<int>(invaders_.size());

    // Bunkers are spread evenly across the width and rebuilt every wave.
    bunkerCount_ = config_.bunkerCount;
    const float bunkerWidth = config_.bunkerCellSize * Bunker::kCols;
    const float gap = (config_.worldWidth - bunkerWidth * static_cast<float>(bunkerCount_)) /
                      static_cast<float>(bunkerCount_ + 1);
    for (int i = 0; i < bunkerCount_; ++i) {
        const float x = gap + static_cast<float>(i) * (bunkerWidth + gap);
        bunkers_[static_cast<std::size_t>(i)] = Bunker(Vec2{x, config_.bunkerY}, config_.bunkerCellSize);
    }

    emit(EventType::WaveStarted, Vec2{config_.worldWidth * 0.5f, config_.worldHeight * 0.5f}, wave);
}

void Game::step(const Input& input, float dt) {
    if (!(dt >= 0.0f)) throw std::invalid_argument("Game::step: dt must be >= 0");
    events_.clear();

    // Debris keeps flying in every phase so explosions finish gracefully.
    updateParticles(dt);

    switch (phase_) {
        case Phase::GameOver:
            return;
        case Phase::WaveClear:
            waveClearTimer_ -= dt;
            if (waveClearTimer_ <= 0.0f) startWave(wave_ + 1);
            return;
        case Phase::Playing:
            break;
    }

    updatePlayer(input, dt);
    marchFormation(dt);
    if (phase_ != Phase::Playing) return;
    enemyFire(dt);
    moveBullets(dt);
    resolveCollisions();

    if (phase_ == Phase::Playing && invaders_.empty()) {
        phase_ = Phase::WaveClear;
        waveClearTimer_ = config_.waveClearPause;
        emit(EventType::WaveCleared, Vec2{config_.worldWidth * 0.5f, config_.worldHeight * 0.5f}, wave_);
    }
}

void Game::updatePlayer(const Input& input, float dt) {
    const int move = std::clamp(input.move, -1, 1);
    const float half = config_.playerWidth * 0.5f;
    playerX_ = std::clamp(playerX_ + static_cast<float>(move) * config_.playerSpeed * dt, half,
                          config_.worldWidth - half);

    fireCooldown_ = std::max(0.0f, fireCooldown_ - dt);
    invulnerableFor_ = std::max(0.0f, invulnerableFor_ - dt);

    if (input.fire && fireCooldown_ <= 0.0f) {
        const Vec2 muzzle{playerX_, config_.playerY - config_.playerHeight * 0.5f - config_.bulletHeight * 0.5f};
        // A full pool means the player's shots are all still in the air.
        // That is the game rule, so a failed acquire is simply "no shot".
        if (playerBullets_.acquire(Bullet{muzzle, -config_.playerBulletSpeed})) {
            fireCooldown_ = config_.playerFireCooldown;
            emit(EventType::PlayerFired, muzzle);
        }
    }
}

float Game::currentMarchInterval() const {
    if (invadersPerWave_ <= 1) return config_.marchIntervalLast;
    const float remaining = static_cast<float>(invaders_.size() - 1) / static_cast<float>(invadersPerWave_ - 1);
    return config_.marchIntervalLast + (config_.marchIntervalFull - config_.marchIntervalLast) * remaining;
}

void Game::marchFormation(float dt) {
    if (invaders_.empty()) return;
    marchTimer_ += dt;
    const float interval = currentMarchInterval();
    if (marchTimer_ < interval) return;
    marchTimer_ -= interval;
    // Never bank more than one step: after a long hitch the fleet should not
    // teleport several steps at once.
    marchTimer_ = std::min(marchTimer_, interval);
    marchFrame_ ^= 1;

    // Find the live fleet's horizontal extent; dead columns don't count, so
    // the survivors march all the way to the wall.
    float minX = config_.worldWidth;
    float maxX = 0.0f;
    invaders_.forEach([&](const Invader& inv) {
        const Rect r = invaderRect(inv);
        minX = std::min(minX, r.x);
        maxX = std::max(maxX, r.right());
    });

    const float dx = config_.marchStepX * marchDirection_;
    const bool hitsWall = minX + dx < config_.formationSideMargin ||
                          maxX + dx > config_.worldWidth - config_.formationSideMargin;
    if (hitsWall) {
        formationOrigin_.y += config_.marchDropY;
        marchDirection_ = -marchDirection_;
    } else {
        formationOrigin_.x += dx;
    }

    // Invaders bulldoze bunkers and win if they reach the cannon's row.
    const float invasionLine = config_.playerY - config_.playerHeight * 0.5f;
    bool invaded = false;
    invaders_.forEach([&](const Invader& inv) {
        const Rect r = invaderRect(inv);
        for (int i = 0; i < bunkerCount_; ++i) bunkers_[static_cast<std::size_t>(i)].eraseOverlapping(r);
        if (r.bottom() >= invasionLine) invaded = true;
    });
    if (invaded) {
        lives_ = 0;
        phase_ = Phase::GameOver;
        emit(EventType::GameOver, Vec2{playerX_, config_.playerY}, score_);
    }
}

void Game::enemyFire(float dt) {
    const float rate =
        config_.enemyShotsPerSecond * (1.0f + config_.enemyFireRateGrowthPerWave * static_cast<float>(wave_ - 1));
    // Always draw from the RNG so the random stream doesn't depend on how
    // many invaders are alive. That keeps replays stable across tuning.
    const float roll = rng_.nextFloat();
    if (roll >= rate * dt) return;

    // Only the lowest invader in each column may shoot; the others would
    // hit their own comrades. Fixed-size scratch arrays keep this
    // allocation-free.
    std::array<int, pool_capacity::kInvaders> lowestRow;
    lowestRow.fill(-1);
    invaders_.forEach([&](const Invader& inv) {
        int& slot = lowestRow[static_cast<std::size_t>(inv.col)];
        slot = std::max(slot, inv.row);
    });
    std::array<int, pool_capacity::kInvaders> shooterCols;
    std::uint32_t shooterCount = 0;
    for (int col = 0; col < config_.invaderCols; ++col) {
        if (lowestRow[static_cast<std::size_t>(col)] >= 0) shooterCols[shooterCount++] = col;
    }
    if (shooterCount == 0) return;

    const int col = shooterCols[rng_.below(shooterCount)];
    const Invader shooter{lowestRow[static_cast<std::size_t>(col)], col, 0};
    const Rect r = invaderRect(shooter);
    const Vec2 muzzle{r.x + r.w * 0.5f, r.bottom() + config_.bulletHeight * 0.5f};
    // Exhaustion here caps how many enemy shots can be on screen at once.
    if (enemyBullets_.acquire(Bullet{muzzle, config_.enemyBulletSpeed})) {
        emit(EventType::InvaderFired, muzzle);
    }
}

void Game::moveBullets(float dt) {
    playerBullets_.forEach([&](Bullet& b) { b.pos.y += b.vy * dt; });
    enemyBullets_.forEach([&](Bullet& b) { b.pos.y += b.vy * dt; });
}

void Game::resolveCollisions() {
    const float halfBullet = config_.bulletHeight * 0.5f;

    // 1. Shots that meet in mid-air cancel each other out.
    playerBullets_.removeIf([&](const Bullet& mine) {
        const Rect a = bulletRect(mine);
        const std::size_t hit = enemyBullets_.removeIf([&](const Bullet& theirs) { return overlaps(a, bulletRect(theirs)); });
        if (hit > 0) emit(EventType::BulletsCollided, mine.pos);
        return hit > 0;
    });

    // 2. Player shots: bunkers block them, otherwise they kill one invader
    //    or leave the top of the screen.
    playerBullets_.removeIf([&](const Bullet& b) {
        const Rect br = bulletRect(b);
        for (int i = 0; i < bunkerCount_; ++i) {
            if (bunkers_[static_cast<std::size_t>(i)].erodeFirstHit(br, /*travellingUp=*/true)) {
                emit(EventType::BunkerHit, b.pos);
                return true;
            }
        }
        bool killed = false;
        invaders_.removeIf([&](const Invader& inv) {
            if (killed) return false;
            const Rect ir = invaderRect(inv);
            if (!overlaps(br, ir)) return false;
            killed = true;
            score_ += inv.points;
            const Vec2 center{ir.x + ir.w * 0.5f, ir.y + ir.h * 0.5f};
            emit(EventType::InvaderKilled, center, inv.points);
            spawnExplosion(center);
            return true;
        });
        return killed || b.pos.y + halfBullet < 0.0f;
    });

    // 3. Enemy shots: bunkers, then the player, then the floor.
    const Rect player = playerRect();
    enemyBullets_.removeIf([&](const Bullet& b) {
        const Rect br = bulletRect(b);
        for (int i = 0; i < bunkerCount_; ++i) {
            if (bunkers_[static_cast<std::size_t>(i)].erodeFirstHit(br, /*travellingUp=*/false)) {
                emit(EventType::BunkerHit, b.pos);
                return true;
            }
        }
        if (phase_ == Phase::Playing && overlaps(br, player)) {
            if (playerInvulnerable()) return true;  // absorbed by the respawn shield
            --lives_;
            invulnerableFor_ = config_.respawnInvulnerability;
            const Vec2 center{playerX_, config_.playerY};
            emit(EventType::PlayerHit, center, lives_);
            spawnExplosion(center);
            if (lives_ <= 0) {
                phase_ = Phase::GameOver;
                emit(EventType::GameOver, center, score_);
            }
            return true;
        }
        return b.pos.y - halfBullet > config_.worldHeight;
    });
}

void Game::spawnExplosion(Vec2 at) {
    for (int i = 0; i < config_.particlesPerExplosion; ++i) {
        const float angle = rng_.range(0.0f, kTwoPi);
        const float speed = config_.particleSpeed * rng_.range(0.3f, 1.0f);
        const float lifetime = config_.particleLifetime * rng_.range(0.6f, 1.0f);
        // Debris is cosmetic: if the pool is full the extra pieces are
        // dropped and the game is none the worse for it.
        particles_.acquire(Particle{at, Vec2{std::cos(angle) * speed, std::sin(angle) * speed}, 0.0f, lifetime});
    }
}

void Game::updateParticles(float dt) {
    const float damping = std::exp(-kParticleDrag * dt);
    particles_.removeIf([&](Particle& p) {
        p.age += dt;
        p.pos.x += p.vel.x * dt;
        p.pos.y += p.vel.y * dt;
        p.vel.x *= damping;
        p.vel.y *= damping;
        return p.age >= p.lifetime;
    });
}

void Game::emit(EventType type, Vec2 where, int value) {
    // Dropping an event is better than growing the vector mid-game.
    if (events_.size() < events_.capacity()) events_.push_back(Event{type, where, value});
}

Rect Game::playerRect() const {
    return centeredRect(Vec2{playerX_, config_.playerY}, config_.playerWidth, config_.playerHeight);
}

Rect Game::invaderRect(const Invader& invader) const {
    return Rect{formationOrigin_.x + config_.invaderSpacingX * static_cast<float>(invader.col),
                formationOrigin_.y + config_.invaderSpacingY * static_cast<float>(invader.row),
                config_.invaderWidth, config_.invaderHeight};
}

Rect Game::bulletRect(const Bullet& bullet) const {
    return centeredRect(bullet.pos, config_.bulletWidth, config_.bulletHeight);
}

std::uint64_t Game::stateHash() const {
    std::uint64_t h = 0xcbf29ce484222325ull;
    auto mix = [&h](const void* data, std::size_t n) {
        const auto* bytes = static_cast<const unsigned char*>(data);
        for (std::size_t i = 0; i < n; ++i) {
            h ^= bytes[i];
            h *= 0x100000001b3ull;
        }
    };
    auto mixFloat = [&mix](float f) {
        std::uint32_t bits;
        std::memcpy(&bits, &f, sizeof bits);
        mix(&bits, sizeof bits);
    };
    const int header[] = {static_cast<int>(phase_), score_, lives_, wave_, marchFrame_, bunkerCount_};
    mix(header, sizeof header);
    mixFloat(playerX_);
    mixFloat(formationOrigin_.x);
    mixFloat(formationOrigin_.y);
    invaders_.forEach([&](const Invader& inv) {
        const int cell[] = {inv.row, inv.col};
        mix(cell, sizeof cell);
    });
    auto mixBullet = [&](const Bullet& b) {
        mixFloat(b.pos.x);
        mixFloat(b.pos.y);
    };
    playerBullets_.forEach(mixBullet);
    enemyBullets_.forEach(mixBullet);
    for (int i = 0; i < bunkerCount_; ++i) {
        const int alive = bunkers_[static_cast<std::size_t>(i)].liveCells();
        mix(&alive, sizeof alive);
    }
    return h;
}

}  // namespace invaders
