package element

import (
	"fmt"
	"testing"

	"github.com/yasufad/facet/layout"
)

// BenchmarkTextMeasureSameWidth measures the cost when the solver asks
// repeatedly for the same text at the same available width (cache hit).
func BenchmarkTextMeasureSameWidth(b *testing.B) {
	frame := newFakeFrame()
	frame.phase = phaseLayoutRequested

	txt := NewText("The quick brown fox jumps over the lazy dog").
		FontSize(16).
		LineHeight(20)

	nodeID := txt.RequestLayout(frame)
	measureCb := frame.measureCallbacks[nodeID]

	avail := layout.Size[layout.AvailableSpace]{
		Width:  layout.Definite(400),
		Height: layout.MaxContent(),
	}
	known := layout.Size[layout.OptF32]{}

	// Warm the cache once
	_ = measureCb(known, avail)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sz := measureCb(known, avail)
		if sz.Width <= 0 {
			b.Fatal("invalid width")
		}
	}
}

// BenchmarkTextMeasureVaryingWidth measures the cost when the available width
// changes on each call, requiring re-evaluation.
func BenchmarkTextMeasureVaryingWidth(b *testing.B) {
	frame := newFakeFrame()
	frame.phase = phaseLayoutRequested

	txt := NewText("The quick brown fox jumps over the lazy dog").
		FontSize(16).
		LineHeight(20)

	nodeID := txt.RequestLayout(frame)
	measureCb := frame.measureCallbacks[nodeID]

	known := layout.Size[layout.OptF32]{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		avail := layout.Size[layout.AvailableSpace]{
			Width:  layout.Definite(float32(200 + (i % 50))),
			Height: layout.MaxContent(),
		}
		sz := measureCb(known, avail)
		if sz.Width <= 0 {
			b.Fatal("invalid width")
		}
	}
}

// BenchmarkTextMeasureFreshPerFrame benchmarks a full fresh element creation and measurement cycle.
func BenchmarkTextMeasureFreshPerFrame(b *testing.B) {
	frame := newFakeFrame()
	frame.phase = phaseLayoutRequested

	avail := layout.Size[layout.AvailableSpace]{
		Width:  layout.Definite(400),
		Height: layout.MaxContent(),
	}
	known := layout.Size[layout.OptF32]{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		txt := NewText(fmt.Sprintf("Label %d", i%100)).
			FontSize(14).
			LineHeight(18)
		nodeID := txt.RequestLayout(frame)
		sz := frame.measureCallbacks[nodeID](known, avail)
		if sz.Width <= 0 {
			b.Fatal("invalid width")
		}
	}
}
