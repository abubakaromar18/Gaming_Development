// Bullet-churn benchmark: every frame spawns a burst of bullets, moves every
// live bullet, and destroys the ones that left the screen. The same workload
// runs against three containers:
//
//   pool         ObjectPool: no allocation, dense iteration
//   vector<ptr>  std::vector<std::unique_ptr<Bullet>>: one new/delete per bullet
//   list         std::list<Bullet>: one node allocation per bullet, pointer chasing
//
// Usage: pool_benchmark [frames]   (default 20000)

#include <algorithm>
#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <list>
#include <memory>
#include <stdexcept>
#include <string>
#include <vector>

#include "core/ObjectPool.hpp"
#include "core/Rng.hpp"

namespace {

using invaders::Rng;

struct Bullet {
    float x, y, vx, vy;
};

constexpr std::size_t kCapacity = 4096;
constexpr int kSpawnPerFrame = 48;
constexpr float kDt = 1.0f / 120.0f;
constexpr float kScreenTop = -100.0f;  // bullets live 30-60 frames, ~2,200 at once

Bullet makeBullet(Rng& rng) { return Bullet{rng.range(0.0f, 224.0f), 0.0f, rng.range(-20.0f, 20.0f), -rng.range(200.0f, 400.0f)}; }

void move(Bullet& b) {
    b.x += b.vx * kDt;
    b.y += b.vy * kDt;
}

bool offscreen(const Bullet& b) { return b.y < kScreenTop; }

struct Result {
    double millis;
    std::size_t checksum;  // live bullets summed over all frames: order-independent
    std::size_t peak;
};

template <typename Fn>
Result timeIt(Fn&& fn) {
    const auto start = std::chrono::steady_clock::now();
    Result r = fn();
    r.millis = std::chrono::duration<double, std::milli>(std::chrono::steady_clock::now() - start).count();
    return r;
}

Result runPool(int frames) {
    static invaders::ObjectPool<Bullet, kCapacity> pool;  // static: 64 KiB is too big for some stacks
    pool.clear();
    Rng rng(1);
    std::size_t checksum = 0;
    std::size_t peak = 0;
    for (int f = 0; f < frames; ++f) {
        for (int i = 0; i < kSpawnPerFrame; ++i) pool.acquire(makeBullet(rng));
        pool.forEach(move);
        pool.removeIf(offscreen);
        checksum += pool.size();
        peak = std::max(peak, pool.size());
    }
    if (pool.exhaustedCount() > 0) std::fprintf(stderr, "pool too small for the workload\n");
    return Result{0.0, checksum, peak};
}

Result runVectorOfPointers(int frames) {
    std::vector<std::unique_ptr<Bullet>> bullets;
    Rng rng(1);
    std::size_t checksum = 0;
    std::size_t peak = 0;
    for (int f = 0; f < frames; ++f) {
        for (int i = 0; i < kSpawnPerFrame; ++i) bullets.push_back(std::make_unique<Bullet>(makeBullet(rng)));
        for (auto& b : bullets) move(*b);
        std::erase_if(bullets, [](const auto& b) { return offscreen(*b); });
        checksum += bullets.size();
        peak = std::max(peak, bullets.size());
    }
    return Result{0.0, checksum, peak};
}

Result runList(int frames) {
    std::list<Bullet> bullets;
    Rng rng(1);
    std::size_t checksum = 0;
    std::size_t peak = 0;
    for (int f = 0; f < frames; ++f) {
        for (int i = 0; i < kSpawnPerFrame; ++i) bullets.push_back(makeBullet(rng));
        for (auto& b : bullets) move(b);
        bullets.remove_if(offscreen);
        checksum += bullets.size();
        peak = std::max(peak, bullets.size());
    }
    return Result{0.0, checksum, peak};
}

void report(const char* name, const Result& r, int frames, double baseline) {
    std::printf("%-12s %9.2f ms  %8.1f ns/frame  peak %5zu live  x%.2f vs pool  (checksum %zu)\n", name, r.millis,
                r.millis * 1e6 / frames, r.peak, r.millis / baseline, r.checksum);
}

}  // namespace

int main(int argc, char** argv) {
    int frames = 20000;
    if (argc > 1) {
        try {
            frames = std::stoi(argv[1]);
        } catch (const std::exception&) {
            std::fprintf(stderr, "usage: pool_benchmark [frames]\n");
            return EXIT_FAILURE;
        }
        if (frames <= 0) {
            std::fprintf(stderr, "frames must be positive\n");
            return EXIT_FAILURE;
        }
    }

    std::printf("%d frames, %d spawns/frame\n", frames, kSpawnPerFrame);
    const Result pool = timeIt([&] { return runPool(frames); });
    const Result vec = timeIt([&] { return runVectorOfPointers(frames); });
    const Result list = timeIt([&] { return runList(frames); });

    report("pool", pool, frames, pool.millis);
    report("vector<ptr>", vec, frames, pool.millis);
    report("list", list, frames, pool.millis);

    // All three must have simulated the same bullets.
    if (pool.checksum != vec.checksum || pool.checksum != list.checksum) {
        std::fprintf(stderr, "checksum mismatch: the workloads diverged\n");
        return EXIT_FAILURE;
    }
    return EXIT_SUCCESS;
}
