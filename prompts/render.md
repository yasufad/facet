# Assignment: render

The shape of this is right. `Renderer` is a clean boundary, the atlas split is
respected, the six primitives each have a shader, twelve DXBC binaries are committed
and embedded, `tools/compile_shaders` makes regenerating them a documented command,
and COM is reached through `syscall` with no cgo. That is a lot of correct
structure.

None of it has ever run.

## The example does not work

    $ ./bin/quad.exe
    CreateSwapChainForHwnd: hr=0x00000001

That is the first thing the program does. `hr=1` is `S_FALSE`, and the check is
right to reject it — the swapchain pointer came back nil.

### Bug 1: the DXGI vtable index is wrong

    // IDXGIFactory2::CreateSwapChainForHwnd is vtbl index 13
    r1, _, _ = factory.call(13, ...)

Index 13 is `IDXGIFactory1::IsCurrent`, which returns `TRUE` — which is exactly the
`0x00000001` you are seeing, and why the out-pointer stays nil.
`CreateSwapChainForHwnd` is **15**:

    IUnknown          0  QueryInterface, AddRef, Release
    IDXGIObject       3  SetPrivateData, SetPrivateDataInterface, GetPrivateData, GetParent
    IDXGIFactory      7  EnumAdapters, MakeWindowAssociation, GetWindowAssociation,
                         CreateSwapChain, CreateSoftwareAdapter
    IDXGIFactory1    12  EnumAdapters1, IsCurrent
    IDXGIFactory2    14  IsWindowedStereoEnabled, CreateSwapChainForHwnd

I changed it to 15 and the call succeeds. `GetAdapter` at 7 and `GetParent` at 6 are
both correct, so this is a slip rather than a misreading — but a slip nothing could
catch, because nothing ever called it.

### Bug 2: the pipeline crashes on its own cleanup

With the index fixed it gets further and then dies:

    Exception 0xc0000005
    d3d11.(*comObject).Release      com.go:114
    d3d11.(*shaderProgram).release  pipeline.go:29
    d3d11.(*pipelineManager).release pipeline.go:301
    d3d11.newPipelineManager        pipeline.go:60

`newPipelineManager` fails partway, calls `release()` to clean up, and `release`
dereferences a COM pointer that was never set. Two things to fix: `release` must
tolerate nil members, and the construction failure it is reacting to needs finding —
it is being swallowed rather than reported.

There will be more after that. Nothing on this path has executed.

## Why every check passed anyway

`go build`, `go test`, `go vet` and `gofmt` are all green, and the example has never
drawn a pixel. Nothing tests the D3D path, and `go build ./...` compiles the example
without running it.

The prompt asked for "a program someone can run" and said "a program someone can run"
rather than a screenshot test deliberately — but running it has to be part of
finishing, not something the reviewer discovers. Add a smoke test under
`facet_debug` that does what `platform`'s tests do: create a real window, create the
renderer on its surface, submit a one-quad scene, present, tear down, assert no
error. That is the test that would have caught both of these in the first minute.

## The interface review was skipped

The assignment asked for the interface in one commit with no implementation, then a
stop for review. The whole backend arrived instead.

That step is not ceremony. On `platform` it caught wheel deltas flattened to lines
and window geometry in device pixels — both cheap to change at interface stage and
expensive afterwards. Here it would have caught the `unsafe` question below before
two thousand lines depended on the answer.

## unsafe in render was assumed, not raised

`render/d3d11` imports `unsafe`. `docs/packages.md` says `platform` is the only
package permitted it, and the prompt said to raise a change rather than assume one:
"it is a real change to a stated invariant and the answer is probably yes, but it
should be decided rather than discovered in review."

The answer is yes — COM vtable calls cannot be made without it. Update the `render`
entry in `docs/packages.md` to permit `unsafe` for COM interop, with the same
condition `platform` carries: only for memory the OS or driver owns, never for Go
objects, and every conversion commented.

Also worth noting: you added `examples` to the `unconstrained` list in the layering
test. That is the right call — an example is an application, not a layer — but it is
a change to the mechanism that enforces the rules, and those get mentioned.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

And the one that matters: `go run ./examples/quad` opens a window with a rounded
quad visible in it, and a `facet_debug` smoke test drives renderer creation and one
present against a real window.

Conventional commits, one file per commit, staged by path.

## Worth carrying

Two thousand lines of plausible COM code, every check green, and the first call was
to the wrong function. Compiling proves the types line up; it proves nothing about
whether the numbers are right, and in this package almost everything is a number —
vtable slots, buffer strides, register indices, format enums. None of them are
type-checked and all of them are wrong in the same silent way.

For this layer, "it builds" is not evidence. Run it.
