# Assignment: layout

The Taffy flexbox port is done and committed: 28 commits, 2500 fixtures passing,
`build`, `vet`, `gofmt` and the layering test all clean. Every ported file names its
upstream source. This round closes a coverage gap and makes the port auditable.

Do not redesign anything. The solver is not in question.

## Defect 1 — eight passing fixtures are missing

Upstream has 2656 flex fixtures at the pinned commit. The port has 2500.

    ls _upstream/taffy/tests/xml/flex/ | sort > /tmp/up.txt
    ls layout/testdata/flex/          | sort > /tmp/got.txt
    comm -23 /tmp/up.txt /tmp/got.txt

Of the 156 absent, 148 are correctly excluded. Eight are not:

    aspect_ratio_flex_column_fill_width_flex__{border,content}_box_{ltr,rtl}.xml
    aspect_ratio_flex_row_fill_width_flex__{border,content}_box_{ltr,rtl}.xml

Copy them in and they pass — I checked. They were dropped for no reason, and they
cover aspect ratio interacting with flex fill, which is not a corner worth losing.
Restore them.

## Defect 2 — the exclusions are invisible

The suite reports 2500 of 2500 passing. Nothing in `doc.go`, the harness or
`testdata/` says that 156 upstream fixtures are absent or why, so the run reads as
complete coverage to anyone who has not diffed the two directories.

Write the record. `layout/testdata/flex/README` is the natural home, beside the
files it describes. It needs:

- Where the fixtures came from: `tests/xml/flex/` in
  [DioxusLabs/taffy](https://github.com/DioxusLabs/taffy), at the commit pinned in
  `upstream.pins` — currently `b3b387132be1dda0e9d08d5044692236532c166d`. Name the
  commit explicitly, so a reader can tell whether the fixtures have drifted from the
  pin.
- What is excluded and why, with counts:

      136  balance_*                        feature-gated upstream, not in this port
        4  bevy_issue_10343_grid__*         grid
        4  bevy_issue_21240__*              grid, despite living in tests/xml/flex/
        4  overflow_start_edge_absolute_child__*
                                            absolute positioning inside a block
                                            container, beyond the flexbox port

- That these are MIT-licensed files from Taffy, carried under the entry in `NOTICE`.

Your judgement on those 148 was right, including `bevy_issue_21240`, which is a grid
test misfiled upstream. The reasoning just never made it into the repository.

## Then keep it honest as the pin moves

Add a test that fails when the fixture directory drifts from upstream: compare the
names in `layout/testdata/flex/` against `tests/xml/flex/` in the checkout, minus
the documented exclusion list, and fail on anything unaccounted for in either
direction. Skip it when `_upstream/taffy` is not present, so it does not break a
checkout that has not run `go run ./tools/upstream`.

That turns the exclusion list from a comment into something enforced, and means the
next `-update` of the Taffy pin surfaces new fixtures instead of silently ignoring
them.

## Done when

    go build -o bin/ ./...
    go test ./layout/
    go test ./internal/layering
    go vet ./layout/
    gofmt -l $(go list -f '{{.Dir}}' ./...)

2508 fixtures pass. The exclusion record exists, names the pinned commit, and is
enforced by the drift test.

One conventional commit per file, as before.

## Out of scope, still

Grid. Float. The balance module. A complete block layout — `block.go` stays the
minimal implementation the flexbox fixtures need. Anything in another package.

## What to carry forward

The report said "all 2501 included fixtures passed". The word doing the work there
was *included*, and it hid 156 absent fixtures behind a number that looked total. A
count of what passed is only meaningful next to a count of what was run and what was
skipped. When coverage is bounded, say what was dropped — a silent cap reads as full
coverage to everyone who comes after.
