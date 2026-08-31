# ui: the text field, with the forecast answered

The clamping is right and `TestScrollViewClampingAgainstContent` knows its answer —
250px from a 400px column in a 150px viewport, not "some positive number". `FlexShrink(0)`
on the content is the kind of detail that would have been a mystery bug later.

The forecast is the most useful thing you have produced. Answers below, in your
lettering. Two of your six are not gaps, one is a gap you did not forecast, and three
are out of scope for this milestone by my decision rather than yours.

## A — Event vocabulary: in flight

`input` is adding the aliases now and has not landed yet; its signatures still name
`platform`. You are blocked on it for text and key handling. Do not start those parts
until it is in.

## B — Caret mapping: `element` exposes it, `ui` does not learn `text`

You were right that `text.ShapedLine` already has what is needed. I checked:
`XForIndex`, `IndexForX`, `ClosestIndexForX`, `Ascent`, `Descent` are all there.

The decision is your option 2, not option 1. `ui` does not gain `text`. `element.Text`
stops being a black box and returns an opaque handle that `element` owns:

    type TextLayout struct{ ... }          // wraps the shaped line, element's type
    func (t *Text) Layout() TextLayout
    func (l TextLayout) XForIndex(byteIndex int) geometry.Pixels
    func (l TextLayout) IndexForX(x geometry.Pixels) (int, bool)
    func (l TextLayout) ClosestIndexForX(x geometry.Pixels) int

`ui` stores a `element.TextLayout` in its state entity and queries it during event
handling, when there is no `Frame` — the same shape as the scroll metrics you just
implemented, and for the same reason.

That is a change to `element`, so raise it there rather than doing it here. Say
exactly which methods you need and no more; an interface method with no caller is a
guess, and we have had two.

## C — Caret and selection quads: not a gap

Both are `Div`s. A caret is an absolutely positioned `Div` two pixels wide at
`XForIndex`, the height of the line. A selection is another one behind it spanning
`XForIndex(start)` to `XForIndex(end)`. `style` has `Position` and `Inset`, and paint
order is child order, so "selection behind, glyphs, caret in front" is the order you
add the children.

Nothing new is needed below you for this. If that turns out to be wrong, that is a
finding worth having.

## D, E — Clipboard and caret blink: out of scope, deliberately

Both are real gaps and neither is in this milestone.

**Clipboard.** Nothing above `platform` can reach `platform.Clipboard`, and the access
happens during event handling where there is no `Frame`. The shape I expect is a small
interface declared in `app` — which imports nothing, so it can declare one — installed
by `window` at construction, reached as `cx.Clipboard()`. I am not deciding that until
something needs it, because a service locator added speculatively grows.

**Blink.** A static caret is fine for a first text field. Note for when it comes up:
until precise invalidation exists, a 500ms blink rebuilds the entire element tree
twice a second, which is a reason to want per-view invalidation rather than a reason
to skip blinking.

## F — Two corrections, and the gap you missed

`style.CursorText` is the I-beam. There is no `CursorIBeam`; use the one that exists.

**There is no pointer capture anywhere in the framework.** I checked `element`,
`window` and `input`: the only "capture" is the dispatch phase, which is a different
thing. Press inside your field, drag outside it, and the move events go to whatever is
under the pointer instead of to you. GPUI keeps a `captured_hitbox` on the window for
exactly this.

So drag selection cannot work today, and neither can a slider, a resize handle, or a
scrollbar thumb. That is the most valuable thing in this round and you did not forecast
it, which is fair — it is invisible until you drag.

## The milestone

Typing, a caret, arrow keys, backspace and delete, and click to place the caret. No
selection, no clipboard, no blink, no multi-line.

Selection is excluded because it needs pointer capture, not because it is hard. Report
that as the blocker so capture gets assigned on its own merits, with the slider and the
scrollbar thumb named alongside it — one gap holding up four widgets is a different
priority from one holding up half of one.

Then stop and report as usual.

## Worth carrying

You forecast six gaps, four of them accurately, before writing a line. That is worth
more than the same six discovered one at a time over a week, and it let three of them
be decided rather than negotiated mid-implementation. Do this again for the widget
after.
