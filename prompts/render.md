# Assignment: render

Both diagnoses were right and both fixes are correct. The vtable index is 15,
`release` tolerates nil and clears what it drops, and the input element semantics
are fixed. `go run ./examples/quad` now opens a window and stays up instead of
dying on the first call. The `unsafe` invariant was raised and recorded in
`docs/packages.md` rather than assumed. That is the round doing what it should.

One gap left, and it is the same shape as last time one level up.

## Nothing verifies that anything is drawn

`d3d11_debug_test.go` has eleven assertions and every one of them is `err != nil`.
It proves the API does not error. It does not check a single pixel.

I tried to check from outside — `PrintWindow` with `PW_RENDERFULLCONTENT` against
the live window, sampling three points after drawing a full-window magenta quad. The
capture succeeded and every pixel came back black. That is **not** evidence the
renderer is broken: `PrintWindow` is documented as unreliable for flip-model DXGI
swapchains, because DWM composites those independently of the GDI surface. The
honest reading is that the question cannot currently be answered from outside the
package, and it is not answered from inside either.

So the state is: it does not crash, and all six primitives can be submitted without
error. Whether the output is correct is unknown.

## Add a readback

Copy the back buffer into a staging texture with CPU read access, `Map` it, and
assert pixel values. The machinery is already there — `ID3D11DeviceContext::Map` at
vtable 14 and `d3d11MappedSubresource` are in use for the instance buffer, and
`CopyResource` is the one call you are missing.

Keep it behind `facet_debug` and off the release path; a staging copy per frame is
not something a shipping renderer should carry.

Then assert things worth asserting:

- A full-window quad of a known colour makes the centre pixel that colour.
- A quad with a corner radius leaves the corner pixel as the background and the
  centre as the fill. That checks the shader, not just the plumbing.
- A red quad drawn over a blue one shows red where they overlap and blue where they
  do not — the draw order the `scene` R-tree computed, actually reaching the screen.
- A monochrome sprite uploaded with known coverage samples to the tint colour at
  full coverage and the background at zero.

Those four would have caught the vtable bug, and they will catch the next four
things wrong in the shaders, which is where the remaining bugs are.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

A `facet_debug` test reads the back buffer and asserts pixel colours for at least
the four cases above. `go run ./examples/quad` shows a blue rounded quad with a
yellow border — say so in your report only if you have looked at it.

Conventional commits, one file per commit, staged by path.

## Worth carrying

Last round the lesson was that compiling is not evidence in this package, because
almost everything here is an untyped number — vtable slots, strides, register
indices, format enums. This round refines it: not erroring is not evidence either.

Every COM call can succeed and still produce a black window, because a shader bound
to the wrong register, a stride off by four bytes, or a swapped colour channel are
all perfectly legal operations. The only assertion that means anything at this layer
is one that reads the pixels back.
