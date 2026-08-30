package app

import (
	"context"
	"testing"
)

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
