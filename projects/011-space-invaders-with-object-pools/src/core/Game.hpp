#pragma once

// The invaders simulation. Pure C++: no SDL, no clock, no global state.
// The platform layer feeds it an Input and a fixed dt each tick and then
// reads back state and events to draw and play effects.
//
// Every short-lived entity (both kinds of bullet, the invaders themselves,
// and explosion debris) lives in an ObjectPool. After construction the
// simulation performs no heap allocation at all, which a test enforces by
// counting calls to operator new across thousands of ticks.

#include <array>
#include <cstddef>
#include <cstdint>
#include <vector>

#include "core/Bunker.hpp"
#include "core/Config.hpp"
#include "core/Geometry.hpp"
#include "core/ObjectPool.hpp"
#include "core/Rng.hpp"

namespace invaders {

// What the player wants this tick, independent of keys or gamepads.
struct Input {
    int move = 0;  // -1 left, 0 still, +1 right
    bool fire = false;
};

enum class Phase { Playing, WaveClear, GameOver };

struct Bullet {
    Vec2 pos;
    float vy = 0.0f;
};

// Invaders store only their grid slot: position is derived from the shared
// formation origin, so marching the whole fleet is one vector addition.
struct Invader {
    int row = 0;
    int col = 0;
    int points = 0;
};

struct Particle {
    Vec2 pos;
    Vec2 vel;
    float age = 0.0f;
    float lifetime = 0.0f;
};

enum class EventType {
    PlayerFired,
    InvaderFired,
    InvaderKilled,
    PlayerHit,
    BunkerHit,
    BulletsCollided,
    WaveCleared,
    WaveStarted,
    GameOver,
};

// Things that happened during the last step, for sound and screen effects.
struct Event {
    EventType type;
    Vec2 where;
    int value = 0;  // points scored, wave number, lives left... depends on type
};

class Game {
public:
    using PlayerBulletPool = ObjectPool<Bullet, pool_capacity::kPlayerBullets>;
    using EnemyBulletPool = ObjectPool<Bullet, pool_capacity::kEnemyBullets>;
    using InvaderPool = ObjectPool<Invader, pool_capacity::kInvaders>;
    using ParticlePool = ObjectPool<Particle, pool_capacity::kParticles>;

    static constexpr std::size_t kMaxEventsPerStep = 128;
    static constexpr std::size_t kMaxBunkers = 8;

    // Validates the config (throws std::invalid_argument) and starts wave 1.
    Game(const GameConfig& config, Rng rng);

    // Pools are non-copyable, and so is the game that owns them.
    Game(const Game&) = delete;
    Game& operator=(const Game&) = delete;

    // Advances the simulation by dt seconds. dt must be >= 0.
    void step(const Input& input, float dt);

    // Starts a fresh game (score 0, full lives, wave 1). Keeps the RNG stream.
    void restart();

    Phase phase() const { return phase_; }
    int score() const { return score_; }
    int lives() const { return lives_; }
    int wave() const { return wave_; }

    float playerX() const { return playerX_; }
    Rect playerRect() const;
    bool playerInvulnerable() const { return invulnerableFor_ > 0.0f; }

    Rect invaderRect(const Invader& invader) const;
    Rect bulletRect(const Bullet& bullet) const;
    Vec2 formationOrigin() const { return formationOrigin_; }
    // Flips on every formation step; the renderer uses it to alternate
    // between two animation frames, exactly in time with the march.
    int marchFrame() const { return marchFrame_; }

    const PlayerBulletPool& playerBullets() const { return playerBullets_; }
    const EnemyBulletPool& enemyBullets() const { return enemyBullets_; }
    const InvaderPool& invaders() const { return invaders_; }
    const ParticlePool& particles() const { return particles_; }

    const Bunker* bunkersBegin() const { return bunkers_.data(); }
    const Bunker* bunkersEnd() const { return bunkers_.data() + bunkerCount_; }

    const std::vector<Event>& events() const { return events_; }
    const GameConfig& config() const { return config_; }

    // FNV-1a digest of the gameplay state, for determinism checks.
    std::uint64_t stateHash() const;

private:
    void startWave(int wave);
    void updatePlayer(const Input& input, float dt);
    void marchFormation(float dt);
    void enemyFire(float dt);
    void moveBullets(float dt);
    void resolveCollisions();
    void updateParticles(float dt);
    void spawnExplosion(Vec2 at);
    void emit(EventType type, Vec2 where, int value = 0);
    float currentMarchInterval() const;

    GameConfig config_;
    Rng rng_;

    Phase phase_ = Phase::Playing;
    int score_ = 0;
    int lives_ = 0;
    int wave_ = 0;
    int invadersPerWave_ = 0;

    float playerX_ = 0.0f;
    float fireCooldown_ = 0.0f;
    float invulnerableFor_ = 0.0f;

    Vec2 formationOrigin_{};
    float marchDirection_ = 1.0f;
    float marchTimer_ = 0.0f;
    int marchFrame_ = 0;
    float waveClearTimer_ = 0.0f;

    PlayerBulletPool playerBullets_;
    EnemyBulletPool enemyBullets_;
    InvaderPool invaders_;
    ParticlePool particles_;
    std::array<Bunker, kMaxBunkers> bunkers_{};
    int bunkerCount_ = 0;

    std::vector<Event> events_;  // reserved once; never grows past capacity
};

}  // namespace invaders
