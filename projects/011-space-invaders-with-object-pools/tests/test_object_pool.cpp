#include <catch2/catch_test_macros.hpp>

#include <map>
#include <vector>

#include "core/ObjectPool.hpp"
#include "core/Rng.hpp"

using invaders::ObjectPool;
using invaders::PoolHandle;

namespace {

struct Tracked {
    static inline int alive = 0;
    int value;
    explicit Tracked(int v) : value(v) { ++alive; }
    Tracked(const Tracked& other) : value(other.value) { ++alive; }
    Tracked& operator=(const Tracked&) = default;
    ~Tracked() { --alive; }
};

}  // namespace

TEST_CASE("acquire hands out slots until the pool is full", "[pool]") {
    ObjectPool<int, 3> pool;
    REQUIRE(pool.acquire(1));
    REQUIRE(pool.acquire(2));
    REQUIRE(pool.acquire(3));
    CHECK(pool.full());
    CHECK(pool.size() == 3);

    CHECK_FALSE(pool.acquire(4).has_value());
    CHECK_FALSE(pool.acquire(5).has_value());
    CHECK(pool.exhaustedCount() == 2);
    CHECK(pool.size() == 3);
}

TEST_CASE("released slots are reused most-recent first", "[pool]") {
    ObjectPool<int, 4> pool;
    const auto a = *pool.acquire(10);
    const auto b = *pool.acquire(20);
    REQUIRE(pool.release(a));
    REQUIRE(pool.release(b));
    const auto c = *pool.acquire(30);
    CHECK(c.index == b.index);  // LIFO keeps hot memory hot
    CHECK(*pool.get(c) == 30);
}

TEST_CASE("stale handles are rejected after their slot is recycled", "[pool]") {
    ObjectPool<int, 2> pool;
    const PoolHandle first = *pool.acquire(1);
    REQUIRE(pool.release(first));

    CHECK(pool.get(first) == nullptr);
    CHECK_FALSE(pool.release(first));  // double release is a no-op

    const PoolHandle reused = *pool.acquire(2);
    REQUIRE(reused.index == first.index);
    CHECK(reused.generation != first.generation);
    CHECK(pool.get(first) == nullptr);  // the old handle must not see the new object
    REQUIRE(pool.get(reused) != nullptr);
    CHECK(*pool.get(reused) == 2);
}

TEST_CASE("out-of-range handles are rejected", "[pool]") {
    ObjectPool<int, 2> pool;
    CHECK(pool.get(PoolHandle{99, 0}) == nullptr);
    CHECK_FALSE(pool.release(PoolHandle{99, 0}));
}

TEST_CASE("forEach visits exactly the live objects", "[pool]") {
    ObjectPool<int, 8> pool;
    std::vector<PoolHandle> handles;
    for (int i = 0; i < 6; ++i) handles.push_back(*pool.acquire(i));
    pool.release(handles[1]);
    pool.release(handles[4]);

    int sum = 0;
    int count = 0;
    pool.forEach([&](int v) {
        sum += v;
        ++count;
    });
    CHECK(count == 4);
    CHECK(sum == 0 + 2 + 3 + 5);
}

TEST_CASE("removeIf releases matching objects while iterating", "[pool]") {
    ObjectPool<int, 16> pool;
    for (int i = 0; i < 16; ++i) pool.acquire(i);

    const std::size_t removed = pool.removeIf([](int v) { return v % 2 == 0; });
    CHECK(removed == 8);
    CHECK(pool.size() == 8);
    pool.forEach([](int v) { CHECK(v % 2 == 1); });

    // Freed slots are immediately reusable.
    for (int i = 0; i < 8; ++i) CHECK(pool.acquire(100 + i));
    CHECK(pool.full());
}

TEST_CASE("objects are constructed in place and destroyed on release", "[pool]") {
    Tracked::alive = 0;
    {
        ObjectPool<Tracked, 4> pool;
        CHECK(Tracked::alive == 0);  // storage is reserved, nothing constructed
        const auto h = *pool.acquire(7);
        pool.acquire(8);
        CHECK(Tracked::alive == 2);
        pool.release(h);
        CHECK(Tracked::alive == 1);
    }
    CHECK(Tracked::alive == 0);  // the pool destroys what is still live
}

TEST_CASE("clear releases everything, invalidates handles and resets stats", "[pool]") {
    ObjectPool<int, 2> pool;
    const auto h = *pool.acquire(1);
    pool.acquire(2);
    pool.acquire(3);  // fails
    REQUIRE(pool.highWater() == 2);
    REQUIRE(pool.exhaustedCount() == 1);

    pool.clear();
    CHECK(pool.empty());
    CHECK(pool.get(h) == nullptr);
    CHECK(pool.highWater() == 0);
    CHECK(pool.exhaustedCount() == 0);
    CHECK(pool.acquire(4));
}

TEST_CASE("high-water mark records peak usage", "[pool]") {
    ObjectPool<int, 10> pool;
    std::vector<PoolHandle> handles;
    for (int i = 0; i < 7; ++i) handles.push_back(*pool.acquire(i));
    for (const auto& h : handles) pool.release(h);
    pool.acquire(0);
    CHECK(pool.size() == 1);
    CHECK(pool.highWater() == 7);
}

TEST_CASE("random churn matches a reference model", "[pool][stress]") {
    // Hammer the pool with random acquire/release/removeIf and compare with
    // a std::map that tracks what should be live.
    ObjectPool<int, 64> pool;
    std::map<std::uint32_t, std::pair<PoolHandle, int>> model;  // slot -> (handle, value)
    invaders::Rng rng(1234);
    int nextValue = 0;

    for (int op = 0; op < 50000; ++op) {
        const std::uint32_t choice = rng.below(10);
        if (choice < 6) {
            const auto handle = pool.acquire(nextValue);
            REQUIRE(handle.has_value() == (model.size() < 64));
            if (handle) model[handle->index] = {*handle, nextValue};
            ++nextValue;
        } else if (choice < 9 && !model.empty()) {
            auto it = model.begin();
            std::advance(it, rng.below(static_cast<std::uint32_t>(model.size())));
            REQUIRE(pool.release(it->second.first));
            model.erase(it);
        } else {
            const int mod = static_cast<int>(rng.below(5)) + 2;
            pool.removeIf([mod](int v) { return v % mod == 0; });
            for (auto it = model.begin(); it != model.end();) {
                it = (it->second.second % mod == 0) ? model.erase(it) : std::next(it);
            }
        }

        REQUIRE(pool.size() == model.size());
    }
    for (const auto& [slot, entry] : model) {
        const int* value = pool.get(entry.first);
        REQUIRE(value != nullptr);
        CHECK(*value == entry.second);
    }
}
