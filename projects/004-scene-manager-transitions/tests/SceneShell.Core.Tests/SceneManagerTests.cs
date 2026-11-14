using SceneShell.Core.Input;
using SceneShell.Core.Scenes;
using SceneShell.Core.Transitions;

namespace SceneShell.Core.Tests;

public class SceneManagerTests
{
    private readonly List<string> _log = new();
    private readonly Dictionary<string, List<RecordingScene>> _instances = new();
    private readonly SceneRegistry<string> _registry = new();

    private static readonly StubTransition Half = new(0.5f, 0.5f);

    private SceneManager<string> CreateManager(ITransition? defaultTransition = null) =>
        new(_registry, defaultTransition ?? Half);

    /// <summary>Register a scene whose instances are tracked in <see cref="_instances"/>.</summary>
    private void Register(
        string id,
        SceneLifetime lifetime = SceneLifetime.Transient,
        bool updatesDuringTransition = false,
        Action<RecordingScene>? configure = null)
    {
        _instances[id] = new List<RecordingScene>();
        _registry.Register(id, () =>
        {
            var scene = new RecordingScene(id, _log, updatesDuringTransition);
            configure?.Invoke(scene);
            _instances[id].Add(scene);
            return scene;
        }, lifetime);
    }

    private RecordingScene Latest(string id) => _instances[id][^1];

    [Fact]
    public void Start_with_no_intro_loads_enters_and_reveals_immediately()
    {
        Register("a");
        var manager = CreateManager();

        manager.Start("a");

        Assert.Equal(new[] { "a.Load", "a.Enter", "a.Revealed" }, _log);
        Assert.Equal("a", manager.CurrentId);
        Assert.False(manager.IsTransitioning);
    }

    [Fact]
    public void Start_with_intro_reveals_only_after_the_fade_in()
    {
        Register("a");
        var manager = CreateManager();

        manager.Start("a", new StubTransition(0f, 0.5f));
        Assert.Equal(TransitionPhase.In, manager.Phase);
        Assert.Equal(1.0, manager.Coverage, 3);

        manager.Update(0.25f, InputFrame.None);
        Assert.Equal(0.5, manager.Coverage, 3);
        Assert.DoesNotContain("a.Revealed", _log);

        manager.Update(0.25f, InputFrame.None);
        Assert.Contains("a.Revealed", _log);
        Assert.Equal(TransitionPhase.None, manager.Phase);
    }

    [Fact]
    public void Change_requested_inside_update_is_deferred_until_the_call_returns()
    {
        var manager = CreateManager();
        Register("b");
        Register("a", configure: scene => scene.OnUpdate = s =>
        {
            manager.ChangeTo("b");
            // Still current: the request only queued.
            Assert.Same(s, manager.Current);
        });
        manager.Start("a");

        manager.Update(0.1f, InputFrame.None);

        Assert.Equal(TransitionPhase.Out, manager.Phase);
        Assert.Same(Latest("a"), manager.Current);
        Assert.Equal(0.0, manager.Coverage, 3);
    }

    [Fact]
    public void Swap_happens_exactly_when_the_screen_is_fully_covered()
    {
        Register("a");
        Register("b");
        var manager = CreateManager();
        manager.Start("a");
        _log.Clear();

        manager.ChangeTo("b");
        manager.Update(0f, InputFrame.None);   // idle update starts the transition
        manager.Update(0.25f, InputFrame.None);
        Assert.Equal(0.5, manager.Coverage, 3);
        Assert.Equal("a", manager.CurrentId);

        manager.Update(0.25f, InputFrame.None);
        Assert.Equal("b", manager.CurrentId);
        Assert.Equal(TransitionPhase.In, manager.Phase);
        Assert.Equal(1.0, manager.Coverage, 3);

        manager.Update(0.5f, InputFrame.None);
        Assert.Equal(TransitionPhase.None, manager.Phase);
        Assert.Equal(
            new[] { "a.Update", "a.Exit", "a.Unload", "b.Load", "b.Enter", "b.Revealed" },
            _log);
    }

    [Fact]
    public void Coverage_rises_monotonically_during_out_and_falls_during_in()
    {
        Register("a");
        Register("b");
        var manager = CreateManager();
        manager.Start("a");
        manager.ChangeTo("b");
        manager.Update(0f, InputFrame.None);

        var samples = new List<(TransitionPhase Phase, float Coverage)>();
        while (manager.IsTransitioning)
        {
            samples.Add((manager.Phase, manager.Coverage));
            manager.Update(Frames.Dt, InputFrame.None);
        }

        var outs = samples.Where(s => s.Phase == TransitionPhase.Out).Select(s => s.Coverage).ToList();
        var ins = samples.Where(s => s.Phase == TransitionPhase.In).Select(s => s.Coverage).ToList();
        Assert.NotEmpty(outs);
        Assert.NotEmpty(ins);
        Assert.True(outs.Zip(outs.Skip(1)).All(p => p.Second >= p.First), "out coverage must not decrease");
        Assert.True(ins.Zip(ins.Skip(1)).All(p => p.Second <= p.First), "in coverage must not increase");
    }

