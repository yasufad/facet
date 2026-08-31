# element: three doc comments still describe the model I got wrong

The package is finished and retired correctly. This is one small thing, reopened
because `window` is about to be written against this interface and will read these
sentences as the specification.

`element/frame.go`:

    59  // IsHovered reports whether the given hit region was hovered in the rendered frame.
    62  // IsActive reports whether the given hit region was actively pressed in the rendered frame.
    65  // IsFocused reports whether the given focus identifier held keyboard focus in the rendered frame.

That is the model I gave you and then corrected. The code does the right thing and
`docs/packages.md` describes the right thing, but these three comments still say
"rendered frame", which is the previous frame.

They should say what actually happens: the region is registered during prepaint of
this frame, `window` resolves the pointer against those regions at step 5, and the
query answers for this frame. `window`'s agent implementing to "the rendered frame"
would produce hover that lags and hit regions that never match.

`AGENTS.md` says a comment stating a guarantee is part of that guarantee. These are
on a layer-boundary interface, so they are the guarantee.

## Done when

The three comments describe the in-frame hit test, and nothing in `element` still
says hover is answered from the rendered frame.

Then retire this again.
