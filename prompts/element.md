# element: elementtest still blocks ui, and one comment claims more than it can

Three good landings. `TextLayout` is the shape asked for and the zero-value guard is real —
I stripped it and the test panicked. The half-leading test comparing sprite position at
natural against taller line height is the right construction, because it does not depend
on which font backs the system, and this repository has had font-dependent assertions
before. The `Div` resolve-once is correct including the part that is easy to get wrong:
`style.Style` has exactly one slice field and `Refine` assigns it wholesale rather than
appending, so the copy your comment relies on really is a copy. I checked.

## 1. elementtest, still — and this is the blocker

`go test ./ui` still panics at HEAD:

    panic: elementtest: PushClip called outside paint phase
        element/elementtest/frame.go:559

I pushed this into your prompt after you had already started, so you plausibly never saw
it. It is unchanged and it is the highest-value thing in the tree right now, because `ui`
cannot run a single widget test until it lands, and `ui` is the last package between this
framework and `examples/button` working.

`Div.Prepaint` pushes a clip. You taught `element/fake_frame_test.go` the dual-stack
semantics and left `element/elementtest/frame.go` — the exported double — enforcing the
old paint-only rule. `PushClip` and `PopClip` both, lines 559 and 581.

Give it the two stacks the real `Frame` has, and add the test that would have caught it:
prepaint a clipping `Div` through `elementtest.Frame` and assert the registered hit region
is clipped. Break the phase rule and confirm it fails.

`go test ./element ./element/elementtest` passing is not evidence here — nothing inside
`elementtest` drives a clipping `Div` through prepaint, which is why the break landed in
`ui` instead. Run `go test ./ui` before you report.

## 2. The width comment is right about content and wrong about style

Dropping `lastAvailWidth` is correct and the test is good — five widths, one `ShapeLine`
call, and it fails with five when the old check comes back.

The justification you wrote into the code is not:

    // Content and style are fixed for the lifetime of this element (they
    // can only change before RequestLayout, which runs once), so shaping
    // once and keeping it for every later call is the whole cache.

Content, yes. Style, no. `Text.Paint` re-reads `f.TextStyle()`, and that value carries the
pseudo-state refinements a container merges in before children paint — `docs/packages.md`
says so explicitly, and hover is resolved at step 5, between prepaint and paint. So the
style the element paints under is not always the style it shaped under.

`Paint` then shapes only `if t.shapedLine == nil`, which is never true by that point, so it
computes `textStyle` and drops it on the floor. Hover a container that changes font size,
family, weight or features, and the glyphs keep the layout-time shaping. Colour still
updates, because that is per-sprite — so it looks half-working, which is worse than
looking broken.

You inherited that guard rather than writing it; the commit only touched the measure path.
What is new is the comment asserting it is safe, and a comment that states a guarantee is
part of it.

Two things to settle, and I want your reading before I decide the second:

Fix the comment now, whatever else happens. Say that content is fixed and style is not,
and name the paint-time merge as the reason.

Then say what the invalidation should be. Re-shaping in paint when the resolved text style
differs from the one shaped under is the obvious answer and it costs a comparison plus, on
a real hover, one `ShapeLine` — which `text` has made roughly 190 times cheaper since this
was written. The alternative is to declare that pseudo-state may not change font metrics,
only colour, and enforce it. That is a narrower framework and a defensible one. It is a
contract question rather than a bug fix, so tell me which you would take and why.

## 3. What is left

Nothing else from the last round. `PushLayer`, `PopLayer` and deferred paint stay with
`window`, and `ui.Button` and `examples/button` stay with `ui` — you were right not to
touch them, and the confusion was mine.

## Done when

    go test ./ui

runs. That is the measure this round, not `go test ./element`.

Plus the `elementtest` prepaint-clip test failing when the phase rule is reverted, the
width comment telling the truth, and your answer on the style-invalidation question.
