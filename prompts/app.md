# Assignment: app

Round two landed all three fixes and they hold up under independent probing.

    six exported guards      silent            all panic
    Context accessor check   6764 ns           2.8 ns
    dispatch, 5 observers    881.6 ns, 3 allocs   117.8 ns, 0 allocs

The subscriber slice is exactly right: registration order, walked directly,
dropped marked in O(1) and compacted on the next pass. That is the shape GPUI's
`BTreeMap` has, reached the Go way. Leave it alone.

What remains is the residue of the generation counter, which I suggested without
thinking it all the way through. It is a small job.

## Defect 1 — the generation counter cannot see a concurrent context

It catches a context used after its update ends. It does not catch one used from
another goroutine while that update is still running, because the generation still
matches:

    UpdateEntity(app, e, func(v *ps, cx *Context[ps]) {
        var wg sync.WaitGroup
        wg.Add(1)
        go func() { defer wg.Done(); cx.Notify() }()  // not caught
        wg.Wait()
    })

`go func() { cx.Notify() }()` inside an update is an ordinary mistake, and before
`c1aa1dd` it panicked. Nothing in the package currently says it no longer does.

Close it where closing it is free: a build-tagged variant that restores `checkUI`
on the Context accessors, behind `race`, or a `facet_debug` tag, or both. Release
builds keep the 2.8ns compare; `go test -race` and CI get full detection, where a
6µs check costs nothing that matters. Three mechanisms, each still placed where it
is cheap.

Add a test for the case above. It should pass under the tag and be skipped without
it.

## Defect 2 — the Threading doc claims more than the code does

`doc.go` opens the Threading section with:

    A context used from another goroutine panics with a message naming the
    mistake rather than corrupting state quietly.

That is no longer true in the during-update case. The bullets underneath describe
the two mechanisms accurately; it is the summary above them that overclaims.

State what actually holds: which mistake each mechanism catches, and which one is
only caught in a debug or race build. A reader deciding whether to trust the
invariant needs the shape of the hole, not a reassurance.

## Done when

    go build -o bin/ ./...
    go test ./app/
    go test ./internal/layering
    go vet ./app/
    gofmt -l $(go list -f '{{.Dir}}' ./...)

The concurrent-context case is caught under the debug or race build and has a test.
`doc.go` describes the guarantee that exists. One conventional commit per fix.

`layout` and `colour` are other agents' packages. Do not touch them.

## What to carry forward

Both rounds of review turned up documentation asserting a guarantee stronger than
the code provided — first "the check is cheap", now "a context used from another
goroutine panics". Each time the prose was written when it was true and left alone
when the code moved underneath it.

When you change how a guarantee is enforced, the sentence that describes the
guarantee is part of the change.

## Constraints that have not changed

The UI runs on one goroutine. Every exported entry point that touches entity state,
the effect queue or the subscriber sets panics when called from another goroutine.
The entity map stays free of mutexes: it is single-goroutine by design, and adding
one to make it safe from elsewhere would remove the reason the rest of the design
works. `app` knows nothing about drawing — no geometry, no colours, no elements.
Observers and subscribers fire in registration order, and dispatch does not
allocate.
