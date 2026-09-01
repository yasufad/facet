# ui: blocked on two, and one thing of yours to fix first

Your capture report worked. It is assigned to `window` this round on exactly the argument
you made — one gap holding up four widgets, named with the slider and the scrollbar thumb
alongside it, rather than half of one. Drag selection comes back into scope for the text
field once it lands.

Two of your dependencies are still in flight. What follows is the order.

## Blocked, and on what

**`input`** has not landed. Its prompt is written and it is the reason the tree is red:
`ui` imports `platform` because `OnScrollWheel` cannot name a wheel event otherwise. Do
not work around it. When the aliases land, `scroll_view.go` drops its `platform` import
and `go test ./internal/layering` goes green.

**`element`** is adding `Listener`, `PhasedListener` and `TextLayout`. All three are yours
to consume and none exists yet. `TextLayout` is the caret mapping you asked for, decided
your way; the listeners are new since your last round and they change how every widget in
this package registers a handler.

The listeners matter more than the text field does. `docs/audit.md` has the reproduction:
a view cannot mutate its own state from a click, because `Render` runs inside an update
and every handler signature forces the caller to capture what that update leased. Your
`Button` has that signature. Its test passes because it sets a test-local `clicked` and
never touches an entity, which is the exact pattern `AGENTS.md` warns about.

So when `element` lands, `Button` migrates first and gets a test that mutates real entity
state through a real dispatch. Then the text field.

## One thing that is yours now

`ScrollView.Paint` writes entity state during the paint phase:

```go
s.state.Update(s.app, func(st *ScrollState, cx *app.Context[ScrollState]) {
    st.UpdateMetrics(bounds.Size.Height, contentBounds.Size.Height)
})
```

It works, and it works only because it does not notify. One `cx.Notify()` in there and
every frame schedules the next one for ever.

The rule I am setting: writing entity state during paint is allowed for recording what the
frame measured, and notifying from paint is not. Put that in your package doc rather than
leaving it as a property nobody wrote down, and add a test that would fail if someone
added the notify — assert the frame count after a paint, not just the recorded metrics.

If you would rather have a mechanism than a rule, say so and propose one. A `Frame` method
that records post-layout metrics without going through an entity is a reasonable shape and
I would consider it. I am not inventing it speculatively.

## Then the text field

Same milestone as before: typing, a caret, arrow keys, backspace and delete, click to place
the caret. Selection is now in scope rather than excluded, because capture is being fixed
in the same round — but only reach for it once `window` reports that capture lands, and
say plainly if it does not behave the way the widget needs.

The caret and the selection quad are both `Div`s, as decided last round. Nothing new is
needed below you for them.

Clipboard and caret blink stay out, unchanged from last round and for the same reasons.

## While you are blocked, forecast

You forecast six gaps before writing the scroll view, four of them accurately, and three
were decided rather than negotiated mid-implementation because of it. That was worth more
than the same six found one at a time.

Do it again, now, for the text field, and add a second one for a virtualised list.

The list is the widget that decides whether an editor is possible with this framework, and
I want your reading of what it needs before I take the decision behind it. `ScrollView`
lays out and paints its entire content every frame and clips the result, so a hundred
thousand lines is a hundred thousand elements. Building only what is visible needs an
element that learns its viewport before it builds its children, and `RequestLayout` has
already built the whole subtree by the time `Prepaint` hands you bounds.

I do not think that is solvable inside `ui`. Tell me what shape you would need from
`element` for it — what an element would have to be able to ask, and when. That is a
finding I want in your words before I write the contract, because you are the one who will
have to build against it.

## Done when

`go test ./internal/layering` is green with `ui` off `platform`.

`Button` uses the listener seam, with a test that changes entity state through a dispatch
and reads it back. Break the listener and confirm the test fails.

The text field meets the milestone above, with `elementtest.Frame` driving it.

The paint-phase rule is written down and tested.

Both forecasts are reported before you start the widget they describe.
