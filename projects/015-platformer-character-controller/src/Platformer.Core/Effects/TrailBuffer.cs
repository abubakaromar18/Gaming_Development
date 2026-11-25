using Platformer.Core.Physics;

namespace Platformer.Core.Effects;

/// <summary>
/// A ring buffer of recent foot positions, drawn by the debug overlay as the jump arc.
/// Seeing the arc is the fastest way to tune a jump: changes to apex hang or fall gravity
/// show up as a visibly different curve.
/// </summary>
public sealed class TrailBuffer
{
    private readonly Vec2[] _points;
    private int _start;

    public TrailBuffer(int capacity)
    {
        if (capacity < 1)
        {
            throw new ArgumentOutOfRangeException(nameof(capacity), capacity, "Capacity must be at least 1.");
        }
        _points = new Vec2[capacity];
    }

    public int Capacity => _points.Length;
    public int Count { get; private set; }

    /// <summary>Oldest first: index 0 is the oldest retained point.</summary>
    public Vec2 this[int index]
    {
        get
        {
            if ((uint)index >= (uint)Count)
            {
                throw new ArgumentOutOfRangeException(nameof(index), index, $"Trail has {Count} points.");
            }
            return _points[(_start + index) % _points.Length];
        }
    }

    public void Add(Vec2 point)
    {
        if (Count < _points.Length)
        {
            _points[(_start + Count) % _points.Length] = point;
            Count++;
        }
        else
        {
            _points[_start] = point;
            _start = (_start + 1) % _points.Length;
        }
    }

    public void Clear()
    {
        _start = 0;
        Count = 0;
    }
}
