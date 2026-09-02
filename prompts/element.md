# element: TextLayout is the last thing blocking the text field

All three gaps closed and I verified each by breaking it. The prepaint clip test fails
with the unclipped bounds named in the message; removing `defer entity.Release()` from
either adapter now fails both balance tests, which is exactly the hole I found last round
and it is shut. The `Frame` doc comment saying the two clip stacks are independent and a
prepaint push does not touch the scene is the sentence someone will need, and it is there.

Bundling `div.go`, `doc.go`, `frame.go` and `fake_frame_test.go` into one commit was
right. The fake had to match or nothing built, and the doc comments state the guarantee
that changed — that is the not-true-in-pieces exception doing its job rather than being
stretched.

## First, my error, because it is your question

You asked whether to take `ui.Button` and `examples/button`. You should not, and the
reason is that I told you to. My last prompt said "take them now"; `prompts/ui.md` assigns
the same migration to whoever holds `ui`. Two prompts claiming one file is the collision I
have spent three rounds trying to prevent, and I wrote it.

**`ui` owns `ui/button.go` and `examples/button`.** Do not touch either.

What you own is the thing that lets `ui` do it in one landing instead of two.

## 1. TextLayout

`ui` has been waiting on this for three rounds and it is now the only thing between the
text field and a caret. Nothing below you is missing; `text.ShapedLine` has all three
already.

```go
type TextLayout struct{ ... }              // element's type, wrapping the shaped line
func (t *Text) Layout() TextLayout
func (l TextLayout) XForIndex(byteIndex int) geometry.Pixels
func (l TextLayout) IndexForX(x geometry.Pixels) (int, bool)
func (l TextLayout) ClosestIndexForX(x geometry.Pixels) int
```

`ui` stores a `TextLayout` in its state entity and queries it during event handling, where
there is no `Frame`. That is the whole point of wrapping it: the caret arithmetic has to
work outside the frame, and `ui` must not learn `text`.

Add exactly these three. If the text field needs a fourth, that is a finding to report,
not a method to guess at.

One thing to get right while you are here, because it is the same arithmetic twice.
`element` puts the baseline at `bounds.Origin.Y + ascent`, so when the line box is taller
than ascent plus descent the whole difference lands below the glyphs. `DefaultTextStyle`
ships `FontSize: 16` with `LineHeight: 20`, so the default configuration already has
leading it puts on one side, and an editor's 1.5 line height pins glyphs to the top of the
box. The rule is CSS's: half the difference above the ascent. Caret height and position
come out of the same numbers, so getting the leading wrong once means getting the caret
wrong twice.

## 2. Text re-shapes for a width that changes nothing

```go
if t.shapedLine == nil || t.lastAvailWidth != avail.Width {
```

`ShapeLine` takes no width and wraps nothing, so its output is identical at every
available width, and flexbox calls a leaf measure several times per solve with different
constraints. Invalidate on content and resolved text style — the inputs `ShapeLine`
actually reads.

`text` has landed a line cache that made the repeat call roughly 190 times cheaper, so the
absolute cost has collapsed. Do it anyway. The call is still wrong, and a cache that makes
a mistake affordable is not the same as not making it.

Add the benchmark that would have caught it: vary the width and assert the shaped line is
not rebuilt, not merely that the answer is right.

## 3. Style resolved three times per frame

`RequestLayout`, `Prepaint` and `Paint` each build a 488-byte `style.Style` from the
refinement. Resolve once in `RequestLayout`, carry it on the element, layer the
pseudo-state refinements onto a copy in `Paint`.

Calibrated: `BenchmarkStyleRefineNonEmpty` is 51 ns against an element build of about
4 µs, so this is roughly 4% and a tidy-up rather than a wall. It is free and worth having;
it is not worth restructuring anything to get.

## Not yours, still

No `PushLayer`, `PopLayer` or deferred paint on `Frame`. `window` implements `Frame`, so
the implementer goes first, and neither of us adds it before `ui` has a popup to call it.
An interface method with no caller is a guess and we have had two.

## Done when

    go test ./element ./element/elementtest
    go test -tags facet_debug ./element

pass, with `TextLayout` driven through `elementtest.Frame`, a benchmark that fails if the
width invalidation comes back, and half-leading applied so a 1.5 line height centres its
glyphs.

Report when `TextLayout` lands, ahead of the rest if it is ready first — `ui` is blocked
on that method set and nothing else.
