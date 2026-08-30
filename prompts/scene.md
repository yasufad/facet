# Assignment: scene

The package is built and committed — fifteen files, one per commit, staged by path,
and every check clean. The draw-order engine is the right call: an R-tree that
returns an order one above anything the new bounds intersect gives correct occlusion
and lets non-overlapping primitives share an order so the renderer can batch them.
That is the substance of this package and it is done.

What is missing is the attribution for it.

## bounds_tree.go is a port and is not attributed as one

Your own doc comment says so, and the code agrees: `MAX_CHILDREN = 12` became
`maxChildren = 12`, `find_max_ordering` became `findMaxOrdering`, `insert_leaf`
became `insertLeaf`. That is a port of `crates/gpui/src/bounds_tree.rs`, and porting
it was the right decision — rewriting an R-tree from scratch to avoid a port would
have been worse.

`AGENTS.md` asks a ported file to carry a header naming the upstream project, the
file it came from, and its licence, and asks the project to appear in `NOTICE`.
Neither is there, and Zed appears nowhere in `NOTICE`.

Three things:

- Put the attribution at the top of `bounds_tree.go`, above `package scene`, naming
  `crates/gpui/src/bounds_tree.rs` and Apache-2.0.
- Add Zed to `NOTICE`, with the notice text copied out of
  `_upstream/gpui/LICENSE-APACHE`. Copy it; do not write it from memory. Two
  attributions in this repository have already been wrong, and both read plausibly.
- Apache-2.0 asks a modified file to say it was modified. Your note that the Rust
  original uses raw pointers for its search stack while the port uses slice indices
  already does that — move it into the header so it reads as the change notice it is.

`docs/sources.md` used to say GPUI was read-only, which would have made this port a
policy breach. It was the policy that was wrong: `crates/gpui` is Apache-2.0 and
portable with attribution. It now says so, and carries a licence map for the Zed
checkout — worth reading, because that repository is two licences and one crate in
it was GPL.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

`bounds_tree.go` carries its attribution and change notice, and `NOTICE` names Zed
with text copied from its licence file.

`NOTICE` is shared and other agents are in this tree. Stage it by path and check
`git show --name-only` afterwards.

## Worth carrying

You wrote "port of GPUI's BoundsTree" in the doc comment and "port the R-tree
draw-order engine from GPUI" in the commit subject. That honesty is the only reason
this was caught — an agent describing the same work as "add an R-tree" would have
buried it, and nobody reviewing a Go file diff would have known to look for a Rust
original. Keep saying plainly where code came from.
