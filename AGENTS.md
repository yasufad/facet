# Facet

A GUI framework for Go: pure Go in, native desktop application out. The design
follows GPUI, the framework behind the Zed editor. Module path
`github.com/yasufad/facet`.

`docs/architecture.md` defines the layer stack, the seams between layers, and which
decisions are still open. Read it before structural changes, and link to it rather
than restating it. It is not loaded automatically, so small changes do not pay for
it.

## Commands

    go build -o bin/ ./...
    go test ./...
    go vet ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

Build output goes to `bin/`, which is gitignored. Plain `go build ./...` drops an
executable in whatever directory you ran it from; always pass `-o bin/`. Nothing
buildable should ever appear at the repository root.

The format check must print nothing. Scope it through `go list` rather than running
`gofmt -l .`, which walks into `_upstream/` and reports every upstream file. There
is no linter configured yet.

## Upstream projects

    go run ./tools/upstream            sync the checkouts to their pinned commits
    go run ./tools/upstream -update    move the pins to their branch heads

`upstream.pins` records the exact commit of each project we read. The checkouts land
in `_upstream/`, which is gitignored and invisible to the Go build tools, and are
shallow, blobless and sparse — reading `crates/gpui` costs 15 MB rather than all of
Zed. Moving to a newer upstream is a one-line diff to `upstream.pins`, not something
that happens quietly when someone re-clones.

    GPUI          https://github.com/zed-industries/zed — crates/gpui
    Taffy         https://github.com/DioxusLabs/taffy
    Wails v3      https://github.com/wailsapp/wails — v3/ on master
    typesetting   https://github.com/go-text/typesetting

`docs/sources.md` says which layer draws on which, and which parts none of them
provide. Read it before assuming something has to be written from scratch.

## Attribution

Facet is Apache-2.0. Taffy, Wails and go-text are MIT, and their notices travel with
any code we take. When you port or copy upstream code:

- Head the file with an attribution comment naming the upstream project, the file it
  came from, and its licence.
- Add the project to `NOTICE` if it is not listed there already.

Reading an upstream project and writing your own implementation needs neither.

## Working alongside other agents

Several agents may be working here at the same time, each on a different package.
`docs/packages.md` says what each package owns, what it may import, and the
invariants it holds. Read your package's entry before you start.

- Stay inside the package you were given. If your change requires touching another
  package's exported API, stop and say so rather than editing it.
- Import only what your entry in `docs/packages.md` permits. A layering test
  enforces it; a failure means the design changed and needs deciding, not patching.
- Interfaces at layer boundaries are contracts. `render.Renderer`,
  `platform.Platform` and `element.Element` change by explicit decision, never as a
  side effect of an implementation. Plan a change that crosses a layer boundary or
  alters one of these before writing it.
- Do not create files that many packages append to — no central `types.go`, no
  widget registry, no enum-plus-switch dispatch. Adding a feature must not require
  editing a list somewhere else.
- One concept per file. Small files collide less.
- Tests live beside the code they test. No cross-package fixtures.
- Write a test when the logic is non-obvious, or to pin a bug as it is fixed. A
  test that restates the implementation is noise, and a change is not incomplete
  for lacking one.
- When coverage is bounded — a ported suite partly adopted, a case left unhandled —
  record what was left out and why, beside the code. A pass count with no exclusion
  count reads as full coverage to everyone who comes after.
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
- A comment that states a guarantee is part of that guarantee. When you change how
  something is enforced, or what it costs, the sentence describing it changes in the
  same commit. Prose written when it was true is the easiest thing to leave behind.
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

## Instruction files

This file is the only place instructions live. `CLAUDE.md` exists solely because
Claude Code does not read `AGENTS.md`; it imports this file and holds nothing of its
own.

Instructions that apply to one package go in `.claude/rules/`, scoped with `paths:`
frontmatter so they load only when the matching files are touched. Add one there
rather than growing this file.

`prompts/` holds one assignment per package: what to build, what to read, what has
already been decided, and what done means. If you have been told to implement a
package, read `prompts/<package>.md` first.
