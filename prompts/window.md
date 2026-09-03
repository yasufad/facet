# window: SetRootFn looks reactive and is not

All three items landed and I verified them independently — that report reached me twice,
and the answer both times is that it holds up.

I broke `element.Listener` so it stops mutating and
`TestClickMutatesEntityStateAndRendersNextFrame` failed. It asserts both halves, entity
state *and* the next frame's quad, which is what makes it the test `docs/audit.md` opens
by asking for. It has existed for about a day, having been named as necessary since before
any of this work started.

The slice change is correct on the two things it invites. Both setters grow before
writing, the growth appends an explicit zero so a reused backing array cannot serve last
frame's bounds to an index this frame skipped, and `clear` on `measureCallbacks` before
the reset drops the closures rather than keeping them alive. I checked all three.

Your `DispatchText` answer is accepted and in `docs/packages.md`. The argument that
decided it was your second, not the consistency one: text and composition are the
platform's input-method state machine addressed to the active client, and a container
swallowing a composition update mid-stream desynchronises us from it.

## What putting OnIMEComposition on *Window revealed

That placement was right, and better than you claimed for it — it is the ordering rule
working, the implementer landing the method where it satisfies nothing and breaks nothing.

It also surfaced a gap neither of us had seen. `element` may not import `platform`, and
`input` does not alias `IMECompositionEvent`. So when `element` declares that method on
`Frame` for `ui`'s text field, it cannot name the parameter type — the same wall `ui` hit
twice. `input` is reopened to alias the whole event vocabulary rather than a third
one-off; when it lands, your signature changes from `platform.IMECompositionEvent` to
`input.IMECompositionEvent`, which is an alias and therefore not a change at all.

## 1. SetRootFn attaches nothing

```go
func (f fnView) Observe(_ *app.App, _ func(*app.App) bool) app.Subscription {
	return app.Subscription{}
}
```

`staticView` doing this is honest — it holds a fixed element and has nothing to observe.
`fnView` is not. `SetRootFn(func() element.Element)` reads as the reactive root, the
closure can read entity state, and nothing will ever repaint when that state changes. It
looks like it works because every input event sets `dirty` directly, so a window that is
being clicked repaints and a window that is not does not.

Neither has a doc comment, so nothing warns anyone.

Decide which of the two it is and make the code say so. Either `fnView` observes what its
closure read — which it cannot know, so this probably means the API is wrong — or
`SetRoot` and `SetRootFn` are documented as static roots for content that does not react,
with `SetRootView` named as the one that does.

I lean to the second: a closure with no declared dependencies cannot be made reactive
without something like a tracking scope, and inventing one here would be a large decision
taken sideways. But say what you find. If `examples/quad` depends on the current
behaviour, that is worth knowing before either of us changes it.

## 2. Still not yet, and now with a date

**The layout tree is rebuilt every frame.** Unchanged, and still the biggest single
performance defect in the tree. It needs node identity surviving a frame, which needs the
element identity decision, which is mine and which I have now deferred four rounds while
saying "soon" each time.

So: I take it when `ui` next reports, with or without the virtualised-list forecast I have
asked them for twice. `docs/audit.md` has GPUI's mechanism read from source — a path from
the root, state carried between the two frames you already keep, reachable in prepaint —
so the decision is no longer waiting on information I lack. It is waiting on one more
consumer's opinion, and it will not wait past that.

**Layers.** Unchanged. They arrive with their first caller in `ui`.

## Done when

`SetRoot` and `SetRootFn` either observe or say plainly that they do not, with a test that
fails if the documented behaviour changes.
