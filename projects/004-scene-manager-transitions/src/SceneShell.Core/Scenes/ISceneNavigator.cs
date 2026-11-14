using SceneShell.Core.Transitions;

namespace SceneShell.Core.Scenes;

/// <summary>
/// The narrow interface scenes use to leave. Scenes depend on this rather than on the
/// whole <see cref="SceneManager{TId}"/>, so they cannot drive another scene's lifecycle.
/// </summary>
public interface ISceneNavigator<in TId> where TId : notnull
{
    /// <summary>
    /// Queue a change to <paramref name="id"/>. The change starts after the current update
    /// returns (or after a running transition finishes), never in the middle of a call.
    /// </summary>
    /// <param name="transition">How to cover the swap; null uses the manager's default.</param>
    void ChangeTo(TId id, ITransition? transition = null);

    /// <summary>Queue a shutdown: cover the screen, then exit and unload every scene.</summary>
    void Quit(ITransition? transition = null);
}
