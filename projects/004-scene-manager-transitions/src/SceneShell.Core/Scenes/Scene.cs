using SceneShell.Core.Input;
using SceneShell.Core.Rendering;

namespace SceneShell.Core.Scenes;

/// <summary>
/// Base class for one screen of the game (title, menu, a level, a results card).
///
/// The <see cref="SceneManager{TId}"/> drives every hook; a scene never calls its own
/// lifecycle methods. The order for one visit is always:
///
/// <code>
///   Load*  →  Enter  →  Revealed  →  Update/Draw …  →  Exit  →  Unload*
/// </code>
///
/// * <c>Load</c> and <c>Unload</c> run once per instance. For a transient scene that is
///   once per visit; a persistent scene is loaded on its first visit and unloaded only
///   when the manager shuts down, while <c>Enter</c>/<c>Exit</c> run on every visit.
///
/// All hooks are optional. Only <see cref="Draw"/> is abstract, because a scene that
/// draws nothing would leave the transition fading over an empty frame.
/// </summary>
public abstract class Scene
{
    /// <summary>
    /// When true, the scene keeps simulating while a transition covers it (for example a
    /// title screen whose background keeps scrolling during the fade). Input is never
    /// delivered during a transition either way: the player cannot act on a screen they
    /// cannot fully see.
    /// </summary>
    public virtual bool UpdatesDuringTransition => false;

    /// <summary>Allocate resources. Runs once per instance, before the first <see cref="Enter"/>.</summary>
    protected internal virtual void Load() { }

    /// <summary>The scene just became current. A transition is usually still covering it.</summary>
    protected internal virtual void Enter() { }

    /// <summary>The transition finished. Input flows to the scene from the next update on.</summary>
    protected internal virtual void Revealed() { }

    /// <summary>Advance the scene by <paramref name="dt"/> seconds.</summary>
    protected internal virtual void Update(float dt, InputFrame input) { }

    /// <summary>The scene is fully covered and about to stop being current.</summary>
    protected internal virtual void Exit() { }

    /// <summary>Release resources. Runs once per instance, after its last <see cref="Exit"/>.</summary>
    protected internal virtual void Unload() { }

    /// <summary>Draw the scene. Transitions are drawn on top by the manager.</summary>
    protected internal abstract void Draw(ICanvas canvas);
}
