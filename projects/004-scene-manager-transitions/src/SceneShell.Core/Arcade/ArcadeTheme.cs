using SceneShell.Core.Rendering;
using SceneShell.Core.Transitions;

namespace SceneShell.Core.Arcade;

/// <summary>The arcade's colors.</summary>
public static class Palette
{
    public static readonly Rgba Background = new(14, 16, 32);
    public static readonly Rgba Ink = new(232, 234, 246);
    public static readonly Rgba Dim = new(110, 116, 150);
    public static readonly Rgba Accent = new(92, 214, 255);
    public static readonly Rgba Gem = new(255, 210, 80);
    public static readonly Rgba Danger = new(255, 84, 104);
    public static readonly Rgba Flash = new(150, 20, 40);
}

/// <summary>
/// The transitions the scenes choose from. Built once from config and shared, which is
/// safe because transitions are stateless.
/// </summary>
public sealed class ArcadeTransitions
{
    public ArcadeTransitions(ArcadeConfig config)
    {
        ArgumentNullException.ThrowIfNull(config);
        Fade = new FadeTransition(Rgba.Black, config.FadeSeconds, config.FadeSeconds);
        Wipe = new WipeTransition(Palette.Accent, config.WipeSeconds, config.WipeSeconds);
        // A fast dip to dark red when the player dies, then a slower reveal of the results.
        DeathFlash = new FadeTransition(Palette.Flash, config.FadeSeconds * 0.5f, config.FadeSeconds * 1.5f);
    }

    /// <summary>Between menus.</summary>
    public ITransition Fade { get; }

    /// <summary>Into and between games.</summary>
    public ITransition Wipe { get; }

    /// <summary>From a lost round to the results card.</summary>
    public ITransition DeathFlash { get; }
}
