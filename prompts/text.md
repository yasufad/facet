# Assignment: text

The shaping side of this package is in good shape — font loading and matching,
script and bidi segmentation, the run cache, line breaking and wrapping. Keep it.

The rasteriser goes. So do three errors in shared files.

## 1 — Replace the custom rasteriser with golang.org/x/image/vector

`raster.go` and `outline.go` were written because a dependency looked forbidden.
It was not: `text/` already imports `golang.org/x/image/math/fixed`, which
typesetting's API forces on it. `golang.org/x/image` is a dependency of this package
already, and `x/image/vector` is part of it.

The rule has been corrected in `AGENTS.md`: a dependency a package genuinely needs
is not blocked. Declare it, contain it, record its licence. Do not hand-write a
worse replacement to avoid one.

The custom rasteriser is also worse, structurally. From `raster.go`:

    const supersample = 4   // "gives 16 coverage levels per pixel"
    ...
    if acc.cover[row+x*supersample+sx] != 0 {   // binary in/out per subsample

Each subsample is a yes-or-no test, so a pixel can carry seventeen distinct coverage
values before being scaled to a byte. That is not a tuning parameter, it is the
ceiling of the algorithm, and text is where it shows: at 10 to 14 pixels the stem
edges land on those steps and stems come out unevenly weighted. `x/image/vector`
computes analytic area coverage — exact, all 256 levels — with SIMD paths on amd64
and arm64.

One note on the claim in `docs/architecture.md` that the masks are "visually
indistinguishable from FreeType's light autohinter at the same supersampling
factor": FreeType does not supersample. There is no shared factor to hold constant,
so that comparison cannot be made. Be wary of a qualifier that makes a claim true by
making it unfalsifiable — it is the sentence that stopped anyone questioning this.

Delete `raster.go` and `outline.go`, rasterise through `x/image/vector`, and keep
the mask type and the atlas exactly as they are: `RasterMask` is our type and
nothing above `text` should notice the change.

Before and after, render the same glyphs at 10, 12, 14 and 16 pixels and record the
timings. If `x/image/vector` somehow loses, say so with the numbers and we will look
again — but decide from output, not from reasoning about it.

## 2 — NOTICE is missing golang.org/x/image

You corrected go-text to Unlicense OR BSD-3-Clause in `38ed50b` — that one is done,
and the original error was mine: `AGENTS.md` told you it was MIT. That line is gone,
replaced by the rule to copy licence name and text out of the upstream `LICENSE`
file and trust no summary, this one included.

`golang.org/x/image` is still absent. It is BSD-3-Clause, copyright The Go Authors,
and it is a direct dependency whether or not the rasteriser change lands. Copy the
text from `go env GOMODCACHE`, not from memory.

## 3 — go.mod adds a dependency without declaring it

`golang.org/x/image` is now a direct requirement. `AGENTS.md` asks the commit that
adds a dependency to say which package needs it and why. Say it: typesetting returns
`fixed.Int26_6`, so `x/image/math/fixed` is unavoidable, and `x/image/vector` is the
rasteriser.

## 4 — docs/architecture.md

Two things. The Text section's rejection of `x/image/vector` as "not permitted" was
wrong and should be replaced by what you actually find in the comparison above.

And line 3 still reads "Status: planning. Sections marked *open* are not decided."
There are no `*open*` markers left — rasterisation was the last one. Five packages
are built. Say where the project actually is.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

`text/` compiles — it currently does not (`undefined: font` in `metrics.go`,
`cannot index acc` in `raster.go`). Glyph masks come from `x/image/vector`, with the
comparison recorded. `NOTICE` matches the upstream licence files. The dependency is
declared in its commit.

One conventional commit per change, staged by path. `NOTICE`, `go.mod` and
`docs/architecture.md` are shared files — stage them by name and check
`git show --name-only` afterwards, so another agent's work does not travel with
yours.

## One more thing

`atlas.go:116` calls `geometry.BoundsToDevicePixels(logical, 1.0)` with a hardcoded
scale factor. The atlas needs the real display scale; at any other factor the tiles
will disagree with the geometry around them by a pixel. That conversion now snaps
both edges and derives the size, so it is correct once you feed it the true scale.
