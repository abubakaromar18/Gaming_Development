namespace Platformer.Core.Randomness;

/// <summary>The randomness the game logic is allowed to use. Injected so tests can seed it.</summary>
public interface IRandom
{
    /// <summary>A uniformly distributed 32-bit value.</summary>
    uint NextUInt();

    /// <summary>A float in [0, 1).</summary>
    float NextFloat();

    /// <summary>An integer in [0, <paramref name="maxExclusive"/>), without modulo bias.</summary>
    int NextInt(int maxExclusive);

    /// <summary>A float in [<paramref name="min"/>, <paramref name="max"/>).</summary>
    float Range(float min, float max);
}

/// <summary>
/// PCG32 (O'Neill, pcg-random.org): a 64-bit LCG whose output is scrambled by an
/// xorshift and a data-dependent rotation. It is small, fast, statistically strong, and,
/// unlike <see cref="System.Random"/>, its sequence for a given seed is specified, so a
/// seed reproduces the same game on every .NET version and platform.
/// </summary>
public sealed class Pcg32 : IRandom
{
    private const ulong Multiplier = 6364136223846793005UL;

    private ulong _state;
    private readonly ulong _increment;

    /// <param name="seed">Starting point of the sequence.</param>
    /// <param name="stream">Selects one of 2^63 independent sequences.</param>
    public Pcg32(ulong seed, ulong stream = 54)
    {
        // Reference initialisation from pcg32_srandom_r.
        _increment = (stream << 1) | 1UL;
        _state = 0;
        NextUInt();
        _state += seed;
        NextUInt();
    }

    public uint NextUInt()
    {
        var old = _state;
        _state = unchecked(old * Multiplier + _increment);
        var xorShifted = (uint)(((old >> 18) ^ old) >> 27);
        var rotation = (int)(old >> 59);
        return (xorShifted >> rotation) | (xorShifted << (-rotation & 31));
    }

    public float NextFloat() => (NextUInt() >> 8) * (1f / (1 << 24));

    public int NextInt(int maxExclusive)
    {
        if (maxExclusive <= 0)
        {
            throw new ArgumentOutOfRangeException(nameof(maxExclusive), maxExclusive, "Must be positive.");
        }
        // Reject the few low values that would make some results more likely than others.
        var bound = (uint)maxExclusive;
        var threshold = unchecked(0u - bound) % bound;
        while (true)
        {
            var value = NextUInt();
            if (value >= threshold)
            {
                return (int)(value % bound);
            }
        }
    }

    public float Range(float min, float max)
    {
        if (!(max >= min))
        {
            throw new ArgumentOutOfRangeException(nameof(max), max, "max must be >= min.");
        }
        return min + NextFloat() * (max - min);
    }
}
