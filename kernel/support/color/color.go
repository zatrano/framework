package color

import (
	"fmt"
	"strconv"
	"strings"
)

// RGBA is an 8-bit color.
type RGBA struct {
	R, G, B, A uint8
}

// Hex returns #RRGGBB or #RRGGBBAA.
func (c RGBA) Hex() string {
	if c.A == 255 {
		return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02X%02X%02X%02X", c.R, c.G, c.B, c.A)
}

// ParseHex parses #RGB, #RRGGBB, or #RRGGBBAA.
func ParseHex(s string) (RGBA, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	var c RGBA
	c.A = 255
	switch len(s) {
	case 3:
		r, _ := strconv.ParseUint(strings.Repeat(string(s[0]), 2), 16, 8)
		g, _ := strconv.ParseUint(strings.Repeat(string(s[1]), 2), 16, 8)
		b, _ := strconv.ParseUint(strings.Repeat(string(s[2]), 2), 16, 8)
		c.R, c.G, c.B = uint8(r), uint8(g), uint8(b)
	case 6:
		r, _ := strconv.ParseUint(s[0:2], 16, 8)
		g, _ := strconv.ParseUint(s[2:4], 16, 8)
		b, _ := strconv.ParseUint(s[4:6], 16, 8)
		c.R, c.G, c.B = uint8(r), uint8(g), uint8(b)
	case 8:
		r, _ := strconv.ParseUint(s[0:2], 16, 8)
		g, _ := strconv.ParseUint(s[2:4], 16, 8)
		b, _ := strconv.ParseUint(s[4:6], 16, 8)
		a, _ := strconv.ParseUint(s[6:8], 16, 8)
		c.R, c.G, c.B, c.A = uint8(r), uint8(g), uint8(b), uint8(a)
	default:
		return c, fmt.Errorf("color: invalid hex %q", s)
	}
	return c, nil
}

// Luminance returns relative luminance 0..1.
func Luminance(c RGBA) float64 {
	return (0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)) / 255
}

// IsDark reports whether the color is visually dark.
func IsDark(c RGBA) bool {
	return Luminance(c) < 0.5
}

// ContrastRatio between two colors (WCAG-ish).
func ContrastRatio(a, b RGBA) float64 {
	l1 := Luminance(a) + 0.05
	l2 := Luminance(b) + 0.05
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return l1 / l2
}

// Mix blends two colors by t in [0,1].
func Mix(a, b RGBA, t float64) RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
		A: uint8(float64(a.A)*(1-t) + float64(b.A)*t),
	}
}
