using SceneShell.Core.Input;
using SceneShell.Core.Randomness;
using SceneShell.Core.Rendering;
using SceneShell.Core.Scenes;

namespace SceneShell.Core.Arcade.Scenes;

/// <summary>
/// Survive the falling blocks. Blocks spawn faster and fall harder the longer you last;
/// the score is time survived. Transient: every round is a fresh instance, so "play again"
/// needs no reset code at all.
/// </summary>
public sealed class DodgeScene : Scene
{
    private readonly ISceneNavigator<SceneId> _navigator;
    private readonly ArcadeTransitions _transitions;
    private readonly ArcadeConfig _config;
    private readonly DodgeConfig _rules;
    private readonly IRandom _rng;
    private readonly ArcadeSession _session;
    private readonly List<Faller> _blocks = new();

    private Paddle _player = null!;
    private float _spawnTimer;

    public DodgeScene(
        ISceneNavigator<SceneId> navigator,
        ArcadeTransitions transitions,
        ArcadeConfig config,
        IRandom rng,
        ArcadeSession session)
    {
        _navigator = navigator;
        _transitions = transitions;
        _config = config;
        _rules = config.Dodge;
        _rng = rng;
        _session = session;
    }

    public float Elapsed { get; private set; }

    public int Score => (int)(Elapsed * _rules.PointsPerSecond);

    public bool IsOver { get; private set; }

    public Box Player => _player.Bounds;

    public IReadOnlyList<Faller> Blocks => _blocks;

    /// <summary>Current spawn interval: shrinks linearly with time survived down to the floor.</summary>
    public float SpawnInterval =>
        Math.Max(_rules.MinSpawnInterval, _rules.StartSpawnInterval - Elapsed * _rules.SpawnIntervalRamp);

    /// <summary>Base fall speed for new blocks: grows with time survived up to the cap.</summary>
    public float FallSpeed =>
        Math.Min(_rules.MaxFallSpeed, _rules.StartFallSpeed + Elapsed * _rules.FallAcceleration);

    protected internal override void Load()
    {
        var floorY = _config.Height - _rules.PlayerHeight - 8f;
        _player = new Paddle(_rules.PlayerWidth, _rules.PlayerHeight, _rules.PlayerSpeed, _config.Width, floorY);
        _spawnTimer = SpawnInterval;
    }

    protected internal override void Update(float dt, InputFrame input)
    {
        if (IsOver)
        {
            return;
        }
        if (input.WasPressed(GameAction.Back))
        {
            // Abandoning a round records nothing.
            IsOver = true;
            _navigator.ChangeTo(SceneId.Menu, _transitions.Fade);
            return;
        }

        Elapsed += dt;
        _player.Steer(dt, input);

        _spawnTimer -= dt;
        while (_spawnTimer <= 0f)
        {
            Spawn();
            _spawnTimer += SpawnInterval;
        }

        foreach (var block in _blocks)
        {
            block.Fall(dt);
        }
        _blocks.RemoveAll(block => block.Y > _config.Height);

        var player = _player.Bounds;
        if (_blocks.Any(block => block.Bounds.Overlaps(player)))
        {
            IsOver = true;
            _session.Record(ArcadeGame.Dodge, Score);
            _navigator.ChangeTo(SceneId.Results, _transitions.DeathFlash);
        }
    }

    private void Spawn()
    {
        var size = _rng.Range(_rules.MinBlockSize, _rules.MaxBlockSize);
        var x = _rng.Range(0f, _config.Width - size);
        // ±20% speed variety keeps the rain from falling in lockstep.
        var speed = FallSpeed * _rng.Range(0.8f, 1.2f);
        _blocks.Add(new Faller(x, -size, size, speed));
    }

    protected internal override void Draw(ICanvas canvas)
    {
        canvas.FillRect(0, 0, _config.Width, _config.Height, Palette.Background);
        canvas.FillRect(0, _player.Y + _player.Height, _config.Width, 1, Palette.Dim);
        foreach (var block in _blocks)
        {
            canvas.FillRect(block.X, block.Y, block.Size, block.Size, Palette.Danger);
        }
        var p = _player.Bounds;
        canvas.FillRect(p.X, p.Y, p.Width, p.Height, Palette.Accent);

        PixelFont.Draw(canvas, $"DODGE  SCORE {Score}", 6, 6, 1, Palette.Ink);
        var best = $"BEST {_session.BestFor(ArcadeGame.Dodge)}";
        PixelFont.Draw(canvas, best, _config.Width - 6 - PixelFont.Measure(best, 1), 6, 1, Palette.Gem);
    }
}
