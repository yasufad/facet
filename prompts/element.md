# element: the four findings landed, two things to close

All four are in and I verified the one that mattered. Disabling the inherited style in
`Text.Paint` fails `TestTextInheritsParentStyleAndHover`, so the inheritance is real
rather than incidentally satisfied.

`element/elementtest` is the best outcome of the round. `ui/button_test.go` now
imports `colour`, `element`, `elementtest`, `geometry`, `input` and `style` and
nothing else — 180 lines of duplicate double gone, and the packages `ui` may never
name in production are gone with it. Anyone writing a Facet widget outside this
repository can now test it.

Two things.

## 1 — Five commits did not compile

The rule added to `AGENTS.md` two days ago says an interface method is implemented
before it is declared, because declaring it first stops the implementing package
compiling. `8359612` added `PushTextStyle`, `PopTextStyle` and `TextStyle` to `Frame`;
`09793db` implemented them on `*Window` five commits later. At `8359612`:

    window\frame.go:51:23: *Window does not implement element.Frame
                           (missing method PopTextStyle)

`main` was unbuildable through four intervening commits. The end state is correct and
I am not asking you to rewrite history, but this is the third time this class of break
has happened and the first since the rule was written down, so read that entry in
`AGENTS.md` before the next interface change: implement on `*Window` first, where the
methods satisfy nothing and break nothing, then declare.

The same applies to the next round, which adds `RequestFocus` to `Frame`.

## 2 — The button still cannot prove it changes its label colour

The whole feature exists because a button could not change its text colour on hover.
I broke the inheritance and `element`'s test failed while `ui`'s entire suite passed.

`ui` tests hover background, active background and focus border, and not the thing
this round was for. Add it: a button with a hover refinement that sets a text colour,
and an assertion that the emitted glyph sprites carry it.

That belongs to `ui`, and I have put it in its prompt. Worth knowing here because it
is the shape to watch for: a capability added for a named consumer, verified only by
the package that provides it. The provider's test proves the mechanism works; only the
consumer's test proves the thing anybody asked for works.

## Done when

`docs/packages.md` says the text style stack is on `Frame`, pushed and popped around
children, and that pseudo-state refinements are merged into it before children paint.
That is a `Frame` contract change and outlives this prompt.

Then retire this. The next round of `element` work is tab order, which
`docs/packages.md` already says belongs here because it needs tree order, and which
`window` will raise when pointer focus lands.