    [Fact]
    public void Input_is_withheld_for_the_whole_transition()
    {
        Register("a");
        Register("b");
        var manager = CreateManager();
        manager.Start("a");
        manager.ChangeTo("b");
        manager.Update(0f, InputFrame.None);
        _log.Clear();

        Frames.Run(dt => manager.Update(dt, InputFrame.Press(GameAction.Confirm)), 0.99f);

        Assert.DoesNotContain("a.Update", _log);
        Assert.DoesNotContain("b.Update", _log);

        manager.Update(Frames.Dt, InputFrame.None);  // finishes the fade-in
        manager.Update(Frames.Dt, InputFrame.Press(GameAction.Confirm));
        Assert.True(Latest("b").Inputs[^1].WasPressed(GameAction.Confirm));
    }

    [Fact]
    public void Scenes_that_opt_in_keep_updating_during_transitions_but_without_input()
    {
        Register("a", updatesDuringTransition: true);
        Register("b");
        var manager = CreateManager();
        manager.Start("a");
        manager.ChangeTo("b");
        manager.Update(0f, InputFrame.None);
        var before = Latest("a").Inputs.Count;

        manager.Update(0.1f, InputFrame.Press(GameAction.Confirm));
        manager.Update(0.1f, InputFrame.Press(GameAction.Back));

        var during = Latest("a").Inputs.Skip(before).ToList();
        Assert.Equal(2, during.Count);
        Assert.True(during.All(frame => frame == InputFrame.None));
    }

    [Fact]
    public void Transient_scenes_are_rebuilt_on_every_visit()
    {
        Register("a");
        Register("b");
        var manager = CreateManager(CutTransition.Instance);
        manager.Start("a");

        manager.ChangeTo("b");
        manager.Update(0f, InputFrame.None);
        manager.ChangeTo("a");
        manager.Update(0f, InputFrame.None);

        Assert.Equal(2, _instances["a"].Count);
        Assert.NotSame(_instances["a"][0], _instances["a"][1]);
        Assert.Equal(2, _log.Count(e => e == "a.Unload") + _log.Count(e => e == "b.Unload"));
    }

    [Fact]
    public void Persistent_scenes_load_once_enter_every_visit_and_unload_on_quit()
    {
        Register("menu", SceneLifetime.Persistent);
        Register("game");
        var manager = CreateManager(CutTransition.Instance);
        manager.Start("menu");

        for (var round = 0; round < 3; round++)
        {
            manager.ChangeTo("game");
            manager.Update(0f, InputFrame.None);
            manager.ChangeTo("menu");
            manager.Update(0f, InputFrame.None);
        }

        Assert.Single(_instances["menu"]);
        Assert.Equal(1, _log.Count(e => e == "menu.Load"));
        Assert.Equal(4, _log.Count(e => e == "menu.Enter"));
        Assert.Equal(0, _log.Count(e => e == "menu.Unload"));

        manager.Quit();
        manager.Update(0f, InputFrame.None);
        Assert.Equal(1, _log.Count(e => e == "menu.Unload"));
    }

    [Fact]
    public void Persistent_scene_unloads_once_when_it_is_the_scene_being_quit()
    {
        Register("menu", SceneLifetime.Persistent);
        var manager = CreateManager(CutTransition.Instance);
        manager.Start("menu");

        manager.Quit();
        manager.Update(0f, InputFrame.None);

        Assert.Equal(new[] { "menu.Load", "menu.Enter", "menu.Revealed", "menu.Update", "menu.Exit", "menu.Unload" }, _log);
    }

    [Fact]
    public void Requests_made_during_a_transition_queue_and_run_in_order()
    {
        Register("a");
        Register("b");
        Register("c");
        var manager = CreateManager();
        manager.Start("a");

        manager.ChangeTo("b");
        manager.Update(0f, InputFrame.None);
        manager.ChangeTo("c");
        Assert.Equal(1, manager.PendingCount);

        manager.Update(1f, InputFrame.None);    // a → b completes; b → c begins at once
        Assert.Equal("b", manager.CurrentId);
        Assert.Equal(TransitionPhase.Out, manager.Phase);

        manager.Update(1f, InputFrame.None);
        Assert.Equal("c", manager.CurrentId);
        Assert.Equal(0, manager.PendingCount);
        Assert.False(manager.IsTransitioning);
    }

    [Fact]
    public void One_long_frame_can_finish_a_whole_transition()
    {
        Register("a");
        Register("b");
        var manager = CreateManager();
        manager.Start("a");
        manager.ChangeTo("b");
        manager.Update(0f, InputFrame.None);

        manager.Update(5f, InputFrame.None);

        Assert.Equal("b", manager.CurrentId);
        Assert.False(manager.IsTransitioning);
        Assert.Contains("b.Revealed", _log);
    }

    [Fact]
    public void Leftover_time_from_the_fade_out_carries_into_the_fade_in()
    {
        Register("a");
        Register("b");
        var manager = CreateManager();
        manager.Start("a");
        manager.ChangeTo("b");
        manager.Update(0f, InputFrame.None);

        manager.Update(0.6f, InputFrame.None);   // 0.5 s out, then 0.1 s into the 0.5 s fade-in

        Assert.Equal(TransitionPhase.In, manager.Phase);
        Assert.Equal(0.8, manager.Coverage, 3);
    }

