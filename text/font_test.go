package text

import (
	"testing"

	"github.com/yasufad/facet/geometry"
)

// TestSystemFamilies checks that NewSystem discovers at least one font family.
func TestSystemFamilies(t *testing.T) {
	s := newTestSystem(t)
	families := s.Families()
	if len(families) == 0 {
		t.Fatal("no system font families found")
	}
}

// TestResolveFace checks that Resolve returns a valid face for a common
// family.
func TestResolveFace(t *testing.T) {
	s := newTestSystem(t)
	face := s.Resolve(FontRequest{Families: []string{"Arial", "Helvetica", "DejaVu Sans", "Liberation Sans"}})
	if !face.valid() {
		t.Fatal("Resolve returned invalid face")
	}
}

// TestResolveDefaults checks that a zero-weight request resolves to a normal
// face.
func TestResolveDefaults(t *testing.T) {
	s := newTestSystem(t)
	face := s.Resolve(FontRequest{Families: []string{"Arial", "DejaVu Sans"}})
	if !face.valid() {
		t.Fatal("Resolve returned invalid face for default weight")
	}
}

// TestMetrics checks that Metrics returns sensible values for a known face.
func TestMetrics(t *testing.T) {
	s := newTestSystem(t)
	face := s.Resolve(FontRequest{Families: []string{"Arial", "DejaVu Sans"}})
	if !face.valid() {
		t.Skip("no face available")
	}
	m := face.Metrics(geometry.Pixels(16))
	if m.Ascent <= 0 {
		t.Fatalf("ascent %v, expected positive", m.Ascent)
	}
	if m.Descent >= 0 {
		t.Fatalf("descent %v, expected negative", m.Descent)
	}
	if m.UnitsPerEm == 0 {
		t.Fatal("units per em is 0")
	}
}

// TestLineHeight checks that LineHeight is positive and equals ascent minus
// descent plus line gap.
func TestLineHeight(t *testing.T) {
	s := newTestSystem(t)
	face := s.Resolve(FontRequest{Families: []string{"Arial", "DejaVu Sans"}})
	if !face.valid() {
		t.Skip("no face available")
	}
	size := geometry.Pixels(16)
	lh := face.LineHeight(size)
	m := face.Metrics(size)
	expected := m.Ascent - m.Descent + m.LineGap
	if lh != expected {
		t.Fatalf("LineHeight %v, expected %v", lh, expected)
	}
	if lh <= 0 {
		t.Fatalf("LineHeight %v, expected positive", lh)
	}
}

// TestFontRequestDefaults checks that withDefaults fills in zero values.
func TestFontRequestDefaults(t *testing.T) {
	r := FontRequest{Family: "Arial"}
	d := r.withDefaults()
	if d.Weight != WeightNormal {
		t.Fatalf("default weight %v, expected %v", d.Weight, WeightNormal)
	}
	if d.Stretch != StretchNormal {
		t.Fatalf("default stretch %v, expected %v", d.Stretch, StretchNormal)
	}
}

// TestFontRequestFamilies checks that families returns the primary family
// first, followed by fallbacks, with empties dropped.
func TestFontRequestFamilies(t *testing.T) {
	r := FontRequest{Family: "Arial", Families: []string{"", "Helvetica", "DejaVu"}}
	got := r.families()
	want := []string{"Arial", "Helvetica", "DejaVu"}
	if len(got) != len(want) {
		t.Fatalf("families %v, expected %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("families[%d] %q, expected %q", i, got[i], want[i])
		}
	}
}

// TestFontRequestFamiliesEmpty checks that an empty request yields a
// non-empty family list (the matcher needs at least one entry).
func TestFontRequestFamiliesEmpty(t *testing.T) {
	r := FontRequest{}
	got := r.families()
	if len(got) == 0 {
		t.Fatal("empty request yielded no families")
	}
}
