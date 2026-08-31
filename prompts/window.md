# window: it draws. One thing to document, then it is done.

Facet renders. I ran the readback test and then disabled `el.Paint` to check it could
fail:

    inside pixel (75, 75): got {0 0 0 0}, want {1 0 1 1} (magenta)
    outside pixel (270, 270): got {0 0 0 0}, want {0 0 1 1} (blue)

A real platform window, a real Direct3D 11 device, an element tree through seven
frame steps, and the right colours in the back buffer. Eleven packages that each
worked are now one thing that works, and this is the first evidence of it.

The reactivity is connected too. Notifying the root view entity marks the window
dirty and schedules exactly one frame through `platform.Dispatch`, with no manual
`Draw` in the test.

## The one gap: reactivity stops at the root view

A view that reads a second entity does not repaint when that entity notifies. I
probed it with a root view holding `app.Entity[childState]` and reading it in
`Render`:

    notifying an entity the view reads left the window clean

`SetRootView` observes the root view's own entity and nothing else, and `app` has no
record of which entities a render read.

I am not asking you to implement read tracking. Explicit observation is a legitimate
design and close to what GPUI does for non-view entities: the view observes what it
depends on and notifies itself, which is one line at the point where the dependency
is already obvious. Read tracking buys convenience at the cost of a per-frame
accessed-set and subscription churn, and that is a decision to take deliberately when
`ui` shows what real applications need.

What is not acceptable is leaving it undocumented. `docs/architecture.md` said "a
view repaints when the entities it holds change", which is not what happens; I have
corrected it to say the view repaints when its own entity notifies, and that other
entities are observed explicitly.

Put the same thing in `window/doc.go`, with the pattern spelled out, because this is
the first question anyone writing a Facet application will hit.

## Then retire it

Two steps. `docs/packages.md` needs what this package now guarantees:

- The seven steps, and that step 5 resolves the pointer against `next` while event
  routing resolves against `rendered`. Two hit tests, different frames, easy to
  conflate. This is the thing most worth writing down.
- Phase enforcement: which `Frame` methods are legal in which phase, and that
  violations panic.
- Frame scheduling: `frameScheduled` collapses a burst into one frame; the root view
  entity is observed; other entities are explicit. Idle costs no draws and no
  presents.
- Resize and scale-change ordering, and what a scale change drops.
- Why `w.mu` exists, since everything above `app` is otherwise single-goroutine.
- That correctness here is established by reading pixels back, the same rule
  `render` holds, and that `window_debug_test.go` is where that happens.

Then set the README row and delete this file.

## What you changed in other packages

`app.AnyEntity` and `element.AnyView.Observe` were both needed and both are the right
shape. Adding them was correct rather than working around them, and it is the outcome
the rule is meant to produce. For the record it should have been raised before
landing, since both packages were retired, but the calls themselves were right and I
would not undo either.

## Worth carrying

The pixel test is the most valuable test in the repository. Every layer had proof it
worked alone and none that they worked together, and "it compiles" had already been
wrong three times in this project. Keep that test green above everything else in this
package; if it ever fails, nothing else here matters until it passes.
