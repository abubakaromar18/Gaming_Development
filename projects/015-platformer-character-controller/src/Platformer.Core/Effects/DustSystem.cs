using Platformer.Core.Physics;
using Platformer.Core.Randomness;

namespace Platformer.Core.Effects;

/// <summary>A single dust puff. Position and velocity are in world pixels.</summary>
public struct DustParticle
{
    public Vec2 Position;
    public Vec2 Velocity;
    public float Age;
    public float Lifetime;
    public bool Alive;

    /// <summary>0 when spawned, 1 when about to expire.</summary>
    public readonly float Progress => Lifetime > 0f ? Age / Lifetime : 1f;
}

/// <summary>
/// Take-off and landing dust from a fixed-size pool (see project 011): no allocation per
/// puff, and when the pool is full the oldest particle is recycled. Spread and speed come
/// from the injected <see cref="IRandom"/>, so a seeded run produces identical dust.
/// </summary>
public sealed class DustSystem
{
    private const float Drag = 6f;
    private const float Rise = -30f;

    private readonly DustParticle[] _pool;
    private readonly IRandom _random;
    private int _next;

    public DustSystem(int capacity, IRandom random)
    {
        if (capacity < 1)
        {
            throw new ArgumentOutOfRangeException(nameof(capacity), capacity, "Capacity must be at least 1.");
        }
        _pool = new DustParticle[capacity];
        _random = random ?? throw new ArgumentNullException(nameof(random));
    }

    public int Capacity => _pool.Length;
    public int AliveCount => _pool.Count(p => p.Alive);
    public ReadOnlySpan<DustParticle> Particles => _pool;

    /// <summary>Spawn <paramref name="count"/> puffs at <paramref name="origin"/>, thrown sideways at up to <paramref name="speed"/> px/s.</summary>
    public void Burst(Vec2 origin, int count, float speed)
    {
        ArgumentOutOfRangeException.ThrowIfNegative(count);
        for (var i = 0; i < count; i++)
        {
            ref var p = ref _pool[_next];
            _next = (_next + 1) % _pool.Length;
            var side = _random.NextFloat() < 0.5f ? -1f : 1f;
            p.Position = origin + new Vec2(_random.Range(-3f, 3f), 0f);
            p.Velocity = new Vec2(side * _random.Range(0.3f, 1f) * speed, _random.Range(-0.4f, 0f) * speed);
            p.Age = 0f;
            p.Lifetime = _random.Range(0.25f, 0.45f);
            p.Alive = true;
        }
    }

    public void Update(float dt)
    {
        var drag = MathF.Exp(-Drag * dt);
        for (var i = 0; i < _pool.Length; i++)
        {
            ref var p = ref _pool[i];
            if (!p.Alive)
            {
                continue;
            }
            p.Age += dt;
            if (p.Age >= p.Lifetime)
            {
                p.Alive = false;
                continue;
            }
            p.Velocity = new Vec2(p.Velocity.X * drag, p.Velocity.Y * drag + Rise * dt);
            p.Position += p.Velocity * dt;
        }
    }

    public void Clear()
    {
        Array.Clear(_pool);
        _next = 0;
    }
}
