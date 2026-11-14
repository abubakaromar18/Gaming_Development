namespace SceneShell.Core.Scenes;

/// <summary>How long a scene instance lives.</summary>
public enum SceneLifetime
{
    /// <summary>A fresh instance per visit; loaded on entry, unloaded on exit.</summary>
    Transient,

    /// <summary>One instance kept for the manager's lifetime, so it remembers its state between visits.</summary>
    Persistent,
}

/// <summary>
/// Maps scene ids to factories. Scenes are created lazily, which keeps start-up cheap and
/// means a scene's constructor runs only when the player actually goes there.
/// </summary>
public sealed class SceneRegistry<TId> where TId : notnull
{
    private readonly Dictionary<TId, Entry> _entries = new();

    internal readonly record struct Entry(Func<Scene> Factory, SceneLifetime Lifetime);

    public SceneRegistry<TId> Register(TId id, Func<Scene> factory, SceneLifetime lifetime = SceneLifetime.Transient)
    {
        ArgumentNullException.ThrowIfNull(factory);
        if (!_entries.TryAdd(id, new Entry(factory, lifetime)))
        {
            throw new ArgumentException($"Scene '{id}' is already registered.", nameof(id));
        }
        return this;
    }

    public bool Contains(TId id) => _entries.ContainsKey(id);

    internal Entry Get(TId id) =>
        _entries.TryGetValue(id, out var entry)
            ? entry
            : throw new KeyNotFoundException($"Scene '{id}' is not registered.");
}
