using SceneShell.Core.Arcade;
using SceneShell.Core.Arcade.Scenes;
using SceneShell.Core.Input;
using SceneShell.Core.Randomness;
using SceneShell.Core.Transitions;

namespace SceneShell.Core.Tests;

/// <summary>End-to-end screen flow through the real arcade, driven by scripted input.</summary>
public class ArcadeFlowTests
{
    private static ArcadeShell NewShell(ulong seed = 1234, ArcadeConfig? config = null) =>
        new(config ?? ArcadeConfig.Default, new Pcg32(seed));

    /// <summary>Run idle frames until the current transition has finished.</summary>
    private static void Settle(ArcadeShell shell)
    {
        for (var i = 0; i < 600 && shell.Scenes.IsTransitioning; i++)
        {
            shell.Update(Frames.Dt, InputFrame.None);
        }
        Assert.False(shell.Scenes.IsTransitioning, "transition did not finish within 10 s");
    }

    private static void Press(ArcadeShell shell, GameAction action)
    {
        shell.Update(Frames.Dt, InputFrame.Press(action));
        Settle(shell);
    }

    private static ArcadeShell ShellAtMenu(ulong seed = 1234, ArcadeConfig? config = null)
    {
        var shell = NewShell(seed, config);
        Settle(shell);
        Press(shell, GameAction.Confirm);
        Assert.Equal(SceneId.Menu, shell.Scenes.CurrentId);
        return shell;
    }

    [Fact]
    public void Boots_into_the_title_with_a_fade_in()
    {
        var shell = NewShell();

        Assert.Equal(SceneId.Title, shell.Scenes.CurrentId);
        Assert.Equal(TransitionPhase.In, shell.Scenes.Phase);
        Assert.Equal(1.0, shell.Scenes.Coverage, 3);
    }

    [Fact]
    public void Title_prompt_appears_only_once_input_is_live()
    {
        var shell = NewShell();
        var title = Assert.IsType<TitleScene>(shell.Scenes.Current);

        Assert.False(title.IsPromptVisible);
        Settle(shell);
        Assert.True(title.IsInteractive);
        Assert.True(title.IsPromptVisible);
    }

    [Fact]
    public void Title_starfield_keeps_moving_while_covered()
    {
        var shell = NewShell();
        var title = Assert.IsType<TitleScene>(shell.Scenes.Current);
        var before = title.StarPositions;

        shell.Update(0.1f, InputFrame.None);   // still fading in

        Assert.True(shell.Scenes.IsTransitioning);
        Assert.NotEqual(before[0].Y, title.StarPositions[0].Y);
        Assert.True(title.AnimationTime > 0f);
    }

    [Fact]
    public void Pressing_enter_during_the_fade_in_is_ignored()
    {
        var shell = NewShell();

        shell.Update(Frames.Dt, InputFrame.Press(GameAction.Confirm));
        Settle(shell);

        Assert.Equal(SceneId.Title, shell.Scenes.CurrentId);
    }

    [Fact]
    public void Menu_cursor_wraps_in_both_directions()
    {
        var shell = ShellAtMenu();
        var menu = Assert.IsType<MenuScene>(shell.Scenes.Current);

        Press(shell, GameAction.Up);
        Assert.Equal(MenuScene.Item.Quit, menu.Selected);
        Press(shell, GameAction.Down);
        Assert.Equal(MenuScene.Item.Dodge, menu.Selected);
    }

    [Fact]
    public void Menu_is_persistent_and_remembers_the_cursor_across_rounds()
    {
        var shell = ShellAtMenu();
        var menu = Assert.IsType<MenuScene>(shell.Scenes.Current);
        Press(shell, GameAction.Down);                 // COLLECT
        Press(shell, GameAction.Confirm);
        Assert.Equal(SceneId.Collect, shell.Scenes.CurrentId);

        Press(shell, GameAction.Back);                 // abandon the round

        Assert.Equal(SceneId.Menu, shell.Scenes.CurrentId);
        Assert.Same(menu, shell.Scenes.Current);
        Assert.Equal(MenuScene.Item.Collect, menu.Selected);
        Assert.Equal(2, menu.Visits);
    }

    [Fact]
    public void Abandoning_a_round_records_no_score()
    {
        var shell = ShellAtMenu();
        Press(shell, GameAction.Confirm);              // DODGE
        Frames.Run(dt => shell.Update(dt, InputFrame.None), 0.5f);

        Press(shell, GameAction.Back);

        Assert.Equal(SceneId.Menu, shell.Scenes.CurrentId);
        Assert.Null(shell.Session.LastResult);
    }

