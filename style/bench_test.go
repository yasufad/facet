package style

import (
	"testing"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/layout"
)

var benchSink Sink

type Sink struct {
	s Style
	r Refinement
	l layout.Style
}

// BenchmarkControlCopy measures a plain copy of Refinement as a control baseline.
func BenchmarkControlCopy(b *testing.B) {
	var src Refinement
	src.display = DisplayFlex
	src.flexShrink = 0.5

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dst := src
		benchSink.r = dst
	}
}

// BenchmarkSetFlexShrink measures setting a single scalar float property on *Refinement.
func BenchmarkSetFlexShrink(b *testing.B) {
	var r Refinement

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.SetFlexShrink(0.5)
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
	var r Refinement
	c := colour.Rgb(0x0000ff)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.SetDisplay(DisplayFlex)
		r.SetBackground(c)
		r.SetFlexShrink(0.8)
		r.SetFlexGrow(1.0)
	}
	benchSink.r = r
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
	r.SetFlexShrink(0.5)
	r.SetBackground(blue)
	r.SetFlexGrow(2.0)
	r.SetPadding(Px(8))
	r.SetWidth(Px(100))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s := base
		s.Refine(r)
		benchSink.s = s
	}
}

func BenchmarkRefinementMergeFrom(b *testing.B) {
	blue := colour.Rgb(0x0000ff)
	yellow := colour.Rgb(0xffff00)

	var r1 Refinement
	r1.SetFlexShrink(0.5)
	r1.SetBackground(blue)
	r1.SetPadding(Px(8))

	var r2 Refinement
	r2.SetFlexShrink(0.0)
	r2.SetBackground(yellow)
	r2.SetFlexGrow(1.0)
	r2.SetWidth(Px(100))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r := r1
		r.MergeFrom(&r2)
		benchSink.r = r
	}
}

func BenchmarkStyleToLayout(b *testing.B) {
	base := Default()
	base.Display = DisplayFlex
	base.FlexDirection = FlexDirectionColumn
	base.FlexGrow = 1.0
	base.Size = NewSize(Px(200), Px(100))
	base.Padding = NewEdges(Px(8), Px(8), Px(8), Px(8))
	base.Margin = NewEdges(Px(4), Px(4), Px(4), Px(4))
	remSize := geometry.Pixels(16)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l := base.ToLayout(remSize)
		benchSink.l = l
	}
}
