package style

import (
	"testing"

	"github.com/yasufad/facet/colour"
)

var benchSink Sink

type Sink struct {
	s Style
	r Refinement
}

// BenchmarkControlCopy48 measures a plain copy of a 48-byte struct as a control baseline.
func BenchmarkControlCopy48(b *testing.B) {
	var src Refinement
	src.display = DisplayFlex
	src.opacity = 0.5

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dst := src
		benchSink.r = dst
	}
}

// BenchmarkSetOpacity measures setting a single scalar float property on *Refinement.
func BenchmarkSetOpacity(b *testing.B) {
	var r Refinement

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.SetOpacity(0.5)
	}
	benchSink.r = r
}

// BenchmarkSetBackground measures setting a colour.Rgba background on *Refinement.
func BenchmarkSetBackground(b *testing.B) {
	var r Refinement
	c := colour.Rgb(0x0000ff)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.SetBackground(c)
	}
	benchSink.r = r
}

// BenchmarkSetSequence4 measures setting 4 properties in sequence on an addressable *Refinement.
func BenchmarkSetSequence4(b *testing.B) {
	c := colour.Rgb(0x0000ff)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var r Refinement
		r.SetDisplay(DisplayFlex)
		r.SetBackground(c)
		r.SetOpacity(0.8)
		r.SetFlexGrow(1.0)
		benchSink.r = r
	}
}

func BenchmarkStyleRefineEmpty(b *testing.B) {
	base := Default()
	empty := Refinement{}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := base
		s.Refine(empty)
		benchSink.s = s
	}
}

func BenchmarkStyleRefineNonEmpty(b *testing.B) {
	base := Default()
	blue := colour.Rgb(0x0000ff)
	var r Refinement
	r.SetOpacity(0.5)
	r.SetBackground(blue)
	r.SetFlexGrow(2.0)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := base
		s.Refine(r)
		benchSink.s = s
	}
}

func BenchmarkRefinementMerge(b *testing.B) {
	blue := colour.Rgb(0x0000ff)
	yellow := colour.Rgb(0xffff00)

	var r1 Refinement
	r1.SetOpacity(0.5)
	r1.SetBackground(blue)

	var r2 Refinement
	r2.SetOpacity(0.0)
	r2.SetBackground(yellow)
	r2.SetFlexGrow(1.0)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		merged := r1.Merge(r2)
		benchSink.r = merged
	}
}