    [Fact]
    public void A_change_requested_from_enter_runs_after_the_reveal()
    {
        var manager = CreateManager(CutTransition.Instance);
        Register("b");
        Register("splash", configure: scene => scene.OnEnter = _ => manager.ChangeTo("b"));

        manager.Start("splash");

        Assert.Equal("b", manager.CurrentId);
        Assert.Equal(
            new[] { "splash.Load", "splash.Enter", "splash.Revealed", "splash.Exit", "splash.Unload", "b.Load", "b.Enter", "b.Revealed" },
            _log);
    }

    [Fact]
    public void Two_scenes_bouncing_with_zero_length_transitions_fail_loudly()
    {
        var manager = CreateManager(CutTransition.Instance);
        Register("ping", configure: scene => scene.OnEnter = _ => manager.ChangeTo("pong"));
        Register("pong", configure: scene => scene.OnEnter = _ => manager.ChangeTo("ping"));

        var error = Assert.Throws<InvalidOperationException>(() => manager.Start("ping"));
        Assert.Contains("scene changes in one update", error.Message);
    }

    [Fact]
    public void Quit_covers_the_screen_then_unloads_everything()
    {
        Register("a");
        var manager = CreateManager();
        manager.Start("a");
        manager.Quit();
        manager.Update(0f, InputFrame.None);

        manager.Update(0.25f, InputFrame.None);
        Assert.False(manager.HasQuit);
        Assert.NotNull(manager.Current);

        manager.Update(0.25f, InputFrame.None);
        Assert.True(manager.HasQuit);
        Assert.Null(manager.Current);
        Assert.False(manager.IsTransitioning);
        Assert.Equal(new[] { "a.Exit", "a.Unload" }, _log.TakeLast(2));
    }

    [Fact]
    public void After_quit_updates_are_ignored_and_requests_are_rejected()
    {
        Register("a");
        var manager = CreateManager(CutTransition.Instance);
        manager.Start("a");
        manager.Quit();
        manager.Update(0f, InputFrame.None);
        var logged = _log.Count;

        manager.Update(1f, InputFrame.Press(GameAction.Confirm));

        Assert.Equal(logged, _log.Count);
        Assert.Throws<InvalidOperationException>(() => manager.ChangeTo("a"));
        Assert.Throws<InvalidOperationException>(() => manager.Quit());
        Assert.Throws<InvalidOperationException>(() => _ = manager.CurrentId);
    }

    [Fact]
    public void Pending_requests_are_dropped_when_quit_runs_first()
    {
        Register("a");
        Register("b");
        var manager = CreateManager(CutTransition.Instance);
        manager.Start("a");

        manager.Quit();
        manager.ChangeTo("b");
        manager.Update(0f, InputFrame.None);

        Assert.True(manager.HasQuit);
        Assert.Empty(_instances["b"]);
    }

    [Fact]
    public void Draw_paints_the_scene_then_the_cover_only_while_transitioning()
    {
        Register("a");
        Register("b");
        var transition = new StubTransition(0.5f, 0.5f);
        var manager = CreateManager(transition);
        manager.Start("a");
        var canvas = new RecordingCanvas();

        manager.Draw(canvas);
        Assert.Single(canvas.Rects);
        Assert.Empty(transition.Draws);

        manager.ChangeTo("b");
        manager.Update(0f, InputFrame.None);
        manager.Update(0.25f, InputFrame.None);
        manager.Draw(canvas);
        Assert.Equal(2, canvas.Rects.Count);
        Assert.Equal((0.5f, TransitionPhase.Out), transition.Draws.Single());
    }

    [Fact]
    public void Misuse_is_reported_with_clear_exceptions()
    {
        Register("a");
        _registry.Register("broken", () => null!);
        var manager = CreateManager();

        Assert.Throws<InvalidOperationException>(() => manager.Update(0.1f, InputFrame.None));
        Assert.Throws<InvalidOperationException>(() => manager.ChangeTo("a"));
        Assert.Throws<KeyNotFoundException>(() => manager.Start("missing"));

        manager.Start("a");
        Assert.Throws<InvalidOperationException>(() => manager.Start("a"));
        Assert.Throws<KeyNotFoundException>(() => manager.ChangeTo("missing"));
        Assert.Throws<ArgumentOutOfRangeException>(() => manager.Update(-0.1f, InputFrame.None));
        Assert.Throws<ArgumentOutOfRangeException>(() => manager.Update(float.NaN, InputFrame.None));

        manager.ChangeTo("broken", CutTransition.Instance);
        Assert.Throws<InvalidOperationException>(() => manager.Update(0f, InputFrame.None));
    }

    [Fact]
    public void Registry_rejects_duplicate_ids()
    {
        Register("a");
        Assert.Throws<ArgumentException>(() => _registry.Register("a", () => new RecordingScene("a", _log)));
    }
}