    [Fact]
    public void Standing_still_in_dodge_ends_on_the_results_card_with_the_score()
    {
        var shell = ShellAtMenu();
        Press(shell, GameAction.Confirm);
        var dodge = Assert.IsType<DodgeScene>(shell.Scenes.Current);

        PlayUntilOver(shell, () => dodge.IsOver);
        var finalScore = dodge.Score;
        Settle(shell);

        var results = Assert.IsType<ResultsScene>(shell.Scenes.Current);
        Assert.True(finalScore > 0);
        Assert.Equal(new GameResult(ArcadeGame.Dodge, finalScore, IsNewBest: true), results.Result);
        Assert.Equal(finalScore, shell.Session.BestFor(ArcadeGame.Dodge));
    }

    [Fact]
    public void Play_again_starts_a_fresh_round_of_the_same_game()
    {
        var shell = ShellAtMenu();
        Press(shell, GameAction.Confirm);
        var first = Assert.IsType<DodgeScene>(shell.Scenes.Current);
        PlayUntilOver(shell, () => first.IsOver);
        Settle(shell);

        Press(shell, GameAction.Confirm);

        var second = Assert.IsType<DodgeScene>(shell.Scenes.Current);
        Assert.NotSame(first, second);
        Assert.False(second.IsOver);
        Assert.Equal(0, second.Score);
    }

    [Fact]
    public void Dodge_difficulty_ramps_up_and_respects_its_limits()
    {
        var rules = new DodgeConfig();
        var shell = ShellAtMenu();
        Press(shell, GameAction.Confirm);
        var dodge = Assert.IsType<DodgeScene>(shell.Scenes.Current);
        var startInterval = dodge.SpawnInterval;
        var startSpeed = dodge.FallSpeed;

        // Weave left and right to survive a while; the ramp depends only on time.
        for (var frame = 0; frame < 120 && !dodge.IsOver; frame++)
        {
            var steer = frame / 30 % 2 == 0 ? GameAction.Left : GameAction.Right;
            shell.Update(Frames.Dt, InputFrame.Hold(steer));
        }

        Assert.True(dodge.SpawnInterval < startInterval);
        Assert.True(dodge.FallSpeed > startSpeed);
        Assert.InRange(dodge.SpawnInterval, rules.MinSpawnInterval, rules.StartSpawnInterval);
        Assert.InRange(dodge.FallSpeed, rules.StartFallSpeed, rules.MaxFallSpeed);
    }

    [Fact]
    public void Player_stays_inside_the_screen()
    {
        var shell = ShellAtMenu();
        Press(shell, GameAction.Confirm);
        var dodge = Assert.IsType<DodgeScene>(shell.Scenes.Current);

        for (var i = 0; i < 90 && !dodge.IsOver; i++)
        {
            shell.Update(Frames.Dt, InputFrame.Hold(GameAction.Left));
        }

        Assert.Equal(0f, dodge.Player.X);
    }

    [Fact]
    public void Same_seed_and_same_input_give_the_same_dodge_round()
    {
        int Play(ulong seed)
        {
            var shell = ShellAtMenu(seed);
            Press(shell, GameAction.Confirm);
            var dodge = Assert.IsType<DodgeScene>(shell.Scenes.Current);
            for (var frame = 0; !dodge.IsOver; frame++)
            {
                Assert.True(frame < 60 * 600, "round never ended");
                var steer = frame / 45 % 2 == 0 ? GameAction.Right : GameAction.Left;
                shell.Update(Frames.Dt, InputFrame.Hold(steer));
            }
            return dodge.Score;
        }

        Assert.Equal(Play(99), Play(99));
    }

    [Fact]
    public void Collect_ends_after_too_many_misses()
    {
        // A tiny basket parked in the middle lets almost every gem fall past.
        var config = ArcadeConfig.Default with { Collect = new CollectConfig { BasketWidth = 1f } };
        var shell = ShellAtMenu(config: config);
        Press(shell, GameAction.Down);
        Press(shell, GameAction.Confirm);
        var collect = Assert.IsType<CollectScene>(shell.Scenes.Current);

        PlayUntilOver(shell, () => collect.IsOver);
        Settle(shell);

        Assert.Equal(config.Collect.MaxMisses, collect.Missed);
        Assert.True(collect.TimeLeft > 0f);
        Assert.Equal(SceneId.Results, shell.Scenes.CurrentId);
        Assert.Equal(ArcadeGame.Collect, shell.Session.LastResult!.Value.Game);
    }

