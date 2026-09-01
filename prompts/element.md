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

**Style is resolved three times per frame.** `RequestLayout`, `Prepaint` and `Paint` each
build a 488-byte `style.Style` from the refinement. Resolve once in `RequestLayout`, keep
it on the element, and layer the hover, active and focus refinements onto a copy in
`Paint`.

Calibrate this correctly before spending the round on it: `BenchmarkStyleRefineNonEmpty`
is 51 ns and an element build is about 4 µs, so the three resolutions are roughly 4% of
element cost, not a wall. It is a tidy-up that happens to be free. The allocation is the
expensive part and the arena will deal with that separately.

## Not yet

Do not add `PushLayer`, `PopLayer` or a deferred-paint method to `Frame` this round, even
though `docs/audit.md` says popups and menus need them and it is true.

`element` declares `Frame` and `window` implements it. Declaring a method before the
implementer has it stops `window` compiling until it catches up, which is the ordering
rule in `AGENTS.md` and the one that has broken `main` before. `window` adds the methods
first, on its own prompt, and then you declare them. That round is next, not this one.

## Done when

    go test ./element ./element/elementtest
    go test -tags facet_debug ./element

pass, including a test that builds a view, dispatches a click through the handler the way
`window` does, and asserts the entity's state changed. Break the listener and confirm
that test fails.

`elementtest.Frame` can drive it end to end without importing `platform` or `window`.

Report before touching `ui` or the examples. Migrating them is one landing with this, but
I want to see the seam before it propagates.
