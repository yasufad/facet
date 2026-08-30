package style

import (
	"testing"

	"github.com/yasufad/facet/colour"
)

var benchStyle SinkStyle

type SinkStyle struct {
	s Style
	r Refinement
}

func BenchmarkStyleRefineEmpty(b *testing.B) {
	base := Default()
	empty := Refinement{}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := base
		s.Refine(empty)
		benchStyle.s = s
	}
}

func BenchmarkStyleRefineNonEmpty(b *testing.B) {
	base := Default()
	blue := colour.Rgb(0x0000ff)
	r := Refinement{}.Opacity(0.5).Bg(blue).FlexGrow(2.0)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := base
		s.Refine(r)
		benchStyle.s = s
	}
}

func BenchmarkRefinementMerge(b *testing.B) {
	blue := colour.Rgb(0x0000ff)
	yellow := colour.Rgb(0xffff00)

	r1 := Refinement{}.Opacity(0.5).Bg(blue)
	r2 := Refinement{}.Opacity(0.0).Bg(yellow).FlexGrow(1.0)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		merged := r1.Merge(r2)
		benchStyle.r = merged
	}
}
