using SceneShell.Core.Input;
using SceneShell.Core.Rendering;
using SceneShell.Core.Scenes;

namespace SceneShell.Core.Arcade.Scenes;

/// <summary>
/// The round-over card. It reads the result from <see cref="ArcadeSession"/> on
/// <see cref="Enter"/>, so it works for any game without knowing who sent the player here.
/// </summary>
public sealed class ResultsScene : Scene
{
    private readonly ISceneNavigator<SceneId> _navigator;
    private readonly ArcadeTransitions _transitions;
    private readonly ArcadeConfig _config;
    private readonly ArcadeSession _session;

    public ResultsScene(ISceneNavigator<SceneId> navigator, ArcadeTransitions transitions, ArcadeConfig config, ArcadeSession session)
    {
        _navigator = navigator;
        _transitions = transitions;
        _config = config;
        _session = session;
    }

    public GameResult Result { get; private set; }

    protected internal override void Enter() =>
        Result = _session.LastResult
            ?? throw new InvalidOperationException("The results scene was entered before any round was recorded.");

    protected internal override void Update(float dt, InputFrame input)
    {
        if (input.WasPressed(GameAction.Confirm))
        {
            _navigator.ChangeTo(ArcadeSession.SceneFor(Result.Game), _transitions.Wipe);
        }
        else if (input.WasPressed(GameAction.Back))
        {
            _navigator.ChangeTo(SceneId.Menu, _transitions.Fade);
        }
    }

    protected internal override void Draw(ICanvas canvas)
    {
        canvas.FillRect(0, 0, _config.Width, _config.Height, Palette.Background);
        var centre = _config.Width / 2f;
        PixelFont.DrawCentered(canvas, $"{Result.Game.ToString().ToUpperInvariant()} OVER", centre, 30, 3, Palette.Danger);
        PixelFont.DrawCentered(canvas, $"SCORE {Result.Score}", centre, 74, 3, Palette.Ink);
        var bestLine = Result.IsNewBest ? "NEW BEST!" : $"BEST {_session.BestFor(Result.Game)}";
        PixelFont.DrawCentered(canvas, bestLine, centre, 106, 2, Palette.Gem);
        PixelFont.DrawCentered(canvas, "ENTER: PLAY AGAIN   ESC: MENU", centre, 150, 1, Palette.Dim);
    }
}
