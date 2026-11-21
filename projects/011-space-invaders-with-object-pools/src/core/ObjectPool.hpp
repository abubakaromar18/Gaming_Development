#pragma once

// A fixed-capacity object pool with O(1) acquire/release, dense iteration,
// and generation-checked handles.
//
// Why games pool objects: a shooter can spawn and destroy hundreds of
// bullets and particles per second. Calling new/delete for each one costs
// time, fragments the heap, and makes frame times spiky. A pool reserves all
// the storage once, up front, and recycles slots forever after. Once the
// pool is built, gameplay never touches the allocator.
//
// Layout (all arrays have Capacity entries, all live inside the pool object):
//
//   slots_        the objects themselves, each in a std::optional so that
//                 construction/destruction happen in place without the heap
//   generations_  bumped on every release; stale handles stop matching
//   freeStack_    indices of free slots (LIFO, so recently-freed and still
//                 cache-warm slots are reused first)
//   dense_        indices of live slots, packed at the front, so iteration
//                 touches only live objects instead of scanning every slot
//   denseIndex_   where each live slot sits inside dense_, so release can
//                 swap-and-pop in O(1)

#include <array>
#include <cstddef>
#include <cstdint>
#include <optional>
#include <utility>

namespace invaders {

// A weak reference to a pooled object. It stays safe to hold after the object
// is released: lookups through a stale handle return nullptr instead of
// silently pointing at whatever reused the slot.
struct PoolHandle {
    std::uint32_t index = 0;
    std::uint32_t generation = 0;

    friend bool operator==(const PoolHandle&, const PoolHandle&) = default;
};

template <typename T, std::size_t Capacity>
class ObjectPool {
    static_assert(Capacity > 0, "an empty pool is not useful");
    static_assert(Capacity <= UINT32_MAX, "indices are stored as uint32");

public:
    ObjectPool() { clear(); }

    // Pools hand out handles that refer to *this* pool's slots, so copying a
    // pool would create two owners of one set of handles. Forbid it.
    ObjectPool(const ObjectPool&) = delete;
    ObjectPool& operator=(const ObjectPool&) = delete;

    // Constructs a T in a free slot. Returns std::nullopt when the pool is
    // exhausted; the caller decides whether that means "drop the request"
    // (a bullet that is never fired) or a hard error.
    template <typename... Args>
    std::optional<PoolHandle> acquire(Args&&... args) {
        if (freeTop_ == 0) {
            ++exhaustedCount_;
            return std::nullopt;
        }
        const std::uint32_t index = freeStack_[--freeTop_];
        slots_[index].emplace(std::forward<Args>(args)...);
        denseIndex_[index] = static_cast<std::uint32_t>(liveCount_);
        dense_[liveCount_++] = index;
        if (liveCount_ > highWater_) highWater_ = liveCount_;
        return PoolHandle{index, generations_[index]};
    }

    // Destroys the object and returns its slot to the free list.
    // Returns false (and does nothing) for a stale or foreign handle.
    bool release(PoolHandle handle) {
        if (!isLive(handle)) return false;
        releaseSlot(handle.index);
        return true;
    }

    T* get(PoolHandle handle) { return isLive(handle) ? &*slots_[handle.index] : nullptr; }
    const T* get(PoolHandle handle) const {
        return isLive(handle) ? &*slots_[handle.index] : nullptr;
    }

    bool isLive(PoolHandle handle) const {
        return handle.index < Capacity && slots_[handle.index].has_value() &&
               generations_[handle.index] == handle.generation;
    }

    // Visits every live object. fn may read and mutate the object but must
    // not acquire or release; use removeIf for removal during iteration.
    template <typename Fn>
    void forEach(Fn&& fn) {
        for (std::size_t i = 0; i < liveCount_; ++i) fn(*slots_[dense_[i]]);
    }
    template <typename Fn>
    void forEach(Fn&& fn) const {
        for (std::size_t i = 0; i < liveCount_; ++i) fn(*slots_[dense_[i]]);
    }

    // Visits every live object and releases those for which pred returns
    // true. Walking the dense list backwards keeps swap-and-pop safe: the
    // element swapped into position i has already been visited.
    // Returns the number of objects released.
    template <typename Pred>
    std::size_t removeIf(Pred&& pred) {
        std::size_t removed = 0;
        for (std::size_t i = liveCount_; i-- > 0;) {
            const std::uint32_t index = dense_[i];
            if (pred(*slots_[index])) {
                releaseSlot(index);
                ++removed;
            }
        }
        return removed;
    }

    // Releases everything and resets statistics. Outstanding handles become
    // stale because every live slot's generation is bumped.
    void clear() {
        for (std::size_t i = 0; i < liveCount_; ++i) {
            const std::uint32_t index = dense_[i];
            slots_[index].reset();
            ++generations_[index];
        }
        liveCount_ = 0;
        // Fill the free stack so slot 0 is handed out first.
        for (std::size_t i = 0; i < Capacity; ++i) {
            freeStack_[i] = static_cast<std::uint32_t>(Capacity - 1 - i);
        }
        freeTop_ = Capacity;
        highWater_ = 0;
        exhaustedCount_ = 0;
    }

    static constexpr std::size_t capacity() { return Capacity; }
    std::size_t size() const { return liveCount_; }
    bool empty() const { return liveCount_ == 0; }
    bool full() const { return freeTop_ == 0; }

    // Tuning statistics: the most objects ever live at once, and how many
    // acquire calls failed. Shipping games log these to size pools.
    std::size_t highWater() const { return highWater_; }
    std::size_t exhaustedCount() const { return exhaustedCount_; }

private:
    void releaseSlot(std::uint32_t index) {
        slots_[index].reset();
        ++generations_[index];

        // Swap-and-pop: move the last live index into the hole.
        const std::uint32_t hole = denseIndex_[index];
        const std::uint32_t last = dense_[--liveCount_];
        dense_[hole] = last;
        denseIndex_[last] = hole;

        freeStack_[freeTop_++] = index;
    }

    std::array<std::optional<T>, Capacity> slots_{};
    std::array<std::uint32_t, Capacity> generations_{};
    std::array<std::uint32_t, Capacity> freeStack_{};
    std::array<std::uint32_t, Capacity> dense_{};
    std::array<std::uint32_t, Capacity> denseIndex_{};
    std::size_t freeTop_ = 0;
    std::size_t liveCount_ = 0;
    std::size_t highWater_ = 0;
    std::size_t exhaustedCount_ = 0;
};

}  // namespace invaders
