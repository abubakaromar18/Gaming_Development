package tiled

import (
	"fmt"
	"strconv"
	"strings"
)

// Color is a non-premultiplied 8-bit RGBA color as Tiled stores it.
type Color struct {
	R, G, B, A uint8
}

// ParseColor parses Tiled's "#RRGGBB" and "#AARRGGBB" color strings.
// Note that Tiled puts alpha first, unlike CSS.
func ParseColor(s string) (Color, error) {
	hex := strings.TrimPrefix(s, "#")
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil || (len(hex) != 6 && len(hex) != 8) {
		return Color{}, fmt.Errorf("%w: color %q is not #RRGGBB or #AARRGGBB", ErrInvalid, s)
	}
	c := Color{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
	if len(hex) == 8 {
		c.A = uint8(v >> 24)
	}
	return c, nil
}

// RGBA implements image/color.Color so a Color can be handed straight to any
// Go drawing API. The returned values are alpha-premultiplied, as required.
func (c Color) RGBA() (r, g, b, a uint32) {
	a = uint32(c.A) * 0x101
	r = uint32(c.R) * 0x101 * a / 0xffff
	g = uint32(c.G) * 0x101 * a / 0xffff
	b = uint32(c.B) * 0x101 * a / 0xffff
	return r, g, b, a
}

func parseOptionalColor(s string) (Color, bool, error) {
	if s == "" {
		return Color{}, false, nil
	}
	c, err := ParseColor(s)
	return c, err == nil, err
}
