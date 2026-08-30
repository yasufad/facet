package colour

import "testing"

func TestParseHexValid(t *testing.T) {
	cases := []struct {
		in   string
		want Rgba
	}{
		// Short form: each digit is duplicated.
		{"#f00", Rgba{1, 0, 0, 1}},
		{"#0f0", Rgba{0, 1, 0, 1}},
		{"#00f", Rgba{0, 0, 1, 1}},
		{"#f09", Rgba{1, 0, 0x99 / 255.0, 1}},
		{"#000", Rgba{0, 0, 0, 1}},
		{"#fff", Rgba{1, 1, 1, 1}},
		// Six-digit form, opaque.
		{"#ff0000", Rgba{1, 0, 0, 1}},
		{"#00ff00", Rgba{0, 1, 0, 1}},
		{"#0000ff", Rgba{0, 0, 1, 1}},
		{"#ff0099", Rgba{1, 0, 0x99 / 255.0, 1}},
		// Eight-digit form with alpha.
		{"#ff0000ff", Rgba{1, 0, 0, 1}},
		{"#ff000000", Rgba{1, 0, 0, 0}},
		{"#3399ffcc", Rgba{0x33 / 255.0, 0x99 / 255.0, 0xff / 255.0, 0xcc / 255.0}},
		// Case-insensitive.
		{"#DeAdbEeF", Rgba32(0xdeadbeef)},
		// Surrounding whitespace is trimmed.
		{"  #f5f5f5ff  ", Rgba32(0xf5f5f5ff)},
	}
	for _, c := range cases {
		got, err := ParseHex(c.in)
		if err != nil {
			t.Errorf("ParseHex(%q): unexpected error %v", c.in, err)
			continue
		}
		if !rgbaEq(got, c.want) {
			t.Errorf("ParseHex(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseHexMalformed(t *testing.T) {
	malformed := []string{
		"",           // empty
		"#",          // no digits
		"ff0000",     // missing leading '#'
		"#ff00",      // four digits, not a supported form
		"#ff000",     // five digits
		"#ff0000f",   // seven digits
		"#ff0000fff", // nine digits
		"#gg0000",    // invalid digit
		"#ff00gg",    // invalid digit late in the string
		"#ff 0000",   // embedded whitespace
	}
	for _, in := range malformed {
		if _, err := ParseHex(in); err == nil {
			t.Errorf("ParseHex(%q): expected an error, got nil", in)
		}
	}
}
