# element: four findings from the first widget, decisions taken

`ui` built a button and reported six gaps. Four are yours. The report was the point of
that milestone and it delivered — leaving the layering test failing rather than
patching it around was the right call, and it is how a design change is supposed to
arrive.

## 1 — `element.NodeID`, and this one is urgent

`internal/layering` is red on `main`. `ui/button.go` imports `layout` for exactly one
reason: `Element.RequestLayout` returns `layout.NodeID`, so every implementer must
name it merely to satisfy the interface.

That is a tax on implementing `Element`, not a use of the layout engine. `ui`'s whole
invariant is that a widget is built from `element` and `style`; forcing it to name
Taffy's vocabulary to declare a method contradicts that, and `docs/packages.md`
already says the two vocabularies should meet at one boundary rather than throughout.

Declare an alias and use it in the interface:

    type NodeID = layout.NodeID

An alias, not a defined type: `window` and `element` pass these to `layout` constantly
and a conversion at every call site would be churn for nothing. The types stay
identical; only the name an implementer has to write changes.

`Frame.RequestLayout(style layout.Style, children []NodeID)` keeps `layout.Style` for
now. A widget that builds its own element rather than delegating to `Div` would have
to name it, and no widget does yet. When one does, that is the moment to decide
whether to alias `Style` too or admit `layout` into `ui` — do not pre-empt it.

Land this first. The tree is red until it does.

## 2 — Text does not inherit its parent's text properties

`Div` resolves `hoverStyle`, `activeStyle` and `focusStyle` in paint and applies them
to its own quad. A child `Text` sees only its own refinement, so a button cannot
change its label colour on hover, which is the most ordinary thing a button does.

`docs/packages.md` has said since before any of this existed that inheritance is
explicit and confined to text properties. That is the design; it is simply not
implemented. Text colour, family, size, weight, style, line height and alignment
should reach a `Text` child from the resolved style of its ancestors.

Work out where the inherited style travels. Passing it down the phase calls changes
the `Element` interface for everyone; carrying it on `Frame` as a stack pushed and
popped around children is what GPUI does with `text_style_stack`, and costs
implementers nothing. I expect the second, but say which and why in `doc.go`.

Pseudo-states follow from it: a `Hover` refinement on the parent that sets a text
property has to be in force while the child paints.

## 3 — `ClickEvent` should carry element-local coordinates

Window-relative only, so every widget that cares subtracts its own bounds origin.
Add the local position. The element knows its bounds when it synthesises the click,
so nothing else has to change.

## 4 — An exported test double, last

`element`, `window` and `ui` each hand-wrote a fake `Frame`, now twenty-one methods.
The report undersells the problem: `ui/button_test.go` has to import `platform`,
`scene` and `text` to implement one, and `ui` is forbidden all three in production. It
only works because the layering test reads non-test imports. So a package above
`element` cannot test an element without naming packages it is not allowed to depend
on, and an external widget author is worse off still.

Export one: `element/elementtest`, with a `Frame` implementation that records what
reached it — layout requests, hit regions, dispatch nodes, primitives — and lets a
test set hover, active and focus. A subpackage inherits `element`'s import permissions,
so it may name `scene` and `text` freely.

This is last because nothing is broken without it. It is also the thing that decides
whether writing a Facet widget outside this repository is pleasant or miserable, so it
is not optional.

## Done when

`element.NodeID` exists, `Element` uses it, and `ui` no longer imports `layout`. The
layering test is green.

A test shows a parent's hover refinement changing a child `Text`'s colour.

`ClickEvent` carries local coordinates.

`element/elementtest` exists and `ui`'s button tests use it instead of a local double.

## Worth carrying

The button was assigned to test one claim — that a widget needs nothing the framework
does not expose — and it disproved it in six places, four of them here. A milestone
whose deliverable is a list of what went wrong is worth more than one that reports
success, and this is the second time that has been true in this project.
