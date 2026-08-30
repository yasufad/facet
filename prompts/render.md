# Assignment: render

This round did the thing. Readback through a staging texture, six pixel tests, and
two more wrong vtable indices found the only way they ever could have been —
`DrawInstanced` at 20 was `DrawIndexedInstanced`, silently drawing nothing with no
index buffer bound, and `Draw` at 12 was `DrawIndexed`. Every COM call succeeded
throughout.

The tests have teeth. I put the `DrawInstanced` index back to 20 and five of the six
failed; restoring it passed them. The path test correctly survived, because it uses
`Draw`. That is a test suite that can actually tell you something.

One gap, and it is a coverage regression rather than a defect.

## Three primitives lost their coverage

`e9eba45` had a smoke test that inserted all six primitives and checked the calls
succeeded. `5018eff` replaced it with pixel tests covering three. The other three
now have nothing at all:

    Quad                pixel-asserted
    MonochromeSprite    pixel-asserted
    Path                pixel-asserted
    Shadow              no coverage
    PolychromeSprite    no coverage
    Underline           no coverage

They went from weak coverage to none. That is worse than where the round started for
half the primitive set.

It matters more than usual here because of the hit rate: two of the paths that *were*
tested had wrong vtable indices. Each of the untested three has its own shader, its
own instanced draw call, and its own set of untyped numbers — a stride, a register,
a format enum, a slot. There is no reason to expect them to be in better shape than
the ones that were wrong.

Shadow is the one I would bet on being broken. A Gaussian blur of a rounded
rectangle is easy to get subtly wrong and impossible to notice without measuring —
a blur radius off by a factor, a sigma in the wrong units, the inset and drop cases
swapped. PolychromeSprite samples a different texture format, BGRA rather than R8,
which is a separate path through the atlas. Underline has a wavy variant with its
own maths.

Add a pixel assertion for each:

- **Shadow** — a shadow under an opaque quad: fully opaque directly beneath the
  quad's edge, partially covered a few pixels out, background well outside. That
  checks the blur actually falls off rather than being a hard rectangle or nothing.
- **PolychromeSprite** — upload a two-colour image and assert both colours land in
  the right halves. That catches a swapped channel order, which BGRA makes easy.
- **Underline** — a straight underline of known colour and thickness: coloured on
  the line, background a few pixels above. Then the wavy variant, asserting it is
  coloured somewhere the straight one is not.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

All six primitives have a pixel assertion. Regressing any single draw call's vtable
index makes at least one test fail — worth checking, the way I checked yours.

Conventional commits, one file per commit, staged by path.

## Worth carrying

Replacing a weak test with a strong one is right. Replacing a weak test with a strong
one that covers less is a trade, and it needs to be deliberate — the primitives that
lost coverage were not mentioned in the report, so it read as pure improvement.

When a test is deleted, the thing it covered either has better coverage now or has
none. Say which.
