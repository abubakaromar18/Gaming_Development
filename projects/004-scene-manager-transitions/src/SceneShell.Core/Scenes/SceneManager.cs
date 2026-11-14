using SceneShell.Core.Input;
using SceneShell.Core.Rendering;
using SceneShell.Core.Transitions;

namespace SceneShell.Core.Scenes;

/// <summary>
/// Owns the current scene and moves between scenes through a covered transition.
///
/// A change runs as a tiny state machine:
///
/// <code>
///   None ──ChangeTo──► Out ──coverage reaches 1──► (swap) ──► In ──coverage reaches 0──► None
///                      old scene drawn under          new scene drawn under
///                      a growing cover                a shrinking cover
/// </code>
///
/// The swap (old <c>Exit</c>/<c>Unload</c>, new <c>Load</c>/<c>Enter</c>) happens at the
/// moment the screen is fully covered, so the player never sees a half-built scene and a
/// slow <c>Load</c> hides behind an opaque frame. Input is withheld for the whole
/// transition, which removes double-confirm bugs and inputs landing on a scene that is
/// leaving.
///
/// Requests are deferred: <see cref="ChangeTo"/> only queues, and the manager starts the
/// change after the current call returns. A scene can therefore ask to leave from inside
/// its own <c>Update</c> or <c>Enter</c> without the manager tearing it down mid-call.
/// </summary>
public sealed class SceneManager<TId> : ISceneNavigator<TId> where TId : notnull
{
    /// <summary>
    /// Upper bound on scene swaps inside one <see cref="Update"/>. Zero-length transitions
    /// complete instantly, so two scenes that change to each other on entry would otherwise
    /// spin forever inside a single frame.
    /// </summary>
    public const int MaxSwapsPerUpdate = 16;

    private readonly SceneRegistry<TId> _registry;
    private readonly ITransition _defaultTransition;
    private readonly Dictionary<TId, Scene> _persistent = new();
    private readonly Queue<Request> _pending = new();

    private ITransition? _transition;
    private Request _active;
    private float _elapsed;
    private int _swapsThisUpdate;
    private TId _currentId = default!;

    private enum RequestKind { Change, Quit }

    private readonly record struct Request(RequestKind Kind, TId Target, ITransition Transition);

    public SceneManager(SceneRegistry<TId> registry, ITransition defaultTransition)
    {
        _registry = registry ?? throw new ArgumentNullException(nameof(registry));
        _defaultTransition = defaultTransition ?? throw new ArgumentNullException(nameof(defaultTransition));
    }

    /// <summary>The scene being updated and drawn, or null before start and after quit.</summary>
    public Scene? Current { get; private set; }

    /// <summary>Id of <see cref="Current"/>.</summary>
    /// <exception cref="InvalidOperationException">There is no current scene.</exception>
    public TId CurrentId => Current is not null
        ? _currentId
        : throw new InvalidOperationException("There is no current scene.");

    public TransitionPhase Phase { get; private set; } = TransitionPhase.None;

    public bool IsTransitioning => Phase != TransitionPhase.None;

    public bool HasStarted { get; private set; }

    /// <summary>True once a quit request has covered the screen and every scene was unloaded.</summary>
    public bool HasQuit { get; private set; }

    /// <summary>Requests waiting for the current transition to finish.</summary>
    public int PendingCount => _pending.Count;

    /// <summary>
    /// How much of the screen the transition covers: rises 0→1 during <see cref="TransitionPhase.Out"/>,
    /// falls 1→0 during <see cref="TransitionPhase.In"/>, and is 0 when idle.
    /// </summary>
    public float Coverage => Phase switch
    {
        TransitionPhase.Out => Progress(_elapsed, _transition!.OutDuration),
        TransitionPhase.In => 1f - Progress(_elapsed, _transition!.InDuration),
        _ => 0f,
    };

    /// <summary>Enter the first scene, revealing it with <paramref name="intro"/> (instant when null).</summary>
    public void Start(TId id, ITransition? intro = null)
    {
        if (HasStarted)
        {
            throw new InvalidOperationException("The scene manager has already started.");
        }
        EnsureRegistered(id);

        HasStarted = true;
        _swapsThisUpdate = 0;
        _transition = intro ?? CutTransition.Instance;
        _elapsed = 0f;
        EnterScene(id);
        Phase = TransitionPhase.In;
        Pump(0f);
    }

    public void ChangeTo(TId id, ITransition? transition = null)
    {
        EnsureAcceptingRequests();
        EnsureRegistered(id);
        _pending.Enqueue(new Request(RequestKind.Change, id, transition ?? _defaultTransition));
    }

    public void Quit(ITransition? transition = null)
    {
        EnsureAcceptingRequests();
        _pending.Enqueue(new Request(RequestKind.Quit, default!, transition ?? _defaultTransition));
    }

