# geometry: reopened — Centre

`geometry` spells centre the American way, throughout its exported API:

    geometry/anchor.go    TopCenter, BottomCenter, LeftCenter, RightCenter
                          Anchor.IsCenter()
    geometry/bounds.go    Center, CenteredAt
    geometry/corners.go   the four Center anchors again

`AGENTS.md` names `centre` as one of its examples, and it covers identifiers, not
only prose. `style` and `layout` renamed their halves this week; this is the same
sweep, in the package everything else imports.

Rename them, along with the tests and doc comments that use them.

## Why now

Nothing outside `geometry` uses these names — I checked, and the four anchors and
`IsCenter` appear only within the package today. `Center` and `CenteredAt` on
`Bounds` are the ones with a future: `element` and `ui` will reach for them, and
`window` will when it centres a window on a display.

So the blast radius is one package this week and four next month. That is the whole
argument for doing it now.

## Not everything spelled the American way is wrong

`render/d3d11` keeps `d3d11ColorWriteEnableAll` and `semColor`, because those quote
`D3D11_COLOR_WRITE_ENABLE_ALL` and the HLSL semantic `INST_COLOR` — the latter has to
match the compiled shader bytecode exactly. `AGENTS.md` now records that exception.
Nothing in `geometry` is quoting anything, so it does not apply here.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

No identifier, comment or doc string in `geometry` spells it `Center`.

One file per commit, staged by path. Other agents are working in this tree.

Then retire this prompt — and check the `geometry` entry in `docs/packages.md` still
describes the package accurately before you do, because that is what outlives it.
