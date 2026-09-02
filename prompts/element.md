# element: the seam is right, and nothing pins it to the slot it fits

The listener work is correct. I read it at HEAD, broke it, and confirmed the tests fail
the way they should. `defer entity.Release()` on the panic path, the dropped-view branch
returning unhandled with its own test, and a doc comment that says plainly this is a
mitigation rather than sugar — all of that is what the round asked for.

Two gaps, both in what is tested rather than in what was written, and then the rest of the
round.

## 1. Nothing tests the seam the design exists for

The whole argument for `PhasedListener`'s shape is that it drops into the existing slots
with no signature change anywhere. Your report says so. No test says so — every test calls
the returned closure directly.

I checked the claim and it holds. All three fit:

```go
var _ input.KeyEventHandler = element.PhasedListener(cx, ...)
var _ input.PointerEventHandler = element.PhasedListener(cx, ...)
var _ input.WheelEventHandler = element.PhasedListener(cx, ...)
element.NewDiv().OnClick(element.Listener(cx, ...))
```

That is four lines and it belongs in the package, not in my scratch directory. It is a
compile-time assertion, so it costs nothing at run time and fails loudly the day someone
adds a parameter to one of those handler types.

This is the `AGENTS.md` rule about setting one thing and checking another. The listeners
were tested in isolation; they exist to fit somewhere.

## 2. A missing Release breaks no test

I removed `defer entity.Release()` from both adapters and the whole package still passed.

The code is right. The point is that the line most likely to be lost in a future edit is
unguarded, in the one package that opts into manual reference counting on a per-event
path. `docs/audit.md` names this as the framework's central lifetime hazard: a missed
`Release` leaks the entity and every observer registered against it, with no diagnostic.
An adapter that runs on every click is where that would accumulate fastest.

Pin the balance. Register a listener, fire it many times, and assert the entity still
drops when its last strong handle goes — a leaked upgrade keeps it alive and the assertion
catches it. Break the `Release` and confirm your test fails.

## 3. Div.Prepaint is unblocked

`window` has landed. `PushClip` and `PopClip` are legal in prepaint, hit regions are
intersected with the prepaint clip stack, and there is a `facet_debug` balance assertion
on that stack.

So push and pop around children in `Div.Prepaint` exactly as `Div.Paint` already does, and
update the phase rules in the `Frame` doc comment in the same commit — the comment states
the guarantee and is part of it.

The test worth writing is the one that was impossible before: register a hit region inside
an overflow-hidden container, put the pointer where the clip excludes it, and assert the
hit misses. A button scrolled out of a `ScrollView` has been invisible and clickable for
the whole life of this repository.

## 4. Carried forward, unchanged

**`TextLayout`.** `ui` is still waiting and it is now the last thing between the text
field and a caret. `XForIndex`, `IndexForX`, `ClosestIndexForX`, wrapping the shaped line
so `ui` never learns `text`.

**`Text` re-shapes for a width that changes nothing.** `ShapeLine` takes no width and
wraps nothing, so the output is identical at every available width, and
`t.lastAvailWidth != avail.Width` throws away a correct answer several times per solve.
Invalidate on content and resolved style.

`text` has since landed a line cache that makes the repeat call roughly 190 times cheaper,
so the absolute cost of the redundant call has collapsed. Do it anyway: the call is still
wrong, and a cache making a mistake affordable is not the same as not making it.

**Style resolved three times per frame.** Resolve once in `RequestLayout`, carry it on the
element, layer the pseudo-state refinements onto a copy in `Paint`. Roughly 4% of element
cost, so it is a tidy-up rather than a wall — the prompt said that last round and it is
still true.

## 5. Then ui and the examples

You stopped before `ui` and `examples/button` to let me see the seam. That was right and
the seam is good.

Take them now. `ui.Button` migrates to `Listener` with a test that changes real entity
state through a dispatch and reads it back, and `examples/button` becomes the thing it has
always claimed to be. It is one landing with the migration; `prompts/ui.md` covers the
widget side and whoever holds `ui` writes the cross-stack test in `internal/integration`.

Report before you start, only so we do not both move `ui` at once.

## On the shared index

You hit `git add` picking up another agent's concurrently staged file twice, caught both
with `git show --name-only`, and fixed them. That is the check working exactly as
`AGENTS.md` intends, and it is worth saying that you caught it rather than that it
happened.

## Done when

    go test ./element ./element/elementtest
    go test -tags facet_debug ./element

pass, with the compile-time slot assertions, a test that fails if `Release` is dropped,
and the prepaint clip test that fails if the clip is ignored.
