# Assignment: ui

Build `ui`: the widget library. Buttons, labels, lists, text fields, scroll views.

Everything below is finished and runs. A program can open a window and draw a styled
element tree containing text, on Windows through Direct3D 11. What it cannot do is
anything a person would recognise as an application, because there is nothing to click.

## Read first

1. `AGENTS.md` — conventions, commit style, GB English
2. `docs/packages.md` — the `ui`, `element`, `style` and `input` entries
3. `docs/architecture.md` — "What is deferred", because two of the entries there are
   things you are likely to run into
4. `element`'s public API. Read `div.go`, `text.go` and `interactivity.go` properly;
   they are what you are building from
5. `_upstream/gpui/crates/gpui/src/elements/` for how GPUI shapes equivalents

## Stop after one widget

A button. Not a button system: one `Button` with a label, a click handler, and hover,
active and focus styling. Built entirely from `element` and `style`, with no new
capability added anywhere below.

Then stop and say so.

That is not caution, it is the point of the milestone. `ui`'s invariant is that a
widget needs nothing the framework does not already expose, and one real widget is
what tests that claim. Everything you have to reach for that is missing is the actual
finding, and it is worth more than the widget.

## What this package is for, beyond widgets

`ui` is the first consumer of the whole stack that is not a test. Every seam below has
been verified in isolation and assembled once, by an example that draws a box and a
letter. A button exercises styling, layout, text, hit testing, dispatch, focus and
pseudo-state styling together, which nothing yet has.

So when something is awkward, say so rather than working around it. A widget library
that has to reach past `element` is telling you the framework is wrong, and that is
the report I want.

## Invariants

Built entirely from the public API of `element`, `style` and `input`. If a widget
needs something those do not expose, the gap is in the framework and gets fixed there
— by the package that owns it, not from here.

No widget registry. Adding a widget adds a file and touches nothing else. No central
`types.go`, no enum-and-switch dispatch.

`ui` imports `geometry`, `colour`, `style`, `element`, `input` and `app`, and nothing
else of ours. Notably not `window`, `render`, `scene` or `platform`.

## Things that will come up

**Text sizing is a single line.** `element.Text` shapes one line with one style run.
A button label is fine. A wrapping paragraph is not implemented, and that is listed
under "What is deferred" rather than something for you to add here.

**Hover, active and focus** are refinement closures on the element, evaluated during
paint against this frame's hit regions. A hover style that changes *layout* rather
than painting will lag a frame; prefer styles that change colour, border and shadow.

**State that survives a frame belongs in an entity.** A button that tracks whether it
is pressed does not keep that on the element, because the element is rebuilt every
frame. This is the first place that rule will bite in ordinary code rather than in
framework code, so get it right once and the rest of the library follows.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

The layering test passes.

A test drives the button through the three phases against a fake frame and asserts
what reached the scene: the background, the border, the label's glyphs. A test shows
the click handler firing from a dispatched pointer event, and hover styling changing
the emitted background.

An example puts a working button on screen and prints something when it is clicked.

A written list of everything you had to work around, or would have, with the package
that owns each gap. That list is the point of this milestone.

Conventional commits, one file per commit, staged by path.

## Habits from earlier rounds

Stay in your package. Where you need something another package does not expose, say
so and stop; every cross-package finding raised this way has turned out to be real
and has been fixed properly at its source.

When a struct of options exists, at least one test passes it empty.

A test that needs to write unexported state to pass is telling you the design cannot
happen in production, not that the test needs a back door.

Compiling is not evidence. A widget that lays out correctly and paints nothing errors
nowhere.
