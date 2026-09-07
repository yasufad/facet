# element: one of the eight should not have been deleted, and it is mine

The round landed and I verified it rather than reading the report. Seven setters are
gone. All five implementations have real consumers: `Visibility` at `div.go:949`,
`TextBackgroundColour` as a quad at `text.go:306`, `BoxShadow` split drop-before-quad
and inset-after at `div.go:960` and `:983`, `Underline` at `text.go:333` and
`Strikethrough` reusing the same primitive at a different baseline offset at `:375`.

I broke both first-producers. Removing the drop-shadow emission failed
`TestDivBoxShadowDropPaintsBeforeBackground`; removing the underline emission failed
three tests including
`TestTextUnderlineAndStrikethroughBothPaint`, which caught it as 1 decoration line
instead of 2 rather than as zero — the assertion knows the difference between the two
producers. `scene.Shadow` and `scene.Underline` have producers for the first time in
this project.

Splitting drop and inset around the background quad, with the reason and GPUI's
matching split named in the comment, is the part I would have got wrong.

## ScrollbarWidth stays. The instruction to delete it was wrong.

I grouped it with `AllowConcurrentScroll` and `RestrictScrollToAxis` and said all three
"describe a scrolling implementation that does not read them". Those two are unread.
`ScrollbarWidth` is not: it leaves this package through `style.toLayout` at
`style/style.go:347` and reaches `layout`'s `scrollGutter` at `leaf.go:201`, called from
`flexbox.go:363`, `leaf.go:94` and three sites in the flexbox solver. It reserves a real
gutter when overflow is scroll, and it does that today.

I checked `element/` for consumers. The consumer is not in `element/` — the property
leaves through the layout conversion, so no grep of this package could ever have found
it. That is the same mistake I made scoping `input`'s prompt: reading a package's own
files instead of following the value out of it. Third defect from my guidance in this
tree, and the second of that exact shape.

Nothing to do but not delete it. One thing is worth adding, though, precisely because
this was the only one of the thirteen that already worked and neither of us knew: if no
test runs `Div.ScrollbarWidth` through to a content box that is actually narrower, add
one. It is the property most likely to be deleted again by someone repeating my search.

## Where the pixel test lives

I said I would decide this. It goes in `internal/integration`, behind `facet_debug`,
against the real D3D11 renderer — that package sits outside the layer table and may
import anything, so it is the only place `element` and `render` can meet.

It is yours to write, because you know what the primitives are supposed to look like.

It waits on `render`. The adapter-absent skip does not exist at HEAD — there is no
sentinel anywhere under `render/` — and without it the test is red on every machine
without a real adapter, which is every hosted runner. `prompts/render.md` carries that
spec. Start when it lands.

What it should check is narrower than "the shadow appears". The insert-level tests
already prove the primitive is emitted, and `render`'s readback tests already prove the
shader draws a `scene.Shadow` correctly from a synthetic one. The join those two do not
cover is **interpretation**: whether the blur radius, the offsets and the corner radii
you put in the struct mean the same thing to the shader that they mean to you. So run
it at a scale factor other than 1, where a logical-versus-device-pixel mistake is
visible and at scale 1 it is not. One drop shadow and one underline is enough.

## Not needed

I asked you to report which properties landed so I could write `docs/packages.md`. I
verified it directly instead — no report needed, and I will write the entry.

## Done when

`ScrollbarWidth` is still there, and reaching its reserved gutter from a `Div` is
covered by a test.

The integration pixel test exists once `render` lands the skip, at a scale factor other
than 1.