    [Fact]
    public void Collect_with_a_full_width_basket_catches_everything_until_time_runs_out()
    {
        var config = ArcadeConfig.Default with
        {
            Collect = new CollectConfig { BasketWidth = ArcadeConfig.Default.Width, RoundSeconds = 5f },
        };
        var shell = ShellAtMenu(config: config);
        Press(shell, GameAction.Down);
        Press(shell, GameAction.Confirm);
        var collect = Assert.IsType<CollectScene>(shell.Scenes.Current);

        PlayUntilOver(shell, () => collect.IsOver);

        Assert.Equal(0, collect.Missed);
        Assert.Equal(0f, collect.TimeLeft);
        Assert.True(collect.Caught > 0);
        Assert.Equal(collect.Caught, shell.Session.LastResult!.Value.Score);
    }

    [Fact]
    public void Results_back_returns_to_the_menu()
    {
        var shell = ShellAtMenu();
        Press(shell, GameAction.Confirm);
        var dodge = Assert.IsType<DodgeScene>(shell.Scenes.Current);
        PlayUntilOver(shell, () => dodge.IsOver);
        Settle(shell);

        Press(shell, GameAction.Back);

        Assert.Equal(SceneId.Menu, shell.Scenes.CurrentId);
    }

    [Fact]
    public void Quit_from_the_menu_fades_out_and_stops_the_shell()
    {
        var shell = ShellAtMenu();
        Press(shell, GameAction.Up);                   // QUIT

        shell.Update(Frames.Dt, InputFrame.Press(GameAction.Confirm));
        Assert.False(shell.HasQuit);
        Frames.Run(dt => shell.Update(dt, InputFrame.None), ArcadeConfig.Default.FadeSeconds + Frames.Dt);

        Assert.True(shell.HasQuit);
        var canvas = new RecordingCanvas();
        shell.Draw(canvas);
        Assert.Single(canvas.Rects);                   // just the black clear
    }

    [Fact]
    public void Escape_on_the_title_quits()
    {
        var shell = NewShell();
        Settle(shell);

        shell.Update(Frames.Dt, InputFrame.Press(GameAction.Back));
        Settle(shell);

        Assert.True(shell.HasQuit);
    }

    [Fact]
    public void Long_frames_are_clamped_before_reaching_the_simulation()
    {
        var shell = ShellAtMenu();
        Press(shell, GameAction.Confirm);
        var dodge = Assert.IsType<DodgeScene>(shell.Scenes.Current);

        shell.Update(5f, InputFrame.None);

        Assert.Equal((double)ArcadeConfig.Default.MaxFrameSeconds, dodge.Elapsed, 5);
    }

    [Fact]
    public void Debug_overlay_toggles_and_draws_the_manager_state()
    {
        var shell = NewShell();
        var plain = new RecordingCanvas();
        shell.Draw(plain);

        shell.Update(Frames.Dt, InputFrame.Press(GameAction.ToggleDebug));
        var withDebug = new RecordingCanvas();
        shell.Draw(withDebug);

        Assert.True(shell.ShowDebug);
        Assert.True(withDebug.Rects.Count > plain.Rects.Count);
    }

    [Fact]
    public void Every_scene_draws_without_errors()
    {
        var shell = ShellAtMenu();
        var canvas = new RecordingCanvas();
        shell.Draw(canvas);
        Press(shell, GameAction.Confirm);
        var dodge = Assert.IsType<DodgeScene>(shell.Scenes.Current);
        Frames.Run(dt => shell.Update(dt, InputFrame.None), 1f);
        shell.Draw(canvas);
        PlayUntilOver(shell, () => dodge.IsOver);
        shell.Draw(canvas);                            // results card under the flash
        Settle(shell);
        shell.Draw(canvas);

        Assert.NotEmpty(canvas.Rects);
    }

    private static void PlayUntilOver(ArcadeShell shell, Func<bool> isOver)
    {
        for (var frame = 0; frame < 60 * 600 && !isOver(); frame++)
        {
            shell.Update(Frames.Dt, InputFrame.None);
        }
        Assert.True(isOver(), "round never ended");
    }
}
