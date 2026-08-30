# Assignment: render

Build `render`: the `Renderer` interface, and the Direct3D 11 backend behind it.

This is the last layer before something appears on screen. Everything beneath it is
built and verified in isolation — `scene` produces ordered, batched primitives,
`text` produces coverage masks, `platform` produces a window and a native surface —
and none of it has ever been drawn. Expect assumptions to break here; that is what
this layer is for.

Read `docs/packages.md` for the `render` and `scene` entries, and the Rendering and
requirements sections of `docs/architecture.md`.

Metal and Vulkan are separate assignments. Do not write them. Do design the
interface so they fit, and say in a comment where you know a backend will differ.

## Stop after the interface

`render.Renderer` is one of the three layer boundaries `AGENTS.md` names. Propose it
first: one commit of types, method signatures and doc comments, no implementation.
Then stop and say so.

That discipline has now paid for itself twice on `platform`. Both rounds of
interface review found things — wheel deltas flattened to lines, window geometry in
the wrong units — that would have been far more expensive to change once backends
and callers existed.

What it has to cover: creating a renderer for a native surface, resizing with the
window, drawing a `scene.Scene`, presenting, and the atlas operations below.

## The atlas split

Three packages touch atlases and the boundaries are already drawn. Do not redraw
them.

    text     rasterises a glyph to a coverage mask, caches by face/size/subpixel.
             Never packs, never uploads. text.AtlasEntry is a mask plus bounds.
    scene    carries a reference only: scene.AtlasTile names a texture, a tile and
             a rectangle. It does not own or allocate anything.
    render   owns the GPU textures, allocates tiles within them, uploads masks,
             and resolves scene.AtlasTile at draw time.

`text` and `render` do not import each other. `window` sits above both and wires
them: it asks `text` for a mask, hands it to `render` for upload, and puts the
returned `scene.AtlasTile` into the primitive. Your interface has to make that
possible without either package learning about the other.

Two texture kinds, as `scene.AtlasTextureKind` already distinguishes: monochrome
coverage for glyphs, polychrome for images and emoji.

## Shaders ship as bytecode

Users run `go build` and nothing else. No `fxc`, no `dxc`, no Windows SDK.

Compile HLSL to DXBC ahead of time, check the bytecode in, and `go:embed` it. Put
the sources under `render/d3d11/shaders/` next to their compiled output so the two
stay together, and add whatever compiles them under `tools/`, so regenerating is a
documented command rather than folklore on one machine.

This is not a detail to defer. Reaching for a runtime `D3DCompile` call is the
obvious shortcut and it puts a platform SDK back into every user's build through
the side door.

## Then the D3D11 backend

Instanced draws, one per batch. `scene.Batches()` already groups consecutive
primitives of one kind — and one texture, for sprites — into exactly the runs a
single instanced call should draw. Walk them; do not regroup.

Six primitives to draw: quads with per-corner radii and per-edge borders, blurred
shadows, monochrome sprites, polychrome sprites, filled paths, and underlines
including the wavy variant.

The `Renderer` sees nothing above `scene`. No elements, no styles, no entities.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

The layering test passes: `render` imports `geometry`, `colour`, `scene` and
`platform`, nothing else of ours.

And the thing this is all for: a program that opens a window and draws a coloured
rounded quad on it. Put it in `examples/`, keep it under fifty lines, and make it
the thing you run to know the stack works. A screenshot test is not required; a
program someone can run is.

Conventional commits, one file per commit, staged by path.

## Constraints

No cgo. `CGO_ENABLED=0` builds on every target. D3D11's COM interfaces are vtables
reachable through `syscall`, the same way `third_party/w32` reaches Win32 — that is
the pattern to follow, and it is why Windows is first.

`unsafe` is permitted in `platform` only. If `render` needs it for COM vtable calls,
raise that rather than assuming: it is a real change to a stated invariant and the
answer is probably yes, but it should be decided rather than discovered in review.

No Go pointer goes into memory the GPU or the OS owns, for the same reason as in
`platform`: the collector cannot see it, and the object is freed while the address
is still live.

## Two habits worth carrying in

Every `platform` round found a defect that its own tests could not see, and twice
the cause was the same — the tests configured the options, so the zero value went
unexercised. When a struct of options exists here, at least one test passes it empty.

And assertions should know the answer. `Size()` returning 625.3 passed a test that
checked it was positive. If a contract is written in a doc comment, the test checks
the contract, not that the number is plausible.
