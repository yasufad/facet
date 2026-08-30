# Assignment: layout

Port Taffy's flexbox solver to Go as the `layout` package. This is a port, not a
reimplementation: the algorithm and its edge cases come across intact and only the
language changes.

## Prerequisites

This package depends on `geometry`, which must be implemented and merged
before it can compile. If they are not there yet, wait — do not stub them. A
placeholder gets imported, drifts from the real API, and turns the merge into a
rewrite.

## Read first

1. `AGENTS.md` — conventions, commit style, and the attribution rule, which applies
   to every file you write here
2. `docs/packages.md` — the `layout` entry
3. `docs/sources.md` — what porting means as opposed to reading
4. `_upstream/taffy/src/` — the source
5. `_upstream/taffy/docs/style-properties.md`

Run `go run ./tools/upstream` if `_upstream/` is not there.

## Build

The flexbox algorithm and the tree plumbing it needs: node storage, the style input
type, intrinsic-size caching, and the measure-function hook that lets a leaf report
its own size — text uses that.

Taffy's own style types come across with it. `layout` defines `Dimension`,
`LengthPercentage` and `LengthPercentageAuto` itself; it must not import `style`,
which converts down to these.

Port the test suite as well. Taffy's tests are generated from browser behaviour and
are the only real evidence the port is correct. Bring them across mechanically —
they are worth more than tests you write yourself.

## Decisions already made

Grid is out of scope. Port the flexbox path and the shared tree code it needs, and
leave the rest.

A behavioural difference from Taffy is a bug in the port, not an improvement. If the
Rust looks wrong, it is far more likely encoding a spec detail or a browser
behaviour you have not hit yet. Port it faithfully and raise the question separately.

Order matters more than it looks. The sizing passes, `auto` margin resolution, min
and max clamping and baseline alignment interact, and reordering them for
readability breaks cases the tests will catch late. Keep the structure recognisable
against the original.

Every file carries an attribution header naming the Taffy file it came from and its
MIT licence, and Taffy goes into `NOTICE`.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test ./internal/layering
    gofmt -l $(go list -f '{{.Dir}}' ./...)

`doc.go` states what the package owns and its invariants, and says plainly that this
is a port and where from. The ported test suite passes. Where a test does not pass,
say which and why rather than deleting it.

## Out of scope

Style properties that are not layout inputs — colours, borders, shadows, text
styling. Those live in `style`, which converts down to this package's types.
