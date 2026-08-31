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
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

The `facet_debug` build turns on checks too expensive to leave in a release binary,
and the tests that exercise them are behind the same tag. Skip that run and those
tests never execute.

`vet`'s `unsafeptr` analyser is off because the vendored Win32 bindings trip it
seventeen times and the noise buries everything else. It cannot be scoped away —
`vet` analyses dependencies, so excluding `third_party` from the package list still
reports it. In exchange, every `unsafe.Pointer` conversion in our own code carries a
comment saying why it is sound, and `go vet ./...` unfiltered is worth reading by
hand after touching `platform`. A real bug has already hidden in that output once.

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

`go-text/typesetting` and `golang.org/x/image` are module dependencies rather than
checkouts, because `text` calls them rather than porting them. Read them through the
module cache; they are not in `upstream.pins` and `tools/upstream` will not fetch
them.

`docs/sources.md` says which layer draws on which, and which parts none of them
provide. Read it before assuming something has to be written from scratch.

## Attribution

Facet is Apache-2.0. Upstream notices travel with any code we take. When you port or
copy upstream code:

- Head the file with an attribution comment naming the upstream project, the file it
  came from, and its licence.
- Add the project to `NOTICE` if it is not listed there already.

Reading an upstream project and writing your own implementation needs neither.

Copy the licence name and text out of the upstream `LICENSE` file. Never write them
from memory, and do not trust a summary — including this one. Two attributions in
this repository have been wrong: a copyright holder who appears nowhere in Taffy's
licence, and go-text recorded as MIT when it is Unlicense OR BSD-3-Clause. Both read
plausibly. Reproducing the actual notice is the one thing these licences ask of us,
so it is the one thing worth checking at the source.

## Working alongside other agents

Several agents may be working here at the same time, each on a different package.
`docs/packages.md` says what each package owns, what it may import, and the
invariants it holds. Read your package's entry before you start.

- Stay inside the package you were given. If your change requires touching another
  package's exported API, stop and say so rather than editing it.
- Stop *before* committing, not after. Renaming an exported name that another package
  reads is not true in pieces: landing your half and raising the rest leaves `main`
  unbuildable for everyone else in the tree. Either the rename crosses the boundary in
  one commit — the stated exception below — or it waits until both sides can land
  together. `scene` renamed `Grayscale` correctly, raised `render` correctly, and
  still broke the build, because the raising happened after the commit.
- Adding a method to an interface one package declares and another implements goes
  backwards: the implementer adds the method first, where it satisfies nothing and
  breaks nothing, and the declaration follows once it is already satisfied. Declaring
  first stops the implementing package compiling until it catches up. If the method
  signature names a new type, that type is its own commit before either.
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
- A dependency a package genuinely needs is not blocked. Say which package needs it
  and why in the commit that adds it, keep its types inside that package so nothing
  above knows it exists, and record it in `NOTICE` from the upstream `LICENSE`. The
  import rules in `docs/packages.md` govern our own packages, not the module graph —
  do not write something worse by hand to avoid a dependency.

## Language

International (GB) English, in prose and in code: `colour`, `centre`, `behaviour`,
`initialise`, `serialise`, `greyscale`. This covers package names, identifiers,
comments and documentation alike.

A name that mirrors a foreign API keeps that API's spelling, because it is quoting,
not writing. `d3d11ColorWriteEnableAll` is `D3D11_COLOR_WRITE_ENABLE_ALL`, and
`semColor` holds the HLSL semantic `INST_COLOR`, which has to match the compiled
shader bytecode byte for byte. Renaming the Go identifier there loses the
correspondence that makes the constant checkable against the SDK header, and invites
someone to change the string next.

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

The exception is a change that is not true in pieces: one behaviour spanning two
files, or code and the doc comment describing it. Splitting those leaves a commit
that contradicts itself, which is worse than a slightly larger diff. The rule exists
for reviewable diffs and fewer collisions, not for its own sake.

