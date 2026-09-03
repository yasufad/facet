> **Read before starting.** You are picking up one package in a framework several agents
> build in parallel. `AGENTS.md` loads automatically and is the standard you are held to —
> read it, and read `input`'s entry in `docs/packages.md`, before you touch code.
>
> Commit one file per commit, by path: `git commit -m "..." -- path/to/file.go`. Never
> `git add -A` or `git commit -a` — the index is shared with other agents and staging then
> committing lets their file land in your commit. Check `git show --name-only` after each
> one. Stay inside `input`. Verify at HEAD in a detached worktree, not in this working
> tree, which holds other agents' unfinished files. Do not edit `docs/`, `README.md` or
> `prompts/`.

# input: alias the whole vocabulary, not the part someone has tripped over

This is the third round of the same gap, and the first two were both my fault for scoping
it narrowly.

Round one: `ui` could not write a wheel handler, so you aliased `KeyEvent`,
`PointerEvent`, `TextEvent` and `WheelEvent`. Round two: `ui` still could not read
`event.Delta.Unit`, because `ScrollPixels` is a constant no type alias covers, so you added
`ScrollUnit` and its two constants.

Round three is here now and nobody has hit it yet, which is the only reason I am writing it
before it costs a round. `window` has landed IME routing and put `OnIMEComposition` on
`*Window`. When `ui`'s text field needs to register for composition, `element` has to
declare that method on the `Frame` interface — and `element` may not import `platform`, and
you do not alias `IMECompositionEvent`. It will hit exactly the wall `ui` hit twice.

## What to add

`platform` declares eight event types. You alias four.

    PointerEvent  WheelEvent  KeyEvent  TextEvent          aliased
    IMECompositionEvent  FocusEvent  ResizeEvent  ScaleChangedEvent   not

Add `IMECompositionEvent` now — it has a named consumer arriving.

Then decide the other three rather than defaulting either way, and say which you chose and
why. The question is not "might someone want it" but "can a package above `input` be
handed one of these and need to name its type". `FocusEvent` plausibly reaches an element;
`ResizeEvent` and `ScaleChangedEvent` look like they stop at `window`, which may import
`platform` freely. If a type genuinely never travels above `window`, aliasing it is noise —
say so and leave it.

Check the same thing for constants while you are there. That is what round two was, and a
type alias does not cover a constant.

## The rule this establishes

**Alias what a caller above you has to write, and work that out from their call sites
rather than from your own signatures.** Both earlier misses came from reading `input`'s
handler types and asking what they mention, instead of opening the file that was blocked
and reading what it names.

Put that in the package doc, in one sentence, so the next person adding an event type here
knows the alias is part of adding it rather than a follow-up.

## Done when

`element` could declare `OnIMEComposition(input.DispatchNodeID, func(input.IMECompositionEvent) bool)`
on `Frame` without importing `platform`. You do not add that method — `element` does, when
`ui` needs it — but the vocabulary has to be there first, and this is the direction data
travels: the name is declared before it is read.

A test in `input` that names only `input` types and constructs each aliased event,
including the IME one. If it compiles, the vocabulary is complete for that type.

The three undecided types are decided, either way, in writing.
