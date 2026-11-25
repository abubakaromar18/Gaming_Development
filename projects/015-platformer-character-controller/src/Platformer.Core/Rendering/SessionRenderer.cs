using System.Globalization;
using Platformer.Core.Controller;
using Platformer.Core.Game;

namespace Platformer.Core.Rendering;

/// <summary>
/// Draws a <see cref="PlatformerSession"/> with filled rectangles only. It lives in the
/// core because it is plain logic over an <see cref="ICanvas"/>: the desktop host only
/// supplies the rectangle primitive, and tests can check what gets drawn.
/// The canvas is expected to be the map's pixel size; one world pixel is one canvas pixel.
/// </summary>
public static class SessionRenderer
{
    public static readonly Rgba Sky = new(24, 26, 38);
    public static readonly Rgba Rock = new(70, 78, 104);
    public static readonly Rgba RockEdge = new(132, 150, 186);
    public static readonly Rgba Text = new(214, 220, 236);
    public static readonly Rgba Dim = new(110, 116, 140);
    public static readonly Rgba On = new(120, 220, 140);
    public static readonly Rgba Off = new(230, 100, 100);
    public static readonly Rgba Dust = new(200, 196, 180);
    public static readonly Rgba TrailColor = new(250, 210, 90);

    public static Rgba BodyColor(MovementPhase phase) => phase switch
    {
        MovementPhase.Grounded => new Rgba(240, 150, 80),
        MovementPhase.Rising => new Rgba(250, 190, 90),
        MovementPhase.Apex => new Rgba(255, 240, 140),
        MovementPhase.Falling => new Rgba(230, 120, 90),
        _ => throw new ArgumentOutOfRangeException(nameof(phase), phase, null),
    };

    public static void Draw(ICanvas canvas, PlatformerSession session, bool debug)
    {
        ArgumentNullException.ThrowIfNull(canvas);
        ArgumentNullException.ThrowIfNull(session);

        canvas.FillRect(0, 0, canvas.Width, canvas.Height, Sky);
        DrawTiles(canvas, session);
        if (debug)
        {
            DrawTrail(canvas, session);
        }
        DrawDust(canvas, session);
        DrawCharacter(canvas, session);
        DrawHud(canvas, session);
        if (debug)
        {
            DrawDebug(canvas, session);
        }
    }

    private static void DrawTiles(ICanvas canvas, PlatformerSession session)
    {
        var map = session.Level.Map;
        var ts = map.TileSize;
        for (var y = 0; y < map.Height; y++)
        {
            for (var x = 0; x < map.Width; x++)
            {
                if (!map.IsSolid(x, y))
                {
                    continue;
                }
                canvas.FillRect(x * ts, y * ts, ts, ts, Rock);
                if (y > 0 && !map.IsSolid(x, y - 1))
                {
                    // A lighter lip on walkable surfaces makes ledges readable.
                    canvas.FillRect(x * ts, y * ts, ts, 2, RockEdge);
                }
            }
        }
    }

    private static void DrawTrail(ICanvas canvas, PlatformerSession session)
    {
        var trail = session.Trail;
        for (var i = 0; i < trail.Count; i += 2)
        {
            var p = trail[i];
            var alpha = 0.15f + 0.85f * i / trail.Count;
            canvas.FillRect(p.X - 0.5f, p.Y - 0.5f, 1, 1, TrailColor.WithAlpha(alpha));
        }
    }

    private static void DrawDust(ICanvas canvas, PlatformerSession session)
    {
        foreach (var p in session.Dust.Particles)
        {
            if (!p.Alive)
            {
                continue;
            }
            var size = 3f * (1f - p.Progress) + 1f;
            canvas.FillRect(p.Position.X - size / 2f, p.Position.Y - size / 2f, size, size, Dust.WithAlpha(1f - p.Progress));
        }
    }

    private static void DrawCharacter(ICanvas canvas, PlatformerSession session)
    {
        var controller = session.Controller;
        var feet = session.InterpolatedFeet;
        var scale = session.Squash.Scale;
        var width = controller.Body.Width * scale.X;
        var height = controller.Body.Height * scale.Y;

        // Anchor the scaled sprite at the feet so squash spreads along the ground.
        var left = feet.X - width / 2f;
        var top = feet.Y - height;
        canvas.FillRect(left, top, width, height, BodyColor(controller.Phase));

        // An eye on the facing side shows direction and makes turnarounds readable.
        var eyeX = controller.Facing > 0 ? left + width * 0.62f : left + width * 0.38f - 2f;
        canvas.FillRect(eyeX, top + height * 0.25f, 2, 3, Sky);
    }

    private static void DrawHud(ICanvas canvas, PlatformerSession session)
    {
        const float x = 22f;
        var y = 20f;
        PixelFont.Draw(canvas, "ARROWS/AD MOVE  SPACE JUMP  1-5 TOGGLE  F3 DEBUG  R RESET", x, y, 1f, Dim);
        y += 10f;
        for (var i = 0; i < Assists.Ordered.Count; i++)
        {
            var feature = Assists.Ordered[i];
            var enabled = session.EnabledAssists.HasFlag(feature);
            PixelFont.Draw(canvas, $"{i + 1} {Assists.Label(feature)}", x, y, 1f, Text);
            PixelFont.Draw(canvas, enabled ? "ON" : "OFF", x + 84f, y, 1f, enabled ? On : Off);
            y += 7f;
        }

        var s = session.Stats;
        var statsX = 140f;
        var statsY = 30f;
        foreach (var line in new[]
        {
            $"JUMPS {s.Jumps}",
            $"COYOTE {s.CoyoteJumps}",
            $"BUFFERED {s.BufferedJumps}",
            $"CUT {s.JumpCuts}",
            $"CORNER {s.CornerCorrections}",
            $"FALLS {s.Falls}",
        })
        {
            PixelFont.Draw(canvas, line, statsX, statsY, 1f, Dim);
            statsY += 7f;
        }
    }

    private static void DrawDebug(ICanvas canvas, PlatformerSession session)
    {
        var c = session.Controller;
        var t = c.Tuning;
        var inv = CultureInfo.InvariantCulture;
        const float x = 220f;
        var y = 30f;
        PixelFont.Draw(canvas, $"PHASE {c.Phase.ToString().ToUpperInvariant()}", x, y, 1f, Text);
        y += 7f;
        PixelFont.Draw(canvas, string.Format(inv, "VX {0:0} VY {1:0}", c.Velocity.X, c.Velocity.Y), x, y, 1f, Text);
        y += 7f;
        PixelFont.Draw(canvas, string.Format(inv, "G {0:0} JUMP V {1:0}", t.Gravity, t.JumpVelocity), x, y, 1f, Dim);
        y += 9f;
        DrawTimerBar(canvas, "COYOTE", c.CoyoteRemaining, t.CoyoteTime, x, y);
        y += 7f;
        DrawTimerBar(canvas, "BUFFER", c.BufferRemaining, t.JumpBufferTime, x, y);
    }

    private static void DrawTimerBar(ICanvas canvas, string label, float remaining, float total, float x, float y)
    {
        const float barWidth = 48f;
        PixelFont.Draw(canvas, label, x, y, 1f, Dim);
        canvas.FillRect(x + 30f, y + 1f, barWidth, 3f, Rock);
        if (total > 0f && remaining > 0f)
        {
            canvas.FillRect(x + 30f, y + 1f, barWidth * remaining / total, 3f, On);
        }
    }
}
