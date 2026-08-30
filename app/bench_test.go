package app

import (
	"testing"
)

// BenchmarkFrameSimulatesUpdateNotify measures the cost of the threading check
// across a frame that updates and notifies many entities. Each UpdateEntity
// opens an update boundary and each Notify queues an effect; both currently
// call checkUI. This benchmark tracks the per-frame overhead so the boundary
// approach can be measured against it.
func BenchmarkFrameSimulatesUpdateNotify(b *testing.B) {
	app := NewApp()
	defer app.Close()

	entities := make([]Entity[counter], 1000)
	for i := range entities {
		entities[i] = New(app, func(cx *Context[counter]) counter {
			return counter{}
		})
	}
	defer func() {
		for _, e := range entities {
			e.Release()
		}
		app.Flush()
	}()

	b.ResetTimer()
	for b.Loop() {
		for i := range entities {
			e := entities[i]
			UpdateEntity(app, e, func(v *counter, cx *Context[counter]) {
				v.count++
				cx.Notify()
			})
		}
	}
}
