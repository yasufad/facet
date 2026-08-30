# Assignment: element

Build `element`: the `Element` interface, the three-phase lifecycle, the element
tree, the `Frame` interface that `window` implements, and `div`.

`window` and `ui` both wait on this, and both are blocked until it lands. It is also
where the fluent style builder lives — see "The builder is yours" below, because that
is a decision already taken and it is not the obvious one.

## Read first

1. `AGENTS.md` — conventions, commit style, GB English
2. `docs/architecture.md` — "The frame", "State and reactivity"
3. `docs/packages.md` — the `element`, `style`, `layout`, `app` and `window` entries
4. `prompts/window.md` — what the package above expects of `Frame`
5. `_upstream/gpui/crates/gpui/src/element.rs` — 790 lines, read all of it
6. `_upstream/gpui/crates/gpui/src/elements/div.rs` — 5,200 lines, read the `Div`
   struct and its three phase methods; skip `Interactivity` for now

Sync the checkouts with `go run ./tools/upstream` if `_upstream/` is not there.

## Stop after one element through three phases

Not `div` in full, and certainly not interactivity. The `Element` interface, the
`Frame` interface, and one element — a plain box with a background and children —
running layout, prepaint and paint against a fake `Frame` in a test.

Then stop and say so.

That discipline has caught real defects twice on `platform`. It was skipped once, on
`render`, and produced two thousand lines built on a call to the wrong function.

## Decide this first: how per-phase state is carried

GPUI's `Element` has two associated types:

    type RequestLayoutState: 'static;
    type PrepaintState: 'static;

`request_layout` returns the first, `prepaint` receives it and returns the second,
`paint` receives both. Go has no associated types on interfaces, so this does not
come across, and how you replace it shapes every element anyone ever writes.

Two shapes worth weighing:

- **Return `any` and pass it back.** Closest to GPUI. Costs a type assertion per
  element per phase per frame, and an allocation whenever the state does not fit in
  an interface word.
- **Keep the state on the element.** An element is a value built fresh each frame and
  discarded after paint, so it can hold `childLayoutIDs` as a field that
  `RequestLayout` fills in and `Paint` reads. The interface takes a pointer receiver
  and the phases mutate. No assertions, no boxing, and the compiler checks the types.

The second is the more Go-ish answer and I expect it to win, but it has a consequence
to state explicitly: elements become non-copyable in practice, and "an element is a
value" in `docs/packages.md` starts to mean "a value addressed by pointer". Say which
you chose and what it costs in `doc.go`.

Whichever you pick, the phase ordering invariant has to be enforceable: prepaint may
not request layout, paint may not register hit regions. Decide whether that is a
documented rule, a runtime check under `facet_debug`, or something the types prevent.

## The Frame interface

`element` declares it. `window` implements it. Elements never import `window` — that
would be a cycle, and this interface is what breaks it.

`prompts/window.md` lists what `window` can supply from the packages that exist:

    request layout         a node in the layout tree, with a style and children
    computed bounds        after the solve, for a node
    register a hit region  during prepaint, returning a handle
    paint primitives       the six in `scene`
    shape text             through `text`, at the window's scale factor
    scale factor           for anything that has to snap to device pixels

Keep it narrow. Every method is something an element may call at any point in any
phase, and every one of them is a thing `window` must then guarantee forever.
`render.Renderer` and `platform.Platform` are the two interfaces we have already
committed to; this is the third, and it is the one users' code sits closest to.

If you need something from `window` that is not on that list, say so and why rather
than adding it quietly. If something on the list turns out not to be needed, drop it.

## The builder is yours

This is settled and it is not where it looks like it should go.

The fluent chain — `div().Flex().Gap(4).Bg(c)` — lives on the element, not on
`style.Refinement`. `style` exposes mutators on `*Refinement`; the element holds a
refinement and each builder method mutates it through the pointer and returns the
element:

    func (d *Div) Flex() *Div { d.style.SetDisplay(style.DisplayFlex); return d }

The reasoning, so you do not undo it: value-receiver builders on `Refinement` cost
~19 ns per call against a 2.55 ns copy control, because copy-in-mutate-copy-out is
the whole cost and it grows with the struct. Pointer receivers on `Refinement` fix
that and break the API — `Refinement{}.Flex()` does not compile, a composite literal
is not addressable. GPUI hit the same wall in a language where the copy would be
free, and resolved it the same way: `styled.rs:22` declares
`fn style(&mut self) -> &mut StyleRefinement` and every fluent method mutates through
it. The element is already behind a pointer, so it is the receiver that works.

The method-per-property list is long and mechanical. It is not part of the first
milestone.

## What the package owns

**The `Element` interface** and the three phases.

**The element tree.** Children are elements; a container holds a slice of them. There
is no reconciler and no diff — the tree is rebuilt every frame from retained state,
and that is the point.

**`div`.** The one general-purpose container, the way GPUI has one. Not a widget
library; `ui` is that, and it is built entirely from what this package exports.

**The fluent builder**, per the section above.

**`Render`.** A view is an entity whose `Render` produces an element tree. `app` owns
entities and knows nothing about drawing, so the interface that ties the two together
lives here.

## Decide these too

Write the answers in `doc.go`.

**Element identity across frames.** GPUI gives elements an optional ID and keys a
per-frame state map by it — scroll offsets and the like. `docs/packages.md` says
anything that outlives the frame belongs in an entity. Those conflict, `window`'s
prompt raises the same question, and the two of you must not answer it differently.
If elements get IDs, say what they are for and what evicts a stale entry, because a
map keyed by element ID with no eviction grows with every list the user scrolls.

**Whether children are typed or erased.** `[]Element` is the obvious answer and it
boxes every child. Say whether that matters at the sizes we expect and what the
alternative would have been.

**What an element costs to build.** The tree is rebuilt every frame, so the
constructor is on the hot path in a way a retained tree's is not. Benchmark building
a small tree — a container with ten children — and put the number in `doc.go`. That
number is the floor on Facet's frame time and nobody has one yet.

## Invariants

Elements are values built fresh each frame and discarded after paint. Anything that
must survive the frame belongs in an entity.

Layout, prepaint and paint run in that order and no phase reaches backwards.

`element` imports `geometry`, `colour`, `scene`, `style`, `layout`, `text` and `app`,
and nothing else of ours. It does not import `window`.

No widget registry, no enum-plus-switch dispatch. Adding an element type adds a file.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

The layering test passes. A test drives one element through all three phases against
a fake `Frame` and asserts what reached it — the layout request, the hit region, the
primitives with their bounds — not that nothing returned an error. A test shows a
phase called out of order is caught, by whatever mechanism you chose. A benchmark
reports the cost of building a ten-child tree.

Conventional commits, one file per commit, staged by path.

## Habits from earlier rounds

When a struct of options exists, at least one test passes it empty. Two defects in
`platform` survived their suites because every test configured the field it was about
to check.

Assertions should know the answer. A test that a size "is positive" passed while the
size was wrong by a window frame. If a contract is in a doc comment, the test checks
the contract.

When a decision is meant to be settled by measurement, measure first and let it
decide — including when you are confident you already know. A plan for `style`
recently named a performance number for an expression that does not compile.
