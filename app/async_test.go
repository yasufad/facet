package app

import (
	"sync"
	"testing"
	"time"
)

// TestAsyncUpdateAfterCloseReturnsError pins the fix for the deadlock this
// package had: run enqueued a closure whose own shutdown check lived inside
// it, so the check only ran if something drained the queue - and once
// App.Close had stopped the foreground executor, nothing ever would again.
// AsyncApp.Update from a background goroutine, called after Close, must
// return the documented error instead of blocking forever.
func TestAsyncUpdateAfterCloseReturnsError(t *testing.T) {
	app := NewApp()
	c := newCounter(t, app, 0)
	async := app.Async()
	app.Close()

	done := make(chan error, 1)
	go func() {
		done <- AsyncUpdateEntity(async, c, func(v *counter, cx *Context[counter]) {
			v.count++
		})
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Update after Close should report an error, not succeed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Update after Close blocked instead of returning an error: the foreground queue is never drained again once Close has stopped it")
	}
}

// TestAsyncReadAfterCloseReturnsError is AsyncRead's counterpart to the above:
// it goes through the same run and must not block either.
func TestAsyncReadAfterCloseReturnsError(t *testing.T) {
	app := NewApp()
	c := newCounter(t, app, 0)
	async := app.Async()
	app.Close()

	done := make(chan error, 1)
	go func() {
		_, err := AsyncReadEntity(async, c)
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Read after Close should report an error, not succeed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Read after Close blocked instead of returning an error")
	}
}

// TestConcurrentShutdownRace hammers AsyncApp.Update from background
// goroutines while the UI goroutine drains and then closes the App, so that
// some in-flight jobs are still queued (enqueued successfully before Close
// stopped the executor) when App.Close marks the App shut down and a final
// Drain runs them. That is the window in which a queued job's own shutdown
// check (rc.isShutdown, read from whichever goroutine happens to run the
// job through Drain) races against markShutdown's write (from the goroutine
// calling Close) if that check is not made under refCounts.mu. Run with
// -race: it must not report a data race, and none of the background
// goroutines may hang past the timeout regardless.
func TestConcurrentShutdownRace(t *testing.T) {
	app := NewApp()
	c := newCounter(t, app, 0)
	async := app.Async()

	const workers = 8
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_ = AsyncUpdateEntity(async, c, func(v *counter, cx *Context[counter]) {
					v.count++
				})
			}
		}()
	}

	// Drain concurrently with the workers enqueueing, then close while they
	// are still in flight, then drain once more to run whatever was queued
	// immediately before the close - the case item 2 needs reached rather
	// than deadlocked past.
	for i := 0; i < 50; i++ {
		app.Foreground().Drain()
	}
	app.Close()
	app.Foreground().Drain()

	close(stop)
	waited := make(chan struct{})
	go func() {
		wg.Wait()
		close(waited)
	}()
	select {
	case <-waited:
	case <-time.After(5 * time.Second):
		t.Fatal("a background worker did not return after Close: Update is deadlocking again")
	}
}
