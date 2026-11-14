using SceneShell.Core.Input;
using SceneShell.Core.Rendering;
using SceneShell.Core.Scenes;
using SceneShell.Core.Transitions;

namespace SceneShell.Core.Tests;

/// <summary>Records every FillRect call so tests can assert what was drawn.</summary>
public sealed class RecordingCanvas : ICanvas
{
    public RecordingCanvas(int width = 320, int height = 180)
    {
        Width = width;
        Height = height;
    }

    public int Width { get; }
    public int Height { get; }

    public List<(float X, float Y, float W, float H, Rgba Color)> Rects { get; } = new();

    public void FillRect(float x, float y, float width, float height, Rgba color) =>
        Rects.Add((x, y, width, height, color));
}

/// <summary>A scene that writes each lifecycle call to a shared log as "name.Hook".</summary>
public class RecordingScene : Scene
{
    private readonly List<string> _log;
    private readonly bool _updatesDuringTransition;

    public RecordingScene(string name, List<string> log, bool updatesDuringTransition = false)
    {
        Name = name;
        _log = log;
        _updatesDuringTransition = updatesDuringTransition;
    }

    public string Name { get; }

    public override bool UpdatesDuringTransition => _updatesDuringTransition;

    public List<InputFrame> Inputs { get; } = new();

    /// <summary>Runs inside Update, after logging, so tests can request changes mid-call.</summary>
    public Action<RecordingScene>? OnUpdate { get; set; }

    /// <summary>Runs inside Enter, after logging.</summary>
    public Action<RecordingScene>? OnEnter { get; set; }

    protected override void Load() => _log.Add($"{Name}.Load");

    protected override void Enter()
    {
        _log.Add($"{Name}.Enter");
        OnEnter?.Invoke(this);
    }

    protected override void Revealed() => _log.Add($"{Name}.Revealed");

    protected override void Update(float dt, InputFrame input)
    {
        _log.Add($"{Name}.Update");
        Inputs.Add(input);
        OnUpdate?.Invoke(this);
    }

    protected override void Exit() => _log.Add($"{Name}.Exit");

    protected override void Unload() => _log.Add($"{Name}.Unload");

    protected override void Draw(ICanvas canvas) => canvas.FillRect(0, 0, 1, 1, Rgba.White);
}

/// <summary>A transition with explicit durations that records each draw instead of painting.</summary>
public sealed class StubTransition : ITransition
{
    public StubTransition(float outDuration, float inDuration)
    {
        OutDuration = outDuration;
        InDuration = inDuration;
    }

    public float OutDuration { get; }
    public float InDuration { get; }

    public List<(float Coverage, TransitionPhase Phase)> Draws { get; } = new();

    public void Draw(ICanvas canvas, float coverage, TransitionPhase phase) => Draws.Add((coverage, phase));
}

internal static class Frames
{
    public const float Dt = 1f / 60f;

    /// <summary>Call <paramref name="update"/> with a 60 Hz dt until <paramref name="seconds"/> have passed.</summary>
    public static void Run(Action<float> update, float seconds)
    {
        var frames = (int)MathF.Ceiling(seconds / Dt);
        for (var i = 0; i < frames; i++)
        {
            update(Dt);
        }
    }
}
