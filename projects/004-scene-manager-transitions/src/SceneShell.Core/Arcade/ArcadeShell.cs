using SceneShell.Core.Arcade.Scenes;
using SceneShell.Core.Input;
using SceneShell.Core.Randomness;
using SceneShell.Core.Rendering;
using SceneShell.Core.Scenes;

namespace SceneShell.Core.Arcade;

/// <summary>
/// Composition root: builds the session, transitions, scene registry and manager, then
/// starts on the title screen. The MonoGame host and the tests both construct this, so
/// they run exactly the same game; the host only feeds it time and input and draws it.
/// </summary>
public sealed class ArcadeShell
{
    private readonly ArcadeConfig _config;

    public ArcadeShell(ArcadeConfig config, IRandom rng)
    {
        ArgumentNullException.ThrowIfNull(config);
        ArgumentNullException.ThrowIfNull(rng);
        config.Validate();
        _config = config;

        Session = new ArcadeSession();
        Transitions = new ArcadeTransitions(config);

        // The manager exists before the factories that capture it; the registry is only
        // consulted from Start onwards, so registering after construction is safe.
        var registry = new SceneRegistry<SceneId>();
        Scenes = new SceneManager<SceneId>(registry, Transitions.Fade);
        var navigator = Scenes;
        registry
            .Register(SceneId.Title, () => new TitleScene(navigator, Transitions, config, rng))
            .Register(SceneId.Menu, () => new MenuScene(navigator, Transitions, config, Session), SceneLifetime.Persistent)
            .Register(SceneId.Dodge, () => new DodgeScene(navigator, Transitions, config, rng, Session))
            .Register(SceneId.Collect, () => new CollectScene(navigator, Transitions, config, rng, Session))
            .Register(SceneId.Results, () => new ResultsScene(navigator, Transitions, config, Session));

        Scenes.Start(SceneId.Title, Transitions.Fade);
    }

    public SceneManager<SceneId> Scenes { get; }

    public ArcadeSession Session { get; }

    public ArcadeTransitions Transitions { get; }

    public bool ShowDebug { get; private set; }

    public bool HasQuit => Scenes.HasQuit;

    /// <summary>
    /// Advance one rendered frame. <paramref name="dt"/> is clamped to
    /// <see cref="ArcadeConfig.MaxFrameSeconds"/> so a window drag or breakpoint does not
    /// teleport blocks through the player.
    /// </summary>
    public void Update(float dt, InputFrame input)
    {
        if (input.WasPressed(GameAction.ToggleDebug))
        {
            ShowDebug = !ShowDebug;
        }
        Scenes.Update(Math.Clamp(dt, 0f, _config.MaxFrameSeconds), input);
    }

    public void Draw(ICanvas canvas)
    {
        ArgumentNullException.ThrowIfNull(canvas);
        canvas.FillRect(0, 0, canvas.Width, canvas.Height, Rgba.Black);
        Scenes.Draw(canvas);
        if (ShowDebug && !HasQuit)
        {
            DrawDebug(canvas);
        }
    }

    /// <summary>A one-line readout of the manager's state machine (toggle with F3).</summary>
    private void DrawDebug(ICanvas canvas)
    {
        var line = FormattableString
            .Invariant($"SCENE {Scenes.CurrentId}  PHASE {Scenes.Phase}  COVER {Scenes.Coverage:0.00}  QUEUE {Scenes.PendingCount}")
            .ToUpperInvariant();
        canvas.FillRect(0, canvas.Height - 9, canvas.Width, 9, Rgba.Black.WithAlpha(0.7f));
        PixelFont.Draw(canvas, line, 3, canvas.Height - 7, 1, Palette.Gem);
    }
}
