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

# ui: everything you were waiting on has landed

Your last prompt said you were blocked on two packages. Both have reported since, and this
is the round where the framework either works end to end or does not.

Read the two blockers below before starting — one of them still stops your tests running,
and it is not yours to fix.

## Still blocked, briefly, on two things not yours

`go test ./ui` panics at HEAD:

    panic: elementtest: PushClip called outside paint phase

`element` taught `Div.Prepaint` to push a clip and left the exported double enforcing the
old paint-only rule. It is assigned and it is that agent's first item this round. You
cannot run a widget test until it lands.

`internal/layering` is still red because `ui/scroll_view.go` reads
`event.Delta.Unit == platform.ScrollPixels`. `input` aliased the four event types but not
the scroll unit vocabulary, which is a gap I created by scoping that prompt from its
handler signatures rather than from your call site. It is reopened and assigned. When
`input.ScrollUnit`, `input.ScrollPixels` and `input.ScrollLines` exist, drop the
`platform` import — that one edit turns the layering test green for the first time.

Start on the rest while you wait; neither blocks the Button work.

## What is now available to you

**`element.Listener` and `element.PhasedListener`.** A handler can reach its view's state.
They close over a weak handle and re-enter `app.UpdateEntity` at dispatch time, so the
callback gets live state and a live `Context`:

```go
element.Listener(cx, func(v *T, e element.ClickEvent, cx *app.Context[T]) bool { ... })
```

`PhasedListener`'s return assigns straight to `input.KeyEventHandler` and its siblings, so
no signature on `Div` changed.

**`element.TextLayout`**, with `XForIndex`, `IndexForX` and `ClosestIndexForX`, queryable
with no `Frame` in scope. This is the caret mapping you asked for, decided your way. Half
leading landed with it, so a taller line height now centres its glyphs and the caret
arithmetic comes out of the same numbers.

**Pointer capture**, in `window`. Press inside an element and the moves and the release
route to it wherever the pointer goes. `IsActive` tracks containment separately, so a
button un-presses when you drag off it. Drag selection is in scope for the text field.

**Hit regions are clipped.** A widget scrolled out of a `ScrollView` is no longer
clickable.

## 1. Button, and the example

`ui.Button` still takes `func(event element.ClickEvent) bool` and its test still sets a
test-local `clicked` and never touches an entity — the exact pattern `AGENTS.md` warns
about, and the reason the defect survived so long.

Migrate it to the listener seam. The test that replaces the current one changes real
entity state through a real dispatch and reads it back. Break the listener and confirm it
fails.

Then `examples/button`. It is written the way the framework intends and has never run;
it is the thing `docs/audit.md` opens with. Make it work.

**Both are yours.** I told `element` to take them last round, which was wrong — your
prompt already assigned them and two prompts claiming one file is what the ordering rules
exist to prevent. That was mine.

## 2. The integration test

One test in `internal/integration`, which exists for this and nothing else: `window` may
not import `ui` and `ui` may not import `window`, so no package can test a widget clicked
in a real window. `docs/packages.md` records why the directory is there.

Build a view with state, put a `ui.Button` in it, drive a frame, dispatch a synthetic
pointer down and up through `window.DispatchEvent`, and assert the entity changed and the
next frame shows it.

Keep it to one test. The value is that the path executes at all — it never has.

## 3. The text field

Same milestone as before: typing, a caret, arrow keys, backspace and delete, click to
place the caret. Selection is in scope now that capture has landed.

The caret and the selection quad are both `Div`s, as decided. Nothing new is needed below
you.

Clipboard and caret blink stay out, unchanged and for the same reasons.

## 4. Still yours from last round

`ScrollView.Paint` still writes entity state during paint, at two call sites:

```go
s.state.Update(s.app, func(st *ScrollState, cx *app.Context[ScrollState]) {
```

It works only because it does not notify. One `cx.Notify()` and every frame schedules the
next for ever.

The rule stands: writing entity state during paint is allowed for recording what the frame
measured; notifying from paint is not. Put it in the package doc and add a test that fails
if someone adds the notify — assert the frame count after a paint, not just the recorded
metrics.

If you would rather have a mechanism than a rule, propose one. A `Frame` method that
records post-layout metrics without going through an entity is a reasonable shape and I
would consider it. I am not inventing it speculatively.

## 5. The forecast I still want

You forecast six gaps before the scroll view and four were right, which is why three of
them were decided rather than negotiated mid-implementation. I asked for the same for a
virtualised list and have not had it.

It is the widget that decides whether an editor is possible here. `ScrollView` builds,
lays out and paints its whole content every frame and clips the result, so a hundred
thousand lines is a hundred thousand elements. Building only what is visible needs an
element that learns its viewport before it builds its children, and `RequestLayout` has
already built the subtree by the time `Prepaint` hands you bounds.

I do not think that is solvable inside `ui`. GPUI keys per-element state on a path from
the root and carries it between the two frames it already keeps, reachable in prepaint —
`docs/audit.md` has the mechanism now, read rather than assumed. Before I write that
contract I want your reading of what you would need from `element`: what an element must
be able to ask, and when.

Your words, before the decision, because you are the one who builds against it.

## Done when

    go test ./internal/layering
    go test ./ui
    go test ./internal/integration

all pass, `examples/button` runs and increments, and the forecast is reported.
