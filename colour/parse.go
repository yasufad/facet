package colour

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseHex parses a CSS-style hex colour string into an Rgba.
//
// It accepts the forms #rgb, #rrggbb and #rrggbbaa, case-insensitively, with
// optional surrounding whitespace. The short form duplicates each digit, so
// #f09 is equivalent to #ff0099. Forms without an alpha channel are opaque.
func ParseHex(s string) (Rgba, error) {
	const expected = "expected #rgb, #rrggbb or #rrggbbaa"

	hex := strings.TrimSpace(s)
	if !strings.HasPrefix(hex, "#") {
		return Rgba{}, fmt.Errorf("colour: parse %q: missing leading '#', %s", s, expected)
	}
	hex = hex[1:]

	var r, g, b, a uint8 = 0, 0, 0, 0xff
	switch len(hex) {
	case 3: // #rgb
		rv, err := nibble(hex[0])
		if err != nil {
			return Rgba{}, fmt.Errorf("colour: parse %q: red component: %w", s, err)
		}
		gv, err := nibble(hex[1])
		if err != nil {
			return Rgba{}, fmt.Errorf("colour: parse %q: green component: %w", s, err)
		}
		bv, err := nibble(hex[2])
		if err != nil {
			return Rgba{}, fmt.Errorf("colour: parse %q: blue component: %w", s, err)
		}
		r, g, b = dup(rv), dup(gv), dup(bv)
	case 6, 8: // #rrggbb or #rrggbbaa
		rv, err := bytePair(hex[0:2])
		if err != nil {
			return Rgba{}, fmt.Errorf("colour: parse %q: red component: %w", s, err)
		}
		gv, err := bytePair(hex[2:4])
		if err != nil {
			return Rgba{}, fmt.Errorf("colour: parse %q: green component: %w", s, err)
		}
		bv, err := bytePair(hex[4:6])
		if err != nil {
			return Rgba{}, fmt.Errorf("colour: parse %q: blue component: %w", s, err)
		}
		r, g, b = rv, gv, bv
		if len(hex) == 8 {
			av, err := bytePair(hex[6:8])
			if err != nil {
				return Rgba{}, fmt.Errorf("colour: parse %q: alpha component: %w", s, err)
			}
			a = av
		}
	default:
		return Rgba{}, fmt.Errorf("colour: parse %q: %d digits, %s", s, len(hex), expected)
	}

	return Rgba{
		R: float32(r) / 255,
		G: float32(g) / 255,
		B: float32(b) / 255,
		A: float32(a) / 255,
	}, nil
}

// nibble parses a single hexadecimal digit into its 0-15 value.
func nibble(c byte) (uint8, error) {
	v, err := strconv.ParseUint(string(c), 16, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid hex digit %q", c)
	}
	return uint8(v), nil
}

// bytePair parses a two-digit hexadecimal byte.
func bytePair(s string) (uint8, error) {
	v, err := strconv.ParseUint(s, 16, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid hex byte %q", s)
	}
	return uint8(v), nil
}

// dup duplicates a single hex digit, so 0xf becomes 0xff.
func dup(v uint8) uint8 { return v<<4 | v }