A rename is the clearest case. Every use of the old name moves in the commit that
renames it, however many files that is. Renaming `TopCenter` in `anchor.go` and
leaving `bounds.go` and `corners.go` calling it produced five consecutive commits
where `geometry` did not compile — a correct end state reached through a history that
cannot be bisected or checked out. Each commit should build.

    feat(layout): solve flex basis before main-axis distribution
    fix(text): handle zero-width joiner in cluster breaking
    docs: record the rendering decision
    refactor(app): extract the effect queue from App
    chore: add gitignore

Scope is the package. The body explains why when the subject cannot. No file lists
in the body — the diff already says that.

Stage by path: `git add <file>`. Never `git add -A`, `git add .` or `git commit -a`.
Other agents have work in progress in the same tree, and a blanket stage commits
their unfinished files under your subject line. Check `git status` before you
commit and `git show --name-only` after; a commit should contain exactly the file
its message describes.

## Leading the work

One agent holds the architecture and reviews what the others land. If that is you,
this section is the job; everything else in this file is what you hold them to.

You own the shared files: `docs/`, `AGENTS.md`, `prompts/`, `README.md`,
`upstream.pins`, `internal/layering`. You do not write package code. Anything that
crosses a layer boundary is yours to decide, and deciding it means editing
`docs/packages.md` and the layering test, not approving a workaround.

**The rhythm.** An agent reports. You verify, then write what remains into
`prompts/<package>.md`, rewritten each round so the file is only the work still
outstanding. When it is empty, the package retires in the two steps under
"Instruction files". A prompt for an unstarted package onboards; a prompt for a
package in flight is a review addressed to someone who already has the code in their
head, and should not restate scope they know.

**Verify, do not accept.** Every package so far has passed its own tests while
getting something wrong, and each of these found a real defect:

- **Break the fix and confirm the test fails.** The single most valuable check. A
  test that cannot fail is not a test. A `Merge` assertion passed through an early
  return; a hover style never fired in production; a mask guard skipped every
  property above bit 63.
- **A test that needs a back door is evidence about the design.** One wrote an
  unexported field no caller can set, said so in a comment, and was committed.
- **Probe zero values.** `Options{}`, nil slices. Two `platform` defects survived
  five tests because every test configured the field it was about to check.
- **Check assertions know the answer.** "Size is positive" passed while a window was
  wrong by a frame.
- **Interaction needs a test that sets one thing and checks another.** Sixteen
  properties shared a mask bit; every test set the property it then read.
- **Compiling is not evidence and not-erroring is not evidence.** Three wrong D3D11
  vtable indices reached reviewed, green code. Read pixels back.
- **Measure the path users write**, not the one that is easy to benchmark.
- **Check the prose still matches.** A comment stating a guarantee is part of it.
- **Verify attribution at the upstream `LICENSE`.** Two notices here were fabricated
  and both read plausibly.

**Be wrong out loud.** Three defects in this repository came from guidance I gave.
Correcting one in the prompt, with the evidence, costs a round; leaving it costs the
package. Say plainly that it was yours.

**Watch the tree, not just the package.** Most breakage has come from correct work
landing in the wrong order. The rules under "Working alongside other agents" are each
an incident.

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

`prompts/` holds one assignment per package with work outstanding: what to build,
what to read, what has already been decided, and what done means. If you have been
told to implement a package, read `prompts/<package>.md` first.

An assignment is retired once its package is finished, so a missing file means the
work is done, not that it was never scoped. What the package guarantees lives in
`docs/packages.md`, which outlives the assignment.

Retiring is two steps and the deletion is the second. First move what the package now
guarantees into its `docs/packages.md` entry — the decisions taken, the invariants
that must survive a rewrite, the traps found. `doc.go` is not that place; nobody
reads another package's `doc.go` before starting work. Set the package's row in the
`README.md` status table at the same time, because that table is the only claim about
this project anyone reads before the code. Then delete the prompt.

The test for whether a prompt can be retired is that it is empty of work, not that
the last thing it asked about is done. If an item in it looks wrong or unnecessary,
say so and leave it — do not close it silently. Three packages have been retired with
open items so far, and each came back.