    /// <summary>
    /// Advance by <paramref name="dt"/> seconds. When idle, the current scene gets the time
    /// and the input. During a transition the scene gets the time only if it opted in, and
    /// never the input; the time drives the transition instead.
    /// </summary>
    public void Update(float dt, InputFrame input)
    {
        if (!HasStarted)
        {
            throw new InvalidOperationException("Call Start before Update.");
        }
        if (!float.IsFinite(dt) || dt < 0f)
        {
            throw new ArgumentOutOfRangeException(nameof(dt), dt, "dt must be a finite, non-negative number of seconds.");
        }
        if (HasQuit)
        {
            return;
        }

        _swapsThisUpdate = 0;
        if (Phase == TransitionPhase.None)
        {
            Current!.Update(dt, input);
            // The idle scene consumed this frame. A change it requested starts now at t = 0.
            Pump(0f);
        }
        else
        {
            if (Current!.UpdatesDuringTransition)
            {
                Current.Update(dt, InputFrame.None);
            }
            Pump(dt);
        }
    }

    /// <summary>Draw the current scene, then the transition cover on top.</summary>
    public void Draw(ICanvas canvas)
    {
        ArgumentNullException.ThrowIfNull(canvas);
        Current?.Draw(canvas);
        if (Phase != TransitionPhase.None)
        {
            _transition!.Draw(canvas, Coverage, Phase);
        }
    }

    /// <summary>
    /// Runs the transition state machine forward by <paramref name="time"/> seconds. It loops
    /// so a large step or a zero-length transition can pass through several phases (and even
    /// several queued requests) in one call.
    /// </summary>
    private void Pump(float time)
    {
        if (Phase != TransitionPhase.None)
        {
            _elapsed += time;
        }

        while (!HasQuit)
        {
            switch (Phase)
            {
                case TransitionPhase.None:
                    if (!_pending.TryDequeue(out var next))
                    {
                        return;
                    }
                    _active = next;
                    _transition = next.Transition;
                    _elapsed = 0f;
                    Phase = TransitionPhase.Out;
                    break;

                case TransitionPhase.Out:
                    if (_elapsed < _transition!.OutDuration)
                    {
                        return;
                    }
                    // Leftover time carries into the fade-in so long frames stay accurate.
                    _elapsed -= _transition.OutDuration;
                    Swap();
                    if (!HasQuit)
                    {
                        Phase = TransitionPhase.In;
                    }
                    break;

                case TransitionPhase.In:
                    if (_elapsed < _transition!.InDuration)
                    {
                        return;
                    }
                    Phase = TransitionPhase.None;
                    _transition = null;
                    _elapsed = 0f;
                    Current!.Revealed();
                    break;
            }
        }
    }

    /// <summary>The fully covered moment: retire the old scene and enter the new one.</summary>
    private void Swap()
    {
        if (++_swapsThisUpdate > MaxSwapsPerUpdate)
        {
            throw new InvalidOperationException(
                $"More than {MaxSwapsPerUpdate} scene changes in one update. " +
                "Two scenes are probably changing to each other with zero-length transitions.");
        }

        var leaving = Current!;
        leaving.Exit();
        if (!_persistent.ContainsKey(_currentId))
        {
            leaving.Unload();
        }
        Current = null;

        if (_active.Kind == RequestKind.Quit)
        {
            Shutdown();
            return;
        }

        EnterScene(_active.Target);
    }

    private void EnterScene(TId id)
    {
        var scene = Resolve(id);
        _currentId = id;
        Current = scene;
        scene.Enter();
    }

    private Scene Resolve(TId id)
    {
        var entry = _registry.Get(id);
        if (entry.Lifetime == SceneLifetime.Persistent && _persistent.TryGetValue(id, out var cached))
        {
            return cached;
        }

        var scene = entry.Factory()
            ?? throw new InvalidOperationException($"The factory for scene '{id}' returned null.");
        scene.Load();
        if (entry.Lifetime == SceneLifetime.Persistent)
        {
            _persistent.Add(id, scene);
        }
        return scene;
    }

    private void Shutdown()
    {
        foreach (var scene in _persistent.Values)
        {
            scene.Unload();
        }
        _persistent.Clear();
        _pending.Clear();
        _transition = null;
        Phase = TransitionPhase.None;
        HasQuit = true;
    }

    private void EnsureAcceptingRequests()
    {
        if (!HasStarted)
        {
            throw new InvalidOperationException("Call Start before requesting scene changes.");
        }
        if (HasQuit)
        {
            throw new InvalidOperationException("The scene manager has already quit.");
        }
    }

    private void EnsureRegistered(TId id)
    {
        if (!_registry.Contains(id))
        {
            throw new KeyNotFoundException($"Scene '{id}' is not registered.");
        }
    }

    private static float Progress(float elapsed, float duration) =>
        duration <= 0f ? 1f : Math.Clamp(elapsed / duration, 0f, 1f);
}
