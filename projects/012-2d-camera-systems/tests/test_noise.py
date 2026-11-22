import random

from camera2d.core import Noise1D


def test_same_seed_same_noise():
    a, b = Noise1D(random.Random(7)), Noise1D(random.Random(7))
    xs = [i * 0.137 for i in range(500)]
    assert [a.sample(x) for x in xs] == [b.sample(x) for x in xs]


def test_different_seeds_differ():
    a, b = Noise1D(random.Random(1)), Noise1D(random.Random(2))
    assert any(a.sample(i * 0.37) != b.sample(i * 0.37) for i in range(50))


def test_range_and_zero_at_lattice_points():
    n = Noise1D(random.Random(3))
    samples = [n.sample(i * 0.013) for i in range(40_000)]
    assert all(-1.0 <= s <= 1.0 for s in samples)
    assert max(samples) - min(samples) > 0.5  # actually varies
    assert all(n.sample(float(i)) == 0.0 for i in range(-20, 300))


def test_noise_is_continuous():
    # Coherent noise: tiny input steps give tiny output steps (unlike random()).
    n = Noise1D(random.Random(4))
    step = 1e-3
    worst = max(abs(n.sample(x * step + step) - n.sample(x * step)) for x in range(20_000))
    assert worst < 0.01


def test_wraps_and_handles_negative_input():
    n = Noise1D(random.Random(5))
    assert n.sample(3.25) == n.sample(3.25 + 256)
    assert -1.0 <= n.sample(-12.6) <= 1.0
