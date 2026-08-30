# gpui-go

A GUI framework for Go: pure Go in, native desktop application out. The design
follows GPUI, the framework behind the Zed editor.

`docs/architecture.md` defines the layer stack, the seams between layers, and which
decisions are still open. Read it before structural changes, and link to it rather
than restating it.

## Commands

    go build ./...
    go test ./...
    go vet ./...
    gofmt -l .

`gofmt -l .` must print nothing. There is no linter configured yet.

## Reference checkouts

    ../gpui     Zed's GPUI — the conceptual source
    ../wails    Wails v3 — the platform layer we borrow from

Prefer borrowing working code from `../wails` over writing new platform code.

## Working alongside other agents

Several agents may be working here at the same time, each on a different package.

- Stay inside the package you were given. If your change requires touching another
  package's exported API, stop and say so rather than editing it.
- Interfaces at layer boundaries are contracts. `render.Renderer`,
  `platform.Platform` and `element.Element` change by explicit decision, never as a
  side effect of an implementation.
- Do not create files that many packages append to — no central `types.go`, no
  widget registry, no enum-plus-switch dispatch. Adding a feature must not require
  editing a list somewhere else.
- One concept per file. Small files collide less.
- Tests live beside the code they test. No cross-package fixtures.
- Adding a dependency to `go.mod` requires saying which package needs it and why.

## Language

International (GB) English, in prose and in code: `colour`, `centre`, `behaviour`,
`initialise`, `serialise`. This covers package names, identifiers, comments and
documentation alike.

## Code style

Go 1.26.

- Exported identifiers carry doc comments. Unexported ones do when the reason for
  their existence is not obvious from the name.
- Comments explain why. What the code does should be readable from the code.
- Wrap errors with context and `%w`: `fmt.Errorf("load font %q: %w", name, err)`.
  Sentinel errors only where callers branch on them.
- Panic only for programmer error with no recovery — using a context off the UI
  goroutine, for example. Never for input.
- No reflection on a per-frame path. No code generation.
- Zero values are usable where that is reasonable.
- Concrete types in struct fields, interfaces at parameters.

## Commits

Conventional commits, one file per commit.

    feat(layout): solve flex basis before main-axis distribution
    fix(text): handle zero-width joiner in cluster breaking
    docs: record the rendering decision
    refactor(app): extract the effect queue from App
    chore: add gitignore

Scope is the package. The body explains why when the subject cannot. No file lists
in the body — the diff already says that.

## Documentation

Prose lives in `docs/`, lower-case filenames. State what we do. Explain why only
where the reasoning is not recoverable from the result, and do not justify decisions
against alternatives that were never taken.
