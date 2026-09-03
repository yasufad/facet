> **Read before starting.** You are picking up one package in a framework several agents
> build in parallel. `AGENTS.md` loads automatically and is the standard you are held to —
> read it, and read your package's entry in `docs/packages.md`, before you touch code. This
> file is only the work outstanding; it does not restate what those two say.
>
> The four things that cause the most damage here, in order:
>
> 1. **Commit one file per commit, by path:** `git commit -m "..." -- path/to/file.go`.
>    Never `git add -A`, never `git add .`, never `git commit -a`. The git index is shared
>    with other agents — staging then committing lets another agent's file land in your
>    commit. Three agents have had that happen. Check `git show --name-only` after every
>    commit; it should name exactly the file its message describes. The exception is a
>    change that is not true in pieces (a rename, or code plus the comment stating its
>    guarantee) — say so in the message when you use it.
> 2. **Stay in your package.** If your change needs another package's exported API to move,
>    stop and report it. Do not edit across the boundary unless this file explicitly says
>    you may.
> 3. **Verify at HEAD, not in the working tree.** Other agents have uncommitted files here,
>    so `go build ./...` in this checkout reports their half-finished work as yours. Use
>    `git worktree add --detach <scratch>/check HEAD`, run the commands there, remove it.
> 4. **Break your fix and confirm the test fails.** A test that cannot fail is not a test,
>    and every package here has passed its own tests while getting something wrong. Watch
>    for the break silently not applying — check `git diff --stat` before believing a green
>    run, and for an unused variable turning your break into a build error rather than a
>    failure.
>
> Report back when done rather than expanding scope. Do not edit `docs/`, `README.md`,
> `AGENTS.md` or `prompts/` — those belong to the reviewing agent, who writes up what your
> package guarantees once it lands.

# ui: the last package between this framework and a working example

Everything you were waiting on has landed. Nothing here is blocked, and two of the items
below are the only red things left in the repository.

## 1. `main` does not build, and it is your file

```
ui\button.go:163:9: b.div.Opacity undefined (type *element.Div has no field or method Opacity)
```

`element` deleted the `Opacity` setter because it had no consumer — it set a property
nothing read, so `Button.Disabled` has rendered identically to an enabled button for as
long as it has existed. The deletion was right and I ordered it.

The sequencing was mine and it was wrong. I told `element` to delete it and told them
explicitly not to fix `ui`, which guarantees the broken `main` we have. `AGENTS.md` covers
renames and interface methods; a deletion is the same shape — the caller stops calling
before the provider removes — and I inverted it two lines below where the rule is written.

Give `Disabled` a real affordance. A dimmer background and a dimmer foreground through the
refinement you already build. That is what the opacity was standing in for and it is the
version that will actually be visible.

Land it on its own, first.

## 2. One import, and the last failing test goes green

```go
if event.Delta.Unit == platform.ScrollPixels {
```

`input` now has `ScrollUnit`, `ScrollPixels` and `ScrollLines`. Swap them, drop the
`platform` import from `ui/scroll_view.go`, and `internal/layering` passes for the first
time since before this work started.

That gap was also mine — I scoped `input`'s first prompt from its own handler signatures
rather than from your call site, so it aliased the four event types and missed the scroll
unit. It cost you a round of being blocked.

## What is available to you now

**`element.Listener` and `element.PhasedListener`.** A handler reaches its view's state:

```go
element.Listener(cx, func(v *T, e element.ClickEvent, cx *app.Context[T]) bool { ... })
```

`PhasedListener`'s return assigns straight to `input.KeyEventHandler` and its siblings.

**`element.TextLayout`** — `XForIndex`, `IndexForX`, `ClosestIndexForX`, queryable with no
`Frame` in scope, which is what lets caret arithmetic run during event handling. Half
leading landed with it, so a taller line height centres its glyphs and the caret comes out
of the same numbers.

**Five style properties that now do something**: `Visibility`, `TextBackgroundColour`,
`Underline`, `Strikethrough`, `BoxShadow`. The last three are the first producers
`scene.Shadow` and `scene.Underline` have ever had. `BoxShadow` is worth knowing about for
`Button` — a real elevation is available where it was not before.

**Pointer capture** in `window`: press inside an element and moves and the release route to
it wherever the pointer goes, with `IsActive` tracking containment separately. Drag
selection is in scope.

**Hit regions are clipped**, so a widget scrolled out of a `ScrollView` is no longer
clickable.

## 3. Button, and the example

`ui.Button` still takes `func(event element.ClickEvent) bool`, and its test still sets a
test-local `clicked` without touching an entity — the pattern `AGENTS.md` warns about, and
the reason the framework's central defect survived as long as it did.

Migrate it to the listener seam, with a test that changes real entity state through a real
dispatch and reads it back. Break the listener and confirm it fails.

Then `examples/button`. It is written the way the framework intends and has never run. It
is what `docs/audit.md` opens with.

## 4. The integration test

One test in `internal/integration`, which exists for this and nothing else: `window` may
not import `ui` and `ui` may not import `window`, so no package can test a widget clicked
in a real window.

`window` has already written the shape you need —
`TestClickMutatesEntityStateAndRendersNextFrame` in `window/window_test.go:1395` builds a
view with entity state, drives a frame, dispatches a synthetic down and up, and asserts
both the entity changed and the next frame reflects it. Yours is that with a `ui.Button` in
the middle. Read it rather than starting from nothing.

Keep it to one test. The value is that the path executes at all.

## 5. The text field

Typing, a caret, arrow keys, backspace and delete, click to place the caret. Selection is
in scope now that capture has landed.

The caret and the selection quad are both `Div`s, as decided. Nothing new is needed below
you. Clipboard and caret blink stay out.

## 6. The paint-phase rule, still yours

`ScrollView.Paint` writes entity state during paint at two call sites. It works only
because it does not notify; one `cx.Notify()` there and every frame schedules the next for
ever.

The rule stands: writing entity state during paint is allowed for recording what the frame
measured, notifying from paint is not. Put it in the package doc and add a test that fails
if someone adds the notify — assert the frame count after a paint, not just the recorded
metrics.

If you would rather have a mechanism than a rule, propose one.

## 7. The forecast, and it now has a deadline

I have asked twice for your reading of what a virtualised list needs from `element`, and I
have deferred the element identity decision four rounds waiting for it.

I am taking that decision when you next report, with or without it. `docs/audit.md` now
carries GPUI's mechanism read from source — a path from the root, per-element state carried
between the two frames `window` already keeps, reachable in prepaint — so it is no longer
waiting on information I lack.

If you want the shape to fit what you will build against, this is the round to say so.
`ScrollView` builds, lays out and paints its whole content every frame and clips the
result; a hundred thousand lines is a hundred thousand elements, and `RequestLayout` has
already built the subtree by the time `Prepaint` hands you bounds. What would an element
need to be able to ask, and when?

## Done when

    go build ./...
    go test ./internal/layering
    go test ./ui
    go test ./internal/integration

all pass, and `examples/button` increments a counter when clicked.
