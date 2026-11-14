using SceneShell.Core.Input;
using SceneShell.Core.Rendering;
using SceneShell.Core.Scenes;

namespace SceneShell.Core.Arcade.Scenes;

/// <summary>
/// The game picker. Registered as <see cref="SceneLifetime.Persistent"/>, so the same
/// instance is re-entered after every round and the cursor stays on the game you last
/// played. It counts its visits to make the persistent lifecycle visible.
/// </summary>
public sealed class MenuScene : Scene
{
    private readonly ISceneNavigator<SceneId> _navigator;
    private readonly ArcadeTransitions _transitions;
    private readonly ArcadeConfig _config;
    private readonly ArcadeSession _session;

    public MenuScene(ISceneNavigator<SceneId> navigator, ArcadeTransitions transitions, ArcadeConfig config, ArcadeSession session)
    {
        _navigator = navigator;
        _transitions = transitions;
        _config = config;
        _session = session;
    }

    public enum Item
    {
        Dodge,
        Collect,
        Quit,
    }

    public static IReadOnlyList<Item> Items { get; } = Enum.GetValues<Item>();

    public int Cursor { get; private set; }

    public Item Selected => Items[Cursor];

    /// <summary>How many times this instance has been entered.</summary>
    public int Visits { get; private set; }

    protected internal override void Enter() => Visits++;

    protected internal override void Update(float dt, InputFrame input)
    {
        if (input.WasPressed(GameAction.Up))
        {
            Cursor = (Cursor + Items.Count - 1) % Items.Count;
        }
        if (input.WasPressed(GameAction.Down))
        {
            Cursor = (Cursor + 1) % Items.Count;
        }

        if (input.WasPressed(GameAction.Confirm))
        {
            switch (Selected)
            {
                case Item.Dodge:
                    _navigator.ChangeTo(SceneId.Dodge, _transitions.Wipe);
                    break;
                case Item.Collect:
                    _navigator.ChangeTo(SceneId.Collect, _transitions.Wipe);
                    break;
                case Item.Quit:
                    _navigator.Quit(_transitions.Fade);
                    break;
            }
        }
        else if (input.WasPressed(GameAction.Back))
        {
            _navigator.ChangeTo(SceneId.Title, _transitions.Fade);
        }
    }

    protected internal override void Draw(ICanvas canvas)
    {
        canvas.FillRect(0, 0, _config.Width, _config.Height, Palette.Background);
        var centre = _config.Width / 2f;
        PixelFont.DrawCentered(canvas, "CHOOSE A GAME", centre, 24, 2, Palette.Accent);

        for (var i = 0; i < Items.Count; i++)
        {
            var item = Items[i];
            var y = 64 + i * 22;
            var selected = i == Cursor;
            var label = item.ToString().ToUpperInvariant();
            if (selected)
            {
                canvas.FillRect(centre - 70, y - 4, 140, 18, Palette.Accent.WithAlpha(0.18f));
                label = $"> {label} <";
            }
            PixelFont.DrawCentered(canvas, label, centre, y, 2, selected ? Palette.Ink : Palette.Dim);
            var best = BestLabel(item);
            if (best is not null)
            {
                PixelFont.Draw(canvas, best, centre + 74, y + 4, 1, Palette.Gem);
            }
        }

        PixelFont.DrawCentered(canvas, $"UP/DOWN  ENTER: PLAY  ESC: TITLE    VISITS {Visits}", centre, 164, 1, Palette.Dim);
    }

    private string? BestLabel(Item item)
    {
        var best = item switch
        {
            Item.Dodge => _session.BestFor(ArcadeGame.Dodge),
            Item.Collect => _session.BestFor(ArcadeGame.Collect),
            _ => 0,
        };
        return best > 0 ? $"BEST {best}" : null;
    }
}
