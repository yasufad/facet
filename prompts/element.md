# element: an event handler cannot reach the state it belongs to

This is the defect that blocks the framework, and it is yours. Read
`docs/audit.md` first; it has the reproduction.

## The problem

`examples/button` is written the way anyone would write it:

```go
func (c *CounterView) Render(cx *app.Context[CounterView]) element.Element {
    return element.NewDiv().Children(
        ui.NewButton("Click Me").
            OnClick(func(event element.ClickEvent) bool {
                c.count++
                cx.Notify()
                return true
            }),
    )
}
```

`View[T].Render` calls `app.UpdateEntity`, so `Render` runs inside an update. `c` is the
pointer that update leased out and `cx` is a context whose generation expires the moment
it returns. The closure is stored in the dispatch tree and invoked from
`window.DispatchEvent`, outside any update. Both halves fail:

    PANIC when the click handler ran: app: context used after its update has ended
    stored count after the click handler ran: 0 (expected 1)

`func(ClickEvent) bool` gives the handler no route back into the entity. Capturing is the
only thing a caller can do and capturing is wrong, so there is currently no correct way to
write a handler on a view. Every phased handler — key, pointer, wheel, text — has the same
shape and the same problem.

`ui.Button`'s test passes because it sets a test-local `clicked` and never goes near an
entity. That is the pattern `AGENTS.md` warns about under "interaction needs a test that
sets one thing and checks another", and the warning was written before this and did not
catch it. Whatever you add here needs a test that mutates real entity state through a real
dispatch and reads it back.

## The decision

Two helpers in `element`. They close over the weak handle and re-enter the update when
the event fires, so the callback is handed live state and a live context instead of
captured ones.

```go
// Listener adapts a handler that needs its view's state into one the element
// tree can store. The entity is upgraded and updated at dispatch time.
func Listener[T, E any](
    cx *app.Context[T],
    f func(v *T, e E, cx *app.Context[T]) bool,
) func(E) bool

// PhasedListener is Listener for the handlers that carry a dispatch phase.
func PhasedListener[T, E any](
    cx *app.Context[T],
    f func(v *T, e E, phase input.DispatchPhase, cx *app.Context[T]) bool,
) func(E, input.DispatchPhase) bool
```

Both take `cx.WeakEntity()` and `cx.App()` at registration, and at dispatch time upgrade,
`app.UpdateEntity`, call `f`, release. A handler whose view has been dropped upgrades to
nothing and reports the event unhandled; it must not panic.

`PhasedListener`'s return type is assignable to `input.KeyEventHandler` and its three
siblings, so no signature on `Div` changes and neither does `input` or `window`.

Two functions rather than one because the phased handlers take two arguments and Go
cannot abstract over arity. A single `Bind` returning an updater would cover both shapes
in one function, and I decided against it: it leaves the handler body outside the update,
so `c.count++` still compiles next to it and still silently does nothing. Putting `v` and
`cx` in the parameter list is what makes the correct thing the one that is in front of
you.

I also considered taking `Render` out of the update entirely — hand it
`(self app.Entity[T], cx *app.App)` and let handlers use `self.Update`, which is already
the framework's correct public idiom and needs no new API at all. It is a cleaner fix to
the root cause. I rejected it because `Render` would lose its `Context`, and
`docs/architecture.md` leans on having one there: a view that reads a second entity
observes it and notifies itself, "explicit and one line". That line needs a `Context`.

So `Render` still runs inside an update, and that remains the underlying hazard. These
helpers are the mitigation at the one place it bites. Say so in the doc comment, so
nobody reads them as sugar.

## Also outstanding

**`TextLayout`.** Carried from the last round, still needed, and `ui` is waiting on it.
`element.Text` becomes queryable without `ui` learning `text`:

```go
type TextLayout struct{ ... }              // element's type, wrapping the shaped line
func (t *Text) Layout() TextLayout
func (l TextLayout) XForIndex(byteIndex int) geometry.Pixels
func (l TextLayout) IndexForX(x geometry.Pixels) (int, bool)
func (l TextLayout) ClosestIndexForX(x geometry.Pixels) int
```

`text.ShapedLine` already has all three. `ui` stores a `TextLayout` in its state entity
and queries it during event handling, where there is no `Frame`. Add exactly these three;
if the text field turns out to need a fourth, that is a finding worth reporting, not a
method to guess at.

**`Text` re-shapes for a width that changes nothing.** The measure callback invalidates
on available width:

```go
if t.shapedLine == nil || t.lastAvailWidth != avail.Width {
    line, err := f.ShapeLine(t.content, runs)
```

`ShapeLine` does not take a width. It shapes one line with no wrapping, so the output for
the same content and style is identical at every available width, and the second call
throws away a correct answer to compute the same one again. Flexbox calls a leaf measure
several times per solve with different constraints, so this fires on most frames.

    BenchmarkTextMeasureSameWidth      6.13 ns      0 B/op    0 allocs/op
    BenchmarkTextMeasureVaryingWidth  37.52 µs  32075 B/op   92 allocs/op

Drop the width from the invalidation and keep it in `lastAvailWidth` only if something
else needs it. Invalidate on content and resolved text style, which are the inputs
`ShapeLine` actually reads.

`text` is separately making `ShapeLine` cheap on a repeat call, which is the other half of
the same number. Do both — a cheap call made too often and an expensive call that should
be cheap are different bugs, and fixing one hides the other.

Add the benchmark that would have caught it: measure with the width varying and assert
the shaped line is not rebuilt, not just that the answer is right.

**Style is resolved three times per frame.** `RequestLayout`, `Prepaint` and `Paint` each
build a 488-byte `style.Style` from the refinement. Resolve once in `RequestLayout`, keep
it on the element, and layer the hover, active and focus refinements onto a copy in
`Paint`.

Calibrate this correctly before spending the round on it: `BenchmarkStyleRefineNonEmpty`
is 51 ns and an element build is about 4 µs, so the three resolutions are roughly 4% of
element cost, not a wall. It is a tidy-up that happens to be free. The allocation is the
expensive part and the arena will deal with that separately.

## Coming under you, and one thing to leave alone

`window` is relaxing `PushClip` and `PopClip` to be legal in prepaint as well as paint,
so that hit regions can be intersected with the clip in force when they are registered.
Today a button scrolled out of a `ScrollView` is invisible and still clickable.

When that lands, `Div.Prepaint` pushes and pops around its children exactly as
`Div.Paint` already does, and the phase rules in the `Frame` doc comment change with it.
Wait for `window` to report; the relaxation has to be under you before you use it.

Do not add `PushLayer`, `PopLayer` or a deferred-paint method to `Frame`, even though
`docs/audit.md` says popups, menus and tooltips need them and it is true. Two things stop
it. `element` declares `Frame` and `window` implements it, so the implementer goes first —
that is the ordering rule in `AGENTS.md` and the one that has broken `main` before. And
neither of us should add it before there is a caller: `ui` gets a popup, and the method
arrives with it. An interface method with no caller is a guess, and we have had two.

## Done when

    go test ./element ./element/elementtest
    go test -tags facet_debug ./element

pass, including a test that builds a view, dispatches a click through the handler the way
`window` does, and asserts the entity's state changed. Break the listener and confirm
that test fails.

`elementtest.Frame` can drive it end to end without importing `platform` or `window`.

Report before touching `ui` or the examples. Migrating them is one landing with this, but
I want to see the seam before it propagates.
