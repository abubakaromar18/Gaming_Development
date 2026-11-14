using System.Globalization;
using SceneShell.Core.Arcade;
using SceneShell.Desktop;

// Usage: SceneShell [seed]
// A fixed seed replays the same block and gem patterns; otherwise one is picked at launch.
ulong seed;
if (args.Length > 0)
{
    if (!ulong.TryParse(args[0], NumberStyles.None, CultureInfo.InvariantCulture, out seed))
    {
        Console.Error.WriteLine($"Seed must be a non-negative integer, got '{args[0]}'.");
        return 1;
    }
}
else
{
    seed = (ulong)Environment.TickCount64;
}

Console.WriteLine($"Scene Shell, seed {seed}");
using var game = new ShellGame(ArcadeConfig.Default, seed);
game.Run();
return 0;
