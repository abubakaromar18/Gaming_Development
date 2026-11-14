using SceneShell.Core.Input;

namespace SceneShell.Core.Arcade;

/// <summary>An axis-aligned rectangle in logical pixels.</summary>
public readonly record struct Box(float X, float Y, float Width, float Height)
{
    public float Right => X + Width;
    public float Bottom => Y + Height;

    /// <summary>Strict overlap: boxes that only touch edges do not count.</summary>
    public bool Overlaps(Box other) =>
        X < other.Right && other.X < Right && Y < other.Bottom && other.Y < Bottom;
}

/// <summary>Something falling at a constant speed: a block in Dodge, a gem in Collect.</summary>
public sealed class Faller
{
    public Faller(float x, float y, float size, float speed)
    {
        X = x;
        Y = y;
        Size = size;
        Speed = speed;
    }

    public float X { get; }
    public float Y { get; private set; }
    public float Size { get; }
    public float Speed { get; }

    public Box Bounds => new(X, Y, Size, Size);

    public void Fall(float dt) => Y += Speed * dt;
}

/// <summary>A horizontally moving player body that stays inside the screen.</summary>
public sealed class Paddle
{
    private readonly float _worldWidth;

    public Paddle(float width, float height, float speed, float worldWidth, float y)
    {
        Width = width;
        Height = height;
        Speed = speed;
        _worldWidth = worldWidth;
        Y = y;
        X = (worldWidth - width) / 2f;
    }

    public float X { get; private set; }
    public float Y { get; }
    public float Width { get; }
    public float Height { get; }
    public float Speed { get; }

    public Box Bounds => new(X, Y, Width, Height);

    /// <summary>Move by held Left/Right. Holding both cancels out.</summary>
    public void Steer(float dt, InputFrame input)
    {
        var direction = (input.IsHeld(GameAction.Right) ? 1f : 0f) - (input.IsHeld(GameAction.Left) ? 1f : 0f);
        X = Math.Clamp(X + direction * Speed * dt, 0f, _worldWidth - Width);
    }
}
