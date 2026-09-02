package app

import (
	"context"
	"runtime"
	"sync"
)

// taskResult is the outcome of a Task: either a value or an error (from a
// panic or cancellation). It travels through a buffered channel so the
// producer can send without a waiter and the work runs to completion whether
// or not the result is consumed.
type taskResult[R any] struct {
	value R
	err   error
}

// Task is a handle to work that completes later. It is the unit the executors
// return.
//
// Await blocks until the work completes and returns its result; it must not be
// called on the UI goroutine, which must not block. Detach lets the work run
// to completion and discards the result. Cancel cancels the work through its
// context.
//
// Go has no RAII, so a Task is not cancelled automatically when it goes out of
// scope. Hold a Task only as long as you need its result, and Cancel or Detach
// it when you are done.
type Task[R any] struct {
	result <-chan taskResult[R]
	cancel context.CancelFunc
}

// Await blocks until the work completes and returns its result. The error is
// non-nil if the work was cancelled or panicked. Await must not be called on
// the UI goroutine.
func (t Task[R]) Await() (R, error) {
	res := <-t.result
	return res.value, res.err
}

// Detach lets the work run to completion and discards the result. The Task
// holds no further resources after Detach returns.
func (t Task[R]) Detach() {
	// The producer holds the send side of the buffered result channel, so the
	// work continues independently of this handle. Nothing to do here.
}

// Cancel cancels the work through its context. The Task's result, when it
// arrives, will carry a cancellation error.
func (t Task[R]) Cancel() {
	if t.cancel != nil {
		t.cancel()
	}
}

// Then runs f on the UI goroutine, through async, when the work completes
// successfully. If the work is cancelled or panics, f is not run. The
// continuation is dispatched onto the foreground executor, so it touches state
// on the UI goroutine.
func (t Task[R]) Then(async *AsyncApp, f func(r R, async *AsyncApp)) {
	go func() {
		res, ok := <-t.result
		if !ok || res.err != nil {
			return
		}
		async.app.fg.enqueue(func() {
			f(res.value, async)
		})
	}()
}

// ForegroundExecutor runs tasks on the UI goroutine. Spawned closures are
// queued and run when the UI loop drains the queue — typically between frames
// and platform events. The queue is goroutine-safe because background tasks
// dispatch their results onto it from other goroutines.
type ForegroundExecutor struct {
	mu     sync.Mutex
	queue  []func()
	wake   chan struct{}
	closed bool
}

func newForegroundExecutor() *ForegroundExecutor {
	return &ForegroundExecutor{
		wake: make(chan struct{}, 1),
	}
}

// Pending returns a channel that receives when there is queued foreground work
// to drain. A platform event loop selects on it to interleave task progress
// with native events; tests call Drain directly. The channel closes when the
// executor is stopped, so a `for range` over it terminates instead of
// blocking forever.
func (fg *ForegroundExecutor) Pending() <-chan struct{} {
	return fg.wake
}

// spawn queues f to run on the UI goroutine and returns a Task for its result.
// It is a top-level function because Go does not permit methods to declare
// type parameters.
func fgSpawn[R any](fg *ForegroundExecutor, f func() R) Task[R] {
	res := make(chan taskResult[R], 1)
	ctx, cancel := context.WithCancel(context.Background())
	job := func() {
		if err := ctx.Err(); err != nil {
			res <- taskResult[R]{err: err}
			return
		}
		defer func() {
			if r := recover(); r != nil {
				res <- taskResult[R]{err: &panicError{recovered: r}}
			}
		}()
		res <- taskResult[R]{value: f()}
	}
	fg.enqueue(job)
	return Task[R]{result: res, cancel: cancel}
}

// enqueue appends f to the queue and wakes a waiting Pending() reader. A call
// after stop is a silent no-op: the closed flag rules out both queueing and
// sending on the (now closed) wake channel, checked under the same lock stop
// holds while closing it, so the two never race.
func (fg *ForegroundExecutor) enqueue(f func()) {
	fg.mu.Lock()
	defer fg.mu.Unlock()
	if fg.closed {
		return
	}
	fg.queue = append(fg.queue, f)
	select {
	case fg.wake <- struct{}{}:
	default:
	}
}

// Drain runs every queued closure until the queue is empty. Closures queued
// while draining are run in the same call. It must be called on the UI
// goroutine.
func (fg *ForegroundExecutor) Drain() {
	for {
		fg.mu.Lock()
		if len(fg.queue) == 0 {
			fg.mu.Unlock()
			return
		}
		batch := fg.queue
		fg.queue = nil
		fg.mu.Unlock()
		for _, f := range batch {
			f()
		}
	}
}

// RunOne runs a single queued closure and reports whether one was available.
func (fg *ForegroundExecutor) RunOne() bool {
	fg.mu.Lock()
	if len(fg.queue) == 0 {
		fg.mu.Unlock()
		return false
	}
	f := fg.queue[0]
	fg.queue = fg.queue[1:]
	fg.mu.Unlock()
	f()
	return true
}

// stop closes wake, so a `for range Pending()` loop (window's dispatch
// goroutine) terminates instead of holding the platform, App and window alive
// for the life of the process. It is idempotent: a second Close on the App
// must not close an already-closed channel and panic.
func (fg *ForegroundExecutor) stop() {
	fg.mu.Lock()
	defer fg.mu.Unlock()
	if fg.closed {
		return
	}
	fg.closed = true
	close(fg.wake)
}

// BackgroundExecutor runs tasks on a bounded pool of goroutines. Work that
// must touch entity state returns to the foreground through AsyncApp.
type BackgroundExecutor struct {
	jobs chan func()
	wg   sync.WaitGroup
	done chan struct{}
	once sync.Once
}

func newBackgroundExecutor() *BackgroundExecutor {
	bg := &BackgroundExecutor{
		jobs: make(chan func(), 256),
		done: make(chan struct{}),
	}
	workers := runtime.GOMAXPROCS(0)
	for i := 0; i < workers; i++ {
		bg.wg.Add(1)
		go bg.worker()
	}
	return bg
}

func (bg *BackgroundExecutor) worker() {
	defer bg.wg.Done()
	for job := range bg.jobs {
		job()
	}
}

// BackgroundSpawn runs f on a background goroutine with a context that is
// cancelled when the returned Task is cancelled. The result is delivered to
// the Task. If the executor has been stopped, the Task completes with a
// shutdown error.
//
// It is a top-level function because Go does not permit methods to declare
// type parameters.
func BackgroundSpawn[R any](bg *BackgroundExecutor, f func(ctx context.Context) R) Task[R] {
	res := make(chan taskResult[R], 1)
	ctx, cancel := context.WithCancel(context.Background())
	job := func() {
		if err := ctx.Err(); err != nil {
			res <- taskResult[R]{err: err}
			return
		}
		defer func() {
			if r := recover(); r != nil {
				res <- taskResult[R]{err: &panicError{recovered: r}}
			}
		}()
		res <- taskResult[R]{value: f(ctx)}
	}
	select {
	case bg.jobs <- job:
	case <-bg.done:
		res <- taskResult[R]{err: &shutdownError{}}
	}
	return Task[R]{result: res, cancel: cancel}
}

func (bg *BackgroundExecutor) stop() {
	bg.once.Do(func() {
		close(bg.done)
		close(bg.jobs)
	})
	bg.wg.Wait()
}

type shutdownError struct{}

func (e *shutdownError) Error() string { return "app: executor shut down" }

// panicError wraps a recovered panic so it travels through a Task as an error.
type panicError struct {
	recovered any
}

func (e *panicError) Error() string {
	return "app: background task panicked"
}
