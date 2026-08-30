# Assignment: input

The package is done and it is the cleanest delivery in this project so far. All five
decisions from the plan review were honoured — no `KeyChar`, `secondary` behind
build tags, no registry, no tab navigation, no `reflect.DeepEqual`. I verified the
four hard cases directly and all four are right: the inner context wins, an outer
binding still fires from within an inner context, an unbound chord neither matches
nor pends, and `ctrl-k` reports pending while `ctrl-k ctrl-s` resolves.

Zero values are safe across `Keymap`, `FocusTree`, `DispatchTree` and `KeyBinding`,
with nil stacks and nil input, without being told twice. That has caught defects in
two other packages.

And `Explain` does what it was asked to:

    candidate 0: outer          depth 1  matched  shadowed by higher-precedence binding
    candidate 1: inner          depth 2  matched  WINNER: highest precedence (depth 2, index 1)
    candidate 2: wrong-context  depth -1 no       context predicate did not match
    candidate 3: disabled       depth -1 no       context predicate did not match

That is the answer to "why did this keystroke do that?", and it is the best
diagnostic anything here has.

Two small things and it is closed.

## 1 — sort.SliceStable uses reflection

Three call sites: `keymap.go:184`, `keymap.go:288`, `explain.go:141`.
`sort.SliceStable` goes through `reflectlite.Swapper`. `AGENTS.md` rules out
reflection on a per-frame path, and while keymap resolution is per keystroke rather
than per frame — so this is not costing anything real — `slices.SortStableFunc` is
the reflection-free equivalent and `slices` is already imported.

Worth doing for consistency more than speed: the same pattern was a genuine defect
in `app`, where dispatch ordering did sit on a per-frame path, and a rule that holds
everywhere is easier to apply than one with remembered exceptions.

## 2 — atomic on a single-goroutine path

`focus.go:15` allocates `FocusID`s with `atomic.AddUint64`. Everything above `app`
runs on one goroutine by design, and `input` is called from the UI goroutine only,
so the atomic is not buying anything.

Either it is needed, in which case say so in `doc.go` and explain what else touches
it — that would be news, and it would contradict the threading model the rest of the
framework rests on. Or it is not, in which case a plain counter is clearer, because
an atomic in single-threaded code reads as a claim that the code is shared.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

No reflective sort. The `FocusID` counter either loses its atomic or gains a comment
saying why it needs one.

Two conventional commits, staged by path.

## One process note

This assignment retired its own prompt before it was reviewed. The prompt going away
is what signals the package is finished and checked — doing it yourself removes the
step that has found something in every package so far, including two in `render` that
had passed every check.

Nothing was wrong here, which is the good outcome. But it was not knowable until
someone looked.
