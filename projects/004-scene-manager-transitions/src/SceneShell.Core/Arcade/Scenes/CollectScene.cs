using SceneShell.Core.Input;
using SceneShell.Core.Randomness;
using SceneShell.Core.Rendering;
using SceneShell.Core.Scenes;

namespace SceneShell.Core.Arcade.Scenes;

/// <summary>
/// Catch gems before the clock runs out. Letting <see cref="CollectConfig.MaxMisses"/>
/// gems hit the floor ends the round early. The score is the number of gems caught.
/// </summary>
public sealed class CollectScene : Scene
{
    private readonly ISceneNavigator<SceneId> _navigator;
    private readonly ArcadeTransitions _transitions;
    private readonly ArcadeConfig _config;
    private readonly CollectConfig _rules;
    private readonly IRandom _rng;
    private readonly ArcadeSession _session;
    private readonly List<Faller> _gems = new();

    private Paddle _basket = null!;
    private float _spawnTimer;

    public CollectScene(
        ISceneNavigator<SceneId> navigator,
        ArcadeTransitions transitions,
        ArcadeConfig config,
        IRandom rng,
        ArcadeSession session)
    {
        _navigator = navigator;
        _transitions = transitions;
        _config = config;
        _rules = config.Collect;
        _rng = rng;
        _session = session;
    }

    public float TimeLeft { get; private set; }

    public int Caught { get; private set; }

    public int Missed { get; private set; }

    public bool IsOver { get; private set; }

    public Box Basket => _basket.Bounds;

    public IReadOnlyList<Faller> Gems => _gems;

    protected internal override void Load()
    {
        var floorY = _config.Height - _rules.BasketHeight - 8f;
        _basket = new Paddle(_rules.BasketWidth, _rules.BasketHeight, _rules.BasketSpeed, _config.Width, floorY);
        TimeLeft = _rules.RoundSeconds;
        // The first gem arrives straight away so the round never opens on an empty screen.
        _spawnTimer = 0f;
    }

    protected internal override void Update(float dt, InputFrame input)
    {
        if (IsOver)
        {
            return;
        }
        if (input.WasPressed(GameAction.Back))
        {
            IsOver = true;
            _navigator.ChangeTo(SceneId.Menu, _transitions.Fade);
            return;
        }

        TimeLeft = Math.Max(0f, TimeLeft - dt);
        _basket.Steer(dt, input);

        _spawnTimer -= dt;
        while (_spawnTimer <= 0f)
        {
            Spawn();
            _spawnTimer += _rules.SpawnInterval;
        }

        var basket = _basket.Bounds;
        foreach (var gem in _gems)
        {
            gem.Fall(dt);
        }
        Caught += _gems.RemoveAll(gem => gem.Bounds.Overlaps(basket));
        Missed += _gems.RemoveAll(gem => gem.Y >= _config.Height);

        if (Missed >= _rules.MaxMisses || TimeLeft <= 0f)
        {
            IsOver = true;
            _session.Record(ArcadeGame.Collect, Caught);
            _navigator.ChangeTo(SceneId.Results, _transitions.Fade);
        }
    }

    private void Spawn()
    {
        var x = _rng.Range(0f, _config.Width - _rules.GemSize);
        var speed = _rng.Range(_rules.MinFallSpeed, _rules.MaxFallSpeed);
        _gems.Add(new Faller(x, -_rules.GemSize, _rules.GemSize, speed));
    }

    protected internal override void Draw(ICanvas canvas)
    {
        canvas.FillRect(0, 0, _config.Width, _config.Height, Palette.Background);
        foreach (var gem in _gems)
        {
            // A small plus-shaped sparkle around a solid core.
            var half = gem.Size / 2f;
            canvas.FillRect(gem.X, gem.Y, gem.Size, gem.Size, Palette.Gem);
            canvas.FillRect(gem.X + half - 0.5f, gem.Y - 2, 1, gem.Size + 4, Palette.Gem.WithAlpha(0.5f));
            canvas.FillRect(gem.X - 2, gem.Y + half - 0.5f, gem.Size + 4, 1, Palette.Gem.WithAlpha(0.5f));
        }
        var b = _basket.Bounds;
        canvas.FillRect(b.X, b.Y, b.Width, b.Height, Palette.Accent);
        canvas.FillRect(b.X, b.Y - 4, 2, 4, Palette.Accent);
        canvas.FillRect(b.Right - 2, b.Y - 4, 2, 4, Palette.Accent);

        PixelFont.Draw(canvas, $"COLLECT  GEMS {Caught}", 6, 6, 1, Palette.Ink);
        var clock = $"TIME {MathF.Ceiling(TimeLeft):0}";
        PixelFont.DrawCentered(canvas, clock, _config.Width / 2f, 6, 1, Palette.Ink);
        for (var i = 0; i < _rules.MaxMisses; i++)
        {
            var color = i < Missed ? Palette.Danger : Palette.Dim;
            canvas.FillRect(_config.Width - 10 - i * 8, 6, 5, 5, color);
        }
    }
}
