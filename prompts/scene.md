# scene: reopened — Greyscale

One word. `scene/sprite.go` spells it `Grayscale`; `AGENTS.md` asks for GB English in
identifiers, so it is `Greyscale`.

`style` and `layout` renamed `Center` to `Centre` this week and `geometry` is doing
the same; this is the last of that sweep.

Check whether `render` reads the field — if it does, that rename crosses a package
boundary, so say so and stop rather than editing `render` yourself. It is a
two-line change in each, but it is two agents' work, not one.

`render/d3d11` keeps `semGrayscale`, because that names the HLSL semantic
`INST_GRAYSCALE` and the string has to match the compiled shader bytecode.
`AGENTS.md` now records that exception, so do not follow the rename in there.

## Done when

    go build -o bin/ ./...
    go test ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

Nothing in `scene` spells it `Grayscale`, and if `render` has to change, that is
raised rather than done.

Then retire this prompt.
