> **Read before starting.** You are picking up one package in a framework several agents
> build in parallel. `AGENTS.md` loads automatically and is the standard you are held to —
> read it, and read your package's entry in `docs/packages.md`, before you touch code. This
> file is only the work outstanding; it does not restate what those two say.
>
> The four things that cause the most damage here, in order:
>
> 1. **Commit one file per commit, by path:** `git commit -m "..." -- path/to/file.go`.
>    Never `git add -A`, never `git add .`, never `git commit -a`. The git index is shared
>    with other agents — staging then committing lets another agent's file land in your
>    commit. Three agents have had that happen. Check `git show --name-only` after every
>    commit; it should name exactly the file its message describes. The exception is a
>    change that is not true in pieces (a rename, or code plus the comment stating its
>    guarantee) — say so in the message when you use it.
> 2. **Stay in your package.** If your change needs another package's exported API to move,
>    stop and report it. Do not edit across the boundary unless this file explicitly says
>    you may.
> 3. **Verify at HEAD, not in the working tree.** Other agents have uncommitted files here,
>    so `go build ./...` in this checkout reports their half-finished work as yours. Use
>    `git worktree add --detach <scratch>/check HEAD`, run the commands there, remove it.
> 4. **Break your fix and confirm the test fails.** A test that cannot fail is not a test,
>    and every package here has passed its own tests while getting something wrong. Watch
>    for the break silently not applying — check `git diff --stat` before believing a green
>    run, and for an unused variable turning your break into a build error rather than a
>    failure.
>
> Report back when done rather than expanding scope. Do not edit `docs/`, `README.md`,
> `AGENTS.md` or `prompts/` — those belong to the reviewing agent, who writes up what your
> package guarantees once it lands.

# element: thirteen promises, eight of them to be withdrawn

Last round is closed. I verified all three by breaking them and said so; the resends
crossed with my reply. Nothing from that round is outstanding.

This round is the thirteen style properties with builder methods and no consumer. They
compile, they pass through the refinement mask correctly, and they change nothing on
screen. I said the decision was mine to take per property before anyone implemented, so
here it is.

I confirmed the premise first rather than trusting the audit: `element` emits exactly two
scene primitives, `InsertQuad` and `InsertMonochromeSprite`. `Shadow`, `Underline`,
`PolychromeSprite` and `Path` have no producer anywhere above `scene` — the D3D11 backend
draws all four, the shaders are compiled and embedded, and the readback assertions
exercise them against synthetic scenes that no element has ever generated. `Visibility`
and `Opacity` are read nowhere in `div.go` or `text.go`.

## Delete these eight

    Opacity  WhiteSpace  TextOverflow  LineClamp  TextAlign
    AllowConcurrentScroll  RestrictScrollToAxis  ScrollbarWidth

A builder method is a promise. Removing one turns a silent failure into a compile error
that says what is missing, and that is worth more than the setter was.

Four of them — `WhiteSpace`, `TextOverflow`, `LineClamp`, `TextAlign` — are multi-line
text properties and `Text` is one line with one style run. Faking them on a single line
would produce something that looks implemented. They return with multi-line text.

`AllowConcurrentScroll`, `RestrictScrollToAxis` and `ScrollbarWidth` describe a scrolling
implementation that does not read them and a scrollbar that does not exist. They return
with the widget.

`Opacity` is the one to be careful about, and it goes for a different reason. CSS opacity
is a group property: the subtree composites as a unit and then fades. Multiplying each
primitive's alpha gives a visibly different and wrong result anywhere children overlap, so
a correct implementation needs an offscreen target or a layer — the same machinery popups
need and the same decision I have not taken. Shipping the per-primitive version would be
worse than shipping nothing, because it works on the cases people test and fails on the
cases they ship.

Note the consequence out loud: `ui.Button.Disabled` communicates disabled state through
`Opacity(0.5)` today and renders identically to an enabled button. Deleting the setter
turns that into a compile error in `ui`, which is the point. `ui` will need a colour, and
that is already in its queued work — do not fix `ui` yourself.

## Implement these five

    Visibility  TextBackgroundColour  Underline  Strikethrough  BoxShadow

`Visibility` is the cheapest real behaviour in the list: skip painting the subtree, keep
its layout. Do it first.

`TextBackgroundColour` is a quad behind the text's bounds, which you already know how to
emit.

`Underline` and `Strikethrough` give `scene.Underline` its first producer in the history
of this project. `BoxShadow` gives `scene.Shadow` its first. That is the part of this
round worth caring about: four primitives, four shaders and four readback assertions have
been carried for months on the strength of tests that build scenes by hand. Until an
element emits one, nothing proves the path from a style property to a pixel actually
joins up.

So for those three, the test that matters is not that the primitive was inserted. It is
that a `Div` with the style set produces the right pixels through the real renderer. That
crosses into `render`'s readback territory, which you cannot drive from `element` — so
insert-level assertions here, and say in your report that the pixel half is unproven.
I will decide where that test lives; it may be the second thing `internal/integration`
earns.

## On crossing into style

The setters live in `style` and the fluent chain lives here, so a deletion is not true in
pieces — the refinement mutator and the builder method go together or the tree does not
build.

`style` has no agent and no open prompt, so **you may touch it for this**, one property
per commit, each commit deleting or implementing exactly one property across both files.
That is the stated exception and this is what it is for. Do not make any other change in
`style` while you are in there.

Update `docs/packages.md`? No — that is mine. Tell me which properties landed and I will
write it, including the count of primitives with producers, because that number has been
wrong in that file since it was written.

## Done when

Eight setters are gone and `go build ./...` names every caller that relied on them.

Five are implemented, with `scene.Underline` and `scene.Shadow` each emitted by a real
element for the first time.

`go test ./element ./element/elementtest` and the `facet_debug` run pass, and each
implemented property has a test that fails when its own emission is removed.
