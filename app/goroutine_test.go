package app

import (
	"context"
	"reflect"
	"sync"
	"testing"
)

// reflectTypeOf returns the reflect.Type for Evt, matching the pattern used
// by Emit and Subscribe in the production code.
func reflectTypeOf[Evt any]() reflect.Type {
	return reflect.TypeOf((*Evt)(nil))
}

func TestContextFromWrongGoroutinePanics(t *testing.T) {
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 0)
	defer c.Release()

	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Update from another goroutine did not panic")
			}
			close(done)
		}()
		c.Update(app, func(v *counter, cx *Context[counter]) { v.count++ })
	}()
	<-done
}

func TestFlushFromWrongGoroutinePanics(t *testing.T) {
	app := NewApp()
	defer app.Close()

	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Flush from another goroutine did not panic")
			}
			close(done)
		}()
		app.Flush()
	}()
	<-done
}

func TestNotifyFromWrongGoroutinePanics(t *testing.T) {
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 0)
	defer c.Release()

	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Notify from another goroutine did not panic")
			}
			close(done)
		}()
		app.Notify(c.EntityID())
	}()
	<-done
}

func TestEmitFromWrongGoroutinePanics(t *testing.T) {
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 0)
	defer c.Release()

	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Emit from another goroutine did not panic")
			}
			close(done)
		}()
		app.Emit(c.EntityID(), &pingEvent{}, reflectTypeOf[pingEvent]())
	}()
	<-done
}

func TestDeferFromWrongGoroutinePanics(t *testing.T) {
	app := NewApp()
	defer app.Close()

	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Defer from another goroutine did not panic")
			}
			close(done)
		}()
		app.Defer(func(*App) {})
	}()
	<-done
}

func TestObserveFromWrongGoroutinePanics(t *testing.T) {
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 0)
	defer c.Release()

	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Observe from another goroutine did not panic")
			}
			close(done)
		}()
		app.Observe(anyEntity{id: c.EntityID()}, func(app *App) bool { return true })
	}()
	<-done
}

func TestSubscribeFromWrongGoroutinePanics(t *testing.T) {
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 0)
	defer c.Release()

	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Subscribe from another goroutine did not panic")
			}
			close(done)
		}()
		app.Subscribe(anyEntity{id: c.EntityID()}, reflectTypeOf[pingEvent](), func(app *App, event any) bool { return true })
	}()
	<-done
}

func TestOnReleaseFromWrongGoroutinePanics(t *testing.T) {
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 0)
	defer c.Release()

	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("OnRelease from another goroutine did not panic")
			}
			close(done)
		}()
		app.OnRelease(anyEntity{id: c.EntityID()}, func(value any, app *App) {})
	}()
	<-done
}

func TestReadFromWrongGoroutinePanics(t *testing.T) {
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 0)
	defer c.Release()

	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Read from another goroutine did not panic")
			}
			close(done)
		}()
		_ = c.Read(app)
	}()
	<-done
}

func TestContextUsedAfterUpdatePanics(t *testing.T) {
	// A Context that escapes its update must panic when used. The generation
	// counter catches this: after the update ends, the generation has moved
	// on, and the context's generation no longer matches.
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 0)
	defer c.Release()

	var escaped *Context[counter]
	UpdateEntity(app, c, func(v *counter, cx *Context[counter]) {
		escaped = cx // store the context for later use
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("using a context after its update did not panic")
		}
	}()
	escaped.Notify()
}

func TestConcurrentContextUseDuringUpdatePanics(t *testing.T) {
	// A Context used from another goroutine while its update is still running
	// must panic. The generation counter alone cannot see this (the generation
	// still matches), so the full goroutine check is needed. That check runs
	// in a facet_debug build; in a release build the case is not caught at
	// the accessor, so the test skips.
	if !debugChecks {
		t.Skip("concurrent context detection requires -tags facet_debug")
	}

	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 0)
	defer c.Release()

	UpdateEntity(app, c, func(v *counter, cx *Context[counter]) {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r == nil {
					t.Error("concurrent context use did not panic")
				}
			}()
			cx.Notify()
		}()
		wg.Wait()
	})
}

func TestAsyncAppReachesStateFromBackground(t *testing.T) {
	// A background task reaches entity state through AsyncApp, which marshals
	// onto the UI goroutine. This must not panic and must see the live value.
	app := NewApp()
	defer app.Close()

	c := newCounter(t, app, 5)
	async := app.Async()

	task := BackgroundSpawn(app.Background(), func(ctx context.Context) int {
		v, err := AsyncReadEntity(async, c)
		if err != nil {
			t.Errorf("AsyncReadEntity: %v", err)
			return 0
		}
		return v.count
	})
	got, err := drainAndAwait(app, task)
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	if got != 5 {
		t.Fatalf("got %d, want 5", got)
	}
	c.Release()
}

// drainAndAwait runs the background task to completion while keeping the
// foreground executor drained, so marshalled reads and updates progress. It
// must be called on the UI goroutine.
func drainAndAwait[R any](app *App, task Task[R]) (R, error) {
	type result struct {
		v   R
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		v, err := task.Await()
		resCh <- result{v, err}
	}()
	for {
		app.Foreground().Drain()
		select {
		case r := <-resCh:
			return r.v, r.err
		case <-app.Foreground().Pending():
		}
	}
}

func BenchmarkGoroutineID(b *testing.B) {
	for b.Loop() {
		_ = goroutineID()
	}
}

func BenchmarkCheckUI(b *testing.B) {
	app := NewApp()
	defer app.Close()
	b.ResetTimer()
	for b.Loop() {
		app.checkUI()
	}
}

func BenchmarkCheckGeneration(b *testing.B) {
	app := NewApp()
	defer app.Close()
	c := New(app, func(cx *Context[counter]) counter { return counter{} })
	defer c.Release()
	// Construct a context with a matching generation directly, to measure
	// just the integer compare without the update boundary overhead.
	cx := &Context[counter]{app: app, self: c.Downgrade(), generation: app.generation}
	b.ResetTimer()
	for b.Loop() {
		cx.checkGeneration()
	}
}
