package app

import "testing"

// BenchmarkRetainDispatch measures the cost of dispatching a notification to
// five observers. It must allocate nothing: dispatch is on the per-frame path,
// and the subscriber set holds subscribers in registration order so dispatch
// walks a slice directly.
func BenchmarkRetainDispatch(b *testing.B) {
	app := NewApp()
	defer app.Close()

	observed := New(app, func(cx *Context[counter]) counter { return counter{} })
	defer observed.Release()
	defer app.Flush()

	host := New(app, func(cx *Context[observerCount]) observerCount {
		for i := 0; i < 5; i++ {
			Observe(cx, observed, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {})
		}
		return observerCount{}
	})
	defer host.Release()
	app.Flush() // activate the subscriptions

	b.ResetTimer()
	for b.Loop() {
		app.observers.retain(observed.EntityID(), func(cb *observerHandler) bool {
			return true
		})
	}
}
