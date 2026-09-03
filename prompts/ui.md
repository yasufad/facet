# ui: the tree is green, and one test agrees with its own bug

Everything in the last prompt landed. `go build`, `go vet`, `gofmt` and `go test ./...`
are all clean at HEAD — the whole tree, for the first time in this project's life.
`internal/layering` had been red since before any of this work started.

I broke `element.Listener` and
`TestButtonClickInWindowMutatesEntityAndRendersNextFrame` failed. A `ui.Button`, clicked
in a real window, mutating entity state, with the next frame rendering it. That path is
what `docs/audit.md` opens by saying could not work, and it is now executed by a test that
fails when it stops working.

The `Disabled` affordance is the right fix rather than the minimum one — guarding the
hover, active and focus pseudo-states when disabled is the part that would have been left
out.

## The one finding

`TestTextFieldClickToPlaceCaret` cannot fail on its own subject.

I replaced **both** `ClosestIndexForX` call sites with a hardcoded `0`. The drag test
failed. `TestTextFieldClickToPlaceCaret` passed.

Its assertion is `expected cursor near 0`, so it clicks at the left edge — the one
position where a caret that always lands at 0 and a correct one agree. It is the same
shape as `window`'s capture test dragging into empty space, and that one took two rounds
to find.

Click in the middle of "Testing Caret Placement" and assert the index the click maps to.
Then break `ClosestIndexForX` and confirm it fails. The drag test already discriminates,
so keep it; this is the sibling that makes the pair meaningful.

Everything else in `TextField` survived breaking. Typing, backspace and delete, arrow keys
and selection, disabled, and drag selection all fail when their own mechanism is removed.

## The element identity decision, which I owe you

Your forecast is what settled it, and your fourth point — that a virtual list must report a
content extent Taffy cannot derive from children it never built — is the constraint I had
not considered.

I read `crates/gpui` against it. Two things came back:

`Element::request_layout` **receives** `global_id`. Identity is available in all three
phases; only `Window::with_element_state`, the frame-keyed store, is prepaint-and-paint.
So identity being unavailable during layout was never the obstacle.

And `list.rs` ignores the id it is given in `request_layout`, because its viewport lives
in a `ListState` handle — an entity, not frame-keyed element state.

That is the answer, and it is smaller than either of us expected. **A widget's viewport
belongs in the widget's own entity, which is where `ScrollView` already keeps it.** You do
not need retained element state in `RequestLayout`; you need last frame's measured
viewport, and `ScrollState` is already exactly that. The virtual list reads its own entity
before building children, the same way `ScrollView` reads its metrics today.

So the identity decision reduces to what is genuinely element-shaped, and I am taking it:
path-based `Frame.PushElementId`/`PopElementId`, state keyed by that path, carried between
the two frames `window` already keeps, pruned when a frame does not touch it. That is
GPUI's shape and it is confirmed from source. It goes to `element` and `window` when
something needs it — and on this reading, the list does not.

**What you do need is your point four**, and it is the real work: a way to declare a
content extent without children producing it. Propose the shape. A container property is
one answer and spacer `Div`s are another, and the difference matters to `layout`, so bring
me the trade rather than picking one. That is the next architectural decision and it is
yours to open.

## Then

Paint-phase rule: your context-splitting proposal is the strongest of the three — an
update that hands no `Context` cannot notify, and that is a mechanism rather than a rule.
It is an `app` change, so it is not yours. I am holding it until the virtual list settles,
because if frame metrics move onto `Frame` the whole question dissolves.

Nothing else outstanding here.
