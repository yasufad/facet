> **Read before starting.** You are picking up one package in a framework several agents
> build in parallel. `AGENTS.md` loads automatically and is the standard you are held to —
> read it, and read your package's entry in `docs/packages.md`, before you touch code. This
> file is only the work outstanding; it does not restate what those two say.
>
> The four things that cause the most damage here, in order:
>
> 1. **Commit one file per commit, by path:** `git commit -m "..." -- path/to/file.go`.
>    Never `git add -A`, never `git add .`, never `git commit -a`. The git index is shared
>    with other agents — staging then committing lets another agent's file land in your
>    commit. Three agents have had that happen. Check `git show --name-only` after every
>    commit; it should name exactly the file its message describes. The exception is a
>    change that is not true in pieces (a rename, or code plus the comment stating its
>    guarantee) — say so in the message when you use it.
> 2. **Stay in your package.** If your change needs another package's exported API to move,
>    stop and report it. Do not edit across the boundary unless this file explicitly says
>    you may.
> 3. **Verify at HEAD, not in the working tree.** Other agents have uncommitted files here,
>    so `go build ./...` in this checkout reports their half-finished work as yours. Use
>    `git worktree add --detach <scratch>/check HEAD`, run the commands there, remove it.
> 4. **Break your fix and confirm the test fails.** A test that cannot fail is not a test,
>    and every package here has passed its own tests while getting something wrong. Watch
>    for the break silently not applying — check `git diff --stat` before believing a green
>    run, and for an unused variable turning your break into a build error rather than a
>    failure.
>
> Report back when done rather than expanding scope. Do not edit `docs/`, `README.md`,
> `AGENTS.md` or `prompts/` — those belong to the reviewing agent, who writes up what your
> package guarantees once it lands.

# text: confirm the hash, and one number to keep honest

Both fixes verified. I re-ran my original aliasing probe at HEAD and it passes, then
removed the clone and `TestLineCacheHitDoesNotAliasCallerMutation` failed.

Cloning on **both** return paths is the half most people miss, and you found the reason
rather than being told it: the miss path stores the same slice it returns, so returning it
unmodified lets that call's own caller corrupt the entry the next call reads. The comment
saying so is doing real work.

Reporting 40x instead of the 190x you had was right. The larger number described a cache
that handed out its own memory; the smaller one describes a correct hit. A benchmark that
measures a bug is not a measurement.

## 1. Confirm the hash on a hit

You flagged this rather than shipping it quietly, which is why I get to decide it instead
of finding it in six months. My answer is to close it.

`lineCacheKey` holds `text`, `maxWidth` and a 64-bit `runsHash`. Two run sets that collide
share an entry and one of them silently renders with the other's shaping. Your comment
states the odds honestly and the odds are negligible — a collision needs the same string
at the same width with different runs whose digests coincide.

I am ruling on consequence, not likelihood. Wrong glyphs from a cache are unfalsifiable in
the field: nothing errors, nothing looks obviously broken, and no test can catch it because
it depends on inputs nobody has. This repository has already been bitten twice by a legal
operation producing plausible output — three wrong D3D11 vtable indices, and a batch offset
that drew the previous batch's instances. Both passed review. Neither reported an error.

**Keep the hash for the map key and confirm it on a hit.** Store the runs the entry was
built from, and when the digest matches, compare them field by field before returning.
The hash keeps the lookup allocation-free; the comparison makes the guarantee exact again.

Two details worth getting right. The stored runs have to be cloned, for the same reason
the shaped lines do — `StyleRun` carries `Families` and `Features` slices, and a caller
who mutates the slice they passed would otherwise corrupt the key of an entry already in
the cache. And the comparison has to walk those slices; `==` will not compile on
`StyleRun` and would be wrong if it did.

That is one clone per insert, on the miss path where you are already shaping, and one
comparison per hit. `element` landed the same shape for the same reason this week —
comparing resolved `[]text.StyleRun` rather than trusting a derived value — so the pattern
is in the tree and the convergence is not a coincidence.

Keep `TestHashRunsDistinguishesFields`. Add one that puts two entries in with a forced
digest collision and asserts the second does not return the first's lines.

## 2. The number to keep honest

    Line-cache hit    2,258 ns/op    1,552 B/op    3 allocs/op

Those three allocations are the clone, and they are the price of correctness with
`ShapedRun.Glyphs` exported. Do not chase them to zero inside `text` — the only way to
zero is to stop handing out mutable glyph slices, which unexports `Glyphs` behind an
accessor and reaches into `element`. That is a real change and it is not yours to start.

I have queued it. When it happens it removes the clone from every hit rather than
optimising around it, and this benchmark is what will prove it.

Say in the package doc that a returned `[]ShapedLine` is the caller's own copy. That is
now a guarantee the package makes, and it is the reason a hit costs what it costs.

## Done when

    go test ./text
    go test -tags facet_debug ./text

pass, with the collision test failing if the confirmation is removed, and the hit-path
number reported again with the comparison in place.
