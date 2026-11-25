namespace Platformer.Core.Rendering;

/// <summary>
/// A 3×5 bitmap font defined in code and drawn as rectangles. It keeps the project free of
/// font files and content-pipeline steps; every glyph is five rows of three cells.
/// Lowercase letters render as uppercase, and unknown characters render as '?'.
/// </summary>
public static class PixelFont
{
    public const int GlyphWidth = 3;
    public const int GlyphHeight = 5;

    /// <summary>Horizontal advance per character in font cells (glyph plus one cell of spacing).</summary>
    public const int Advance = GlyphWidth + 1;

    private static readonly Dictionary<char, bool[]> Glyphs = BuildGlyphs();

    /// <summary>Width in pixels of <paramref name="text"/> at <paramref name="scale"/>, without trailing spacing.</summary>
    public static float Measure(string text, float scale)
    {
        ArgumentNullException.ThrowIfNull(text);
        return text.Length == 0 ? 0f : (text.Length * Advance - 1) * scale;
    }

    /// <summary>True if the character has its own glyph (rather than falling back to '?').</summary>
    public static bool Supports(char c) => Glyphs.ContainsKey(char.ToUpperInvariant(c));

    public static void Draw(ICanvas canvas, string text, float x, float y, float scale, Rgba color)
    {
        ArgumentNullException.ThrowIfNull(canvas);
        ArgumentNullException.ThrowIfNull(text);
        if (scale <= 0f)
        {
            throw new ArgumentOutOfRangeException(nameof(scale), scale, "Scale must be positive.");
        }

        var penX = x;
        foreach (var c in text)
        {
            var cells = Glyphs.TryGetValue(char.ToUpperInvariant(c), out var glyph) ? glyph : Glyphs['?'];
            for (var row = 0; row < GlyphHeight; row++)
            {
                for (var col = 0; col < GlyphWidth; col++)
                {
                    if (cells[row * GlyphWidth + col])
                    {
                        canvas.FillRect(penX + col * scale, y + row * scale, scale, scale, color);
                    }
                }
            }
            penX += Advance * scale;
        }
    }

    /// <summary>Draw <paramref name="text"/> horizontally centred on <paramref name="centerX"/>.</summary>
    public static void DrawCentered(ICanvas canvas, string text, float centerX, float y, float scale, Rgba color) =>
        Draw(canvas, text, centerX - Measure(text, scale) / 2f, y, scale, color);

    private static Dictionary<char, bool[]> BuildGlyphs()
    {
        // Each glyph is five space-separated rows of three cells: '#' is ink, '.' is empty.
        var source = new Dictionary<char, string>
        {
            ['A'] = ".#. #.# ### #.# #.#",
            ['B'] = "##. #.# ##. #.# ##.",
            ['C'] = ".## #.. #.. #.. .##",
            ['D'] = "##. #.# #.# #.# ##.",
            ['E'] = "### #.. ##. #.. ###",
            ['F'] = "### #.. ##. #.. #..",
            ['G'] = ".## #.. #.# #.# .##",
            ['H'] = "#.# #.# ### #.# #.#",
            ['I'] = "### .#. .#. .#. ###",
            ['J'] = "..# ..# ..# #.# .#.",
            ['K'] = "#.# #.# ##. #.# #.#",
            ['L'] = "#.. #.. #.. #.. ###",
            ['M'] = "#.# ### #.# #.# #.#",
            ['N'] = "##. #.# #.# #.# #.#",
            ['O'] = ".#. #.# #.# #.# .#.",
            ['P'] = "##. #.# ##. #.. #..",
            ['Q'] = ".#. #.# #.# ##. .##",
            ['R'] = "##. #.# ##. #.# #.#",
            ['S'] = ".## #.. .#. ..# ##.",
            ['T'] = "### .#. .#. .#. .#.",
            ['U'] = "#.# #.# #.# #.# ###",
            ['V'] = "#.# #.# #.# #.# .#.",
            ['W'] = "#.# #.# #.# ### #.#",
            ['X'] = "#.# #.# .#. #.# #.#",
            ['Y'] = "#.# #.# .#. .#. .#.",
            ['Z'] = "### ..# .#. #.. ###",
            ['0'] = "### #.# #.# #.# ###",
            ['1'] = ".#. ##. .#. .#. ###",
            ['2'] = "##. ..# .#. #.. ###",
            ['3'] = "##. ..# .#. ..# ##.",
            ['4'] = "#.# #.# ### ..# ..#",
            ['5'] = "### #.. ##. ..# ##.",
            ['6'] = ".## #.. ### #.# ###",
            ['7'] = "### ..# .#. .#. .#.",
            ['8'] = "### #.# ### #.# ###",
            ['9'] = "### #.# ### ..# ##.",
            [' '] = "... ... ... ... ...",
            ['.'] = "... ... ... ... .#.",
            [':'] = "... .#. ... .#. ...",
            ['-'] = "... ... ### ... ...",
            ['!'] = ".#. .#. .#. ... .#.",
            ['?'] = "##. ..# .#. ... .#.",
            ['>'] = "#.. .#. ..# .#. #..",
            ['<'] = "..# .#. #.. .#. ..#",
            ['/'] = "..# ..# .#. #.. #..",
        };

        var glyphs = new Dictionary<char, bool[]>(source.Count);
        foreach (var (c, cells) in source)
        {
            var packed = cells.Replace(" ", string.Empty, StringComparison.Ordinal);
            if (packed.Length != GlyphWidth * GlyphHeight)
            {
                throw new InvalidOperationException($"Glyph '{c}' has {packed.Length} cells, expected {GlyphWidth * GlyphHeight}.");
            }
            glyphs[c] = packed.Select(cell => cell == '#').ToArray();
        }
        return glyphs;
    }
}
