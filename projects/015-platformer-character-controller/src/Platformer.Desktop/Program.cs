using System.Globalization;
using Platformer.Desktop;

// Usage: Platformer [seed]
// The seed only drives cosmetic dust; a fixed one makes a run's visuals reproducible.
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

Console.WriteLine($"Platformer Character Controller, seed {seed}");
using var game = new PlatformerGame(seed);
game.Run();
return 0;
