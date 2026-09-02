package app

import (
	"context"
	"testing"
	"time"
)

func TestBackgroundSpawnDeliversResult(t *testing.T) {
	app := NewApp()
	defer app.Close()

	task := BackgroundSpawn(app.Background(), func(ctx context.Context) int {
		return 21 * 2
	})
	got, err := drainAndAwait(app, task)
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	if got != 42 {
		t.Fatalf("got %d, want 42", got)
	}
}

func TestBackgroundSpawnCancel(t *testing.T) {
	app := NewApp()
	defer app.Close()

	task := BackgroundSpawn(app.Background(), func(ctx context.Context) int {
		<-ctx.Done()
		return 0
	})
	task.Cancel()
	if _, err := drainAndAwait(app, task); err == nil {
		t.Fatal("cancelled task should report an error")
	}
}

func TestForegroundSpawnRunsOnUIGoroutine(t *testing.T) {
	app := NewApp()
	defer app.Close()

	async := app.Async()
	ran := false
	task := AsyncSpawn(async, func(async *AsyncApp) bool {
		// Runs on the UI goroutine during Drain.
		ran = goroutineID() == app.uiGoroutine
		return ran
	})
	// Detach the foreground task; it completes when we drain.
	task.Detach()
	app.Foreground().Drain()
	if !ran {
		t.Fatal("foreground task did not run on the UI goroutine")
	}
}

func TestForegroundExecutorStopClosesWakeAndIsIdempotent(t *testing.T) {
	// window.New ranges over Pending() forever; that range must terminate
	// when the executor stops, and a second stop (App.Close called twice)
	// must not close an already-closed channel and panic.
	fg := newForegroundExecutor(goroutineID())
	fg.stop()

	select {
	case _, ok := <-fg.Pending():
		if ok {
			t.Fatal("Pending() should be closed, not deliver a value")
		}
	default:
		t.Fatal("Pending() should be immediately closed after stop")
	}

	fg.stop() // must not panic

	fg.enqueue(func() {}) // must not panic or block
}

func TestTaskThenRunsOnForeground(t *testing.T) {
	app := NewApp()
	defer app.Close()

	async := app.Async()
	task := BackgroundSpawn(app.Background(), func(ctx context.Context) int {
		return 7
	})
	got := 0
	task.Then(async, func(r int, async *AsyncApp) {
		_ = async.Update(func(app *App) {
			got = r * 3
		})
	})
	// Poll: drain the foreground queue until the Then continuation has run.
	// The background task completes asynchronously and enqueues the
	// continuation on the foreground executor; draining picks it up.
	for i := 0; i < 200 && got == 0; i++ {
		app.Foreground().Drain()
		time.Sleep(10 * time.Millisecond)
	}
	if got != 21 {
		t.Fatalf("got %d, want 21", got)
	}
}
