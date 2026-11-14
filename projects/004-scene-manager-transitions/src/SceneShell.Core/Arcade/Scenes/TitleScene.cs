using SceneShell.Core.Input;
using SceneShell.Core.Randomness;
using SceneShell.Core.Rendering;
using SceneShell.Core.Scenes;

namespace SceneShell.Core.Arcade.Scenes;

/// <summary>
/// Title card over a scrolling starfield. It demonstrates two lifecycle details:
/// the stars keep moving while a fade covers the screen (<see cref="UpdatesDuringTransition"/>),
/// and the "PRESS ENTER" prompt only appears once <see cref="Revealed"/> says input is live.
/// </summary>
public sealed class TitleScene : Scene
{
    private const float BlinkPeriod = 1f;

    private readonly ISceneNavigator<SceneId> _navigator;
    private readonly ArcadeTransitions _transitions;
    private readonly ArcadeConfig _config;
    private readonly IRandom _rng;
    private readonly List<Star> _stars = new();

    private readonly record struct Star(float X, float Y, float Speed, byte Brightness);

    public TitleScene(ISceneNavigator<SceneId> navigator, ArcadeTransitions transitions, ArcadeConfig config, IRandom rng)
    {
        _navigator = navigator;
        _transitions = transitions;
        _config = config;
        _rng = rng;
    }

    public override bool UpdatesDuringTransition => true;

    /// <summary>Seconds of animation since the scene was entered.</summary>
    public float AnimationTime { get; private set; }

    public bool IsInteractive { get; private set; }

    public bool IsPromptVisible => IsInteractive && AnimationTime % BlinkPeriod < BlinkPeriod * 0.6f;

    public IReadOnlyList<(float X, float Y)> StarPositions => _stars.Select(s => (s.X, s.Y)).ToList();

    protected internal override void Load()
    {
        for (var i = 0; i < _config.StarCount; i++)
        {
            _stars.Add(new Star(
                _rng.Range(0, _config.Width),
                _rng.Range(0, _config.Height),
                _rng.Range(6f, 30f),
                (byte)_rng.Range(70, 230)));
        }
    }

    protected internal override void Enter()
    {
        AnimationTime = 0f;
        IsInteractive = false;
    }

    protected internal override void Revealed() => IsInteractive = true;

    protected internal override void Exit() => IsInteractive = false;

    protected internal override void Unload() => _stars.Clear();

    protected internal override void Update(float dt, InputFrame input)
    {
        AnimationTime += dt;
        for (var i = 0; i < _stars.Count; i++)
        {
            var star = _stars[i];
            var y = (star.Y + star.Speed * dt) % _config.Height;
            _stars[i] = star with { Y = y };
        }

        if (input.WasPressed(GameAction.Confirm))
        {
            _navigator.ChangeTo(SceneId.Menu, _transitions.Fade);
        }
        else if (input.WasPressed(GameAction.Back))
        {
            _navigator.Quit(_transitions.Fade);
        }
    }

    protected internal override void Draw(ICanvas canvas)
    {
        canvas.FillRect(0, 0, _config.Width, _config.Height, Palette.Background);
        foreach (var star in _stars)
        {
            canvas.FillRect(star.X, star.Y, 1, 1, new Rgba(star.Brightness, star.Brightness, star.Brightness));
        }

        var centre = _config.Width / 2f;
        PixelFont.DrawCentered(canvas, "SCENE SHELL", centre, 44, 4, Palette.Accent);
        PixelFont.DrawCentered(canvas, "AN ARCADE OF SCENES AND TRANSITIONS", centre, 76, 1, Palette.Dim);
        if (IsPromptVisible)
        {
            PixelFont.DrawCentered(canvas, "PRESS ENTER", centre, 120, 2, Palette.Ink);
        }
        PixelFont.DrawCentered(canvas, "ESC: QUIT", centre, 164, 1, Palette.Dim);
    }
}
