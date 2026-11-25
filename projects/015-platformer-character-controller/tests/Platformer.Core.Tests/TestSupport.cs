using Platformer.Core.Controller;
using Platformer.Core.Input;
using Platformer.Core.Physics;
using Platformer.Core.Rendering;
using Platformer.Core.World;

namespace Platformer.Core.Tests;

internal static class Maps
{
    public const int Tile = 16;
    public const float Dt = 1f / 120f;

    public static Level Parse(params string[] rows) => Level.Parse(string.Join("\n", rows), Tile);

    /// <summary>A wide open room with a floor; spawn on the floor near the left wall.</summary>
    public static Level Room() => Parse(
        "#..............................#",
        "#..............................#",
        "#..............................#",
        "#..............................#",
        "#..............................#",
        "#..............................#",
        "#..............................#",
        "#..P...........................#",
        "################################");

    /// <summary>A ledge that ends at x = 80 px, with a deep drop after it.</summary>
    public static Level Ledge() => Parse(
        "#..............................#",
        "#..............................#",
        "#.P............................#",
        "#####..........................#",
        "#..............................#",
        "#..............................#",
        "#..............................#",
        "#..............................#",
        "#..............................#",
        "################################");

    /// <summary>Spawn in mid-air, five tiles above the floor.</summary>
    public static Level MidAir() => Parse(
        "#..........#",
        "#....P.....#",
        "#..........#",
        "#..........#",
        "#..........#",
        "#..........#",
        "#..........#",
        "############");

    /// <summary>A ceiling at y 32..48 with a two-tile gap over x 64..96.</summary>
    public static Level Notch() => Parse(
        "#..........#",
        "#..........#",
        "####..######",
        "#..........#",
        "#..........#",
        "#.P........#",
        "############");
}

internal static class Sim
{
    public static CharacterController Controller(Level level, ControllerTuning? tuning = null) =>
        new(tuning ?? ControllerTuning.Default, level.Map, level.Spawn);

    public static ControllerInput Hold(float moveX = 0f, bool jump = false) => new(moveX, jump, false);

    public static ControllerInput Press(float moveX = 0f) => new(moveX, true, true);

    /// <summary>Step <paramref name="steps"/> times with the same input; returns every event seen.</summary>
    public static List<ControllerEvent> Run(CharacterController c, ControllerInput input, int steps)
    {
        var events = new List<ControllerEvent>();
        for (var i = 0; i < steps; i++)
        {
            c.Step(input, Maps.Dt);
            events.AddRange(c.Events);
        }
        return events;
    }

    /// <summary>
    /// Press jump once, keep it held for <paramref name="holdSteps"/> steps, then release, and
    /// run until landing. Returns the peak height above the take-off point and the airborne step count.
    /// </summary>
    public static (float Peak, int AirSteps) Jump(CharacterController c, int holdSteps, int maxSteps = 600)
    {
        var startY = c.Feet.Y;
        var minY = startY;
        c.Step(Press(), Maps.Dt);
        var air = 1;
        for (var i = 1; i < maxSteps && !c.Grounded; i++)
        {
            c.Step(Hold(jump: i < holdSteps), Maps.Dt);
            minY = MathF.Min(minY, c.Feet.Y);
            air++;
        }
        return (startY - minY, air);
    }
}

internal sealed class RecordingCanvas : ICanvas
{
    public RecordingCanvas(int width, int height)
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

internal static class Boxes
{
    public static Aabb At(float x, float y, float w = 12f, float h = 14f) => new(x, y, w, h);
}
