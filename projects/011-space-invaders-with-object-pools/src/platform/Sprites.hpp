#pragma once

// Procedurally defined 1-bit sprites and a 3x5 pixel font. Every asset in
// the game is one of these string bitmaps, so there are no image files.

#include <array>
#include <string_view>

namespace invaders::sprites {

// A bitmap is a list of rows; 'X' is a lit pixel.
template <std::size_t Rows>
using Bitmap = std::array<std::string_view, Rows>;

inline constexpr Bitmap<8> kCannon = {
    "......X......",
    ".....XXX.....",
    ".....XXX.....",
    ".XXXXXXXXXXX.",
    "XXXXXXXXXXXXX",
    "XXXXXXXXXXXXX",
    "XXXXXXXXXXXXX",
    "XXXXXXXXXXXXX",
};

// Three original invader designs, two animation frames each, 12x8.
inline constexpr std::array<Bitmap<8>, 2> kSquid = {{
    {"....XXXX....", "..XXXXXXXX..", ".XX.XXXX.XX.", ".XXXXXXXXXX.",
     "...XX..XX...", "..XX.XX.XX..", ".XX......XX.", "..X......X.."},
    {"....XXXX....", "..XXXXXXXX..", ".XX.XXXX.XX.", ".XXXXXXXXXX.",
     "...XX..XX...", "..XX.XX.XX..", "...XX..XX...", "..XX....XX.."},
}};

inline constexpr std::array<Bitmap<8>, 2> kCrab = {{
    {"..X......X..", "...X....X...", "..XXXXXXXX..", ".XX.XXXX.XX.",
     "XXXXXXXXXXXX", "X.XXXXXXXX.X", "X.X......X.X", "...XX..XX..."},
    {"..X......X..", "X..X....X..X", "X.XXXXXXXX.X", "XXX.XXXX.XXX",
     "XXXXXXXXXXXX", ".XXXXXXXXXX.", "..X......X..", ".X........X."},
}};

inline constexpr std::array<Bitmap<8>, 2> kBlob = {{
    {"...XXXXXX...", ".XXXXXXXXXX.", "XXXXXXXXXXXX", "XX..XXXX..XX",
     "XXXXXXXXXXXX", "..XXX..XXX..", ".XX..XX..XX.", "XX........XX"},
    {"...XXXXXX...", ".XXXXXXXXXX.", "XXXXXXXXXXXX", "XX..XXXX..XX",
     "XXXXXXXXXXXX", "...XX..XX...", "..XX.XX.XX..", "...X....X..."},
}};

// 3x5 glyphs for the characters the HUD uses; unknown characters are blank.
inline Bitmap<5> glyph(char c) {
    switch (c) {
        case '0': return {"XXX", "X.X", "X.X", "X.X", "XXX"};
        case '1': return {".X.", "XX.", ".X.", ".X.", "XXX"};
        case '2': return {"XXX", "..X", "XXX", "X..", "XXX"};
        case '3': return {"XXX", "..X", ".XX", "..X", "XXX"};
        case '4': return {"X.X", "X.X", "XXX", "..X", "..X"};
        case '5': return {"XXX", "X..", "XXX", "..X", "XXX"};
        case '6': return {"XXX", "X..", "XXX", "X.X", "XXX"};
        case '7': return {"XXX", "..X", ".X.", ".X.", ".X."};
        case '8': return {"XXX", "X.X", "XXX", "X.X", "XXX"};
        case '9': return {"XXX", "X.X", "XXX", "..X", "XXX"};
        case 'A': return {".X.", "X.X", "XXX", "X.X", "X.X"};
        case 'B': return {"XX.", "X.X", "XX.", "X.X", "XX."};
        case 'C': return {".XX", "X..", "X..", "X..", ".XX"};
        case 'D': return {"XX.", "X.X", "X.X", "X.X", "XX."};
        case 'E': return {"XXX", "X..", "XX.", "X..", "XXX"};
        case 'F': return {"XXX", "X..", "XX.", "X..", "X.."};
        case 'G': return {".XX", "X..", "X.X", "X.X", ".XX"};
        case 'H': return {"X.X", "X.X", "XXX", "X.X", "X.X"};
        case 'I': return {"XXX", ".X.", ".X.", ".X.", "XXX"};
        case 'L': return {"X..", "X..", "X..", "X..", "XXX"};
        case 'M': return {"X.X", "XXX", "XXX", "X.X", "X.X"};
        case 'N': return {"XX.", "X.X", "X.X", "X.X", "X.X"};
        case 'O': return {".X.", "X.X", "X.X", "X.X", ".X."};
        case 'P': return {"XX.", "X.X", "XX.", "X..", "X.."};
        case 'R': return {"XX.", "X.X", "XX.", "X.X", "X.X"};
        case 'S': return {".XX", "X..", ".X.", "..X", "XX."};
        case 'T': return {"XXX", ".X.", ".X.", ".X.", ".X."};
        case 'U': return {"X.X", "X.X", "X.X", "X.X", "XXX"};
        case 'V': return {"X.X", "X.X", "X.X", "X.X", ".X."};
        case 'W': return {"X.X", "X.X", "XXX", "XXX", "X.X"};
        case 'X': return {"X.X", "X.X", ".X.", "X.X", "X.X"};
        case 'Y': return {"X.X", "X.X", ".X.", ".X.", ".X."};
        case '/': return {"..X", "..X", ".X.", "X..", "X.."};
        default: return {"...", "...", "...", "...", "..."};
    }
}

}  // namespace invaders::sprites
