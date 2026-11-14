namespace SceneShell.Core.Arcade;

/// <summary>The scenes of the arcade shell.</summary>
public enum SceneId
{
    Title,
    Menu,
    Dodge,
    Collect,
    Results,
}

/// <summary>The playable mini-games.</summary>
public enum ArcadeGame
{
    Dodge,
    Collect,
}

public readonly record struct GameResult(ArcadeGame Game, int Score, bool IsNewBest);

/// <summary>
/// State that outlives any single scene: the last result and the best score per game.
/// Scenes pass data through this shared object instead of through each other, so the
/// results screen does not need to know which scene sent the player there.
/// </summary>
public sealed class ArcadeSession
{
    private readonly Dictionary<ArcadeGame, int> _best = new();

    public GameResult? LastResult { get; private set; }

    public int BestFor(ArcadeGame game) => _best.GetValueOrDefault(game);

    public GameResult Record(ArcadeGame game, int score)
    {
        ArgumentOutOfRangeException.ThrowIfNegative(score);
        var isNewBest = score > BestFor(game);
        if (isNewBest)
        {
            _best[game] = score;
        }
        var result = new GameResult(game, score, isNewBest);
        LastResult = result;
        return result;
    }

    public static SceneId SceneFor(ArcadeGame game) => game switch
    {
        ArcadeGame.Dodge => SceneId.Dodge,
        ArcadeGame.Collect => SceneId.Collect,
        _ => throw new ArgumentOutOfRangeException(nameof(game), game, "Unknown game."),
    };
}
