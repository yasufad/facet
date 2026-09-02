# text: the line cache hands out its own memory

The measurement is real and the design is right. I reproduced the numbers, and 111 µs to
585 ns for a repeated `ShapeLine` is the largest single win landed in this repository so
far. Two things have to change before it goes in, and the first is serious.

I also broke your LRU to check the test could fail: removing the promotion from `Get`
turns it FIFO, and `TestByteLRUEvictsOldest` fails on the right line with the right
message. That is the standard, and it is the first eviction test here that meets it.
`byteLRU` is correct on the case I expected to catch it, too — a value larger than the
whole ceiling evicts itself rather than sitting over budget.

## 1. A cached line is aliased, and callers can corrupt it

`wrap` returns the cache's own slice:

```go
if lines, ok := s.lineCache.lru.Get(key); ok {
    return lines, nil
}
```

`ShapedLine.Runs()` hands back `[]ShapedRun`, `ShapedRun.Glyphs` is exported and mutable,
and every caller of `ShapeLine` for the same text now shares one backing array. Your
report says "a map lookup and a slice copy". There is no copy.

Reproduced through the exported API only — no test hooks, nothing internal:

```go
first, _ := sys.ShapeLine(s, run(s))
before := first.Runs()[0].Glyphs[0].Position
first.Runs()[0].Glyphs[0].Position = geometry.Point[geometry.Pixels]{X: 999}

second, _ := sys.ShapeLine(s, run(s))
// second.Runs()[0].Glyphs[0].Position is {X:999 Y:0}, want {X:0 Y:14.484375}
```

The part that decides it: **that probe passes at HEAD and fails with your change.** Before
the line cache, `wrap` built a fresh result every call and a caller mutating a glyph hurt
nobody. The cache introduces the hazard, no test in the package can see it, and the
consumer is `element`, which walks glyphs every frame to emit sprites and is one
subpixel-snapping adjustment away from tripping it.

Decide which of two contracts you are offering and enforce the one you pick:

Copy on the way out. Correct, obvious, and it puts an allocation back on the hot path —
though a copy of a small glyph slice is nothing beside the 111 µs it replaces. Measure it
rather than assuming; if a line-cache hit is still under a microsecond, this is the answer.

Or make the result genuinely immutable and say so. `Runs()` returning a shared slice is
already a loaded gun, cache or no cache, and unexporting `Glyphs` behind an accessor is a
larger change than this round — it reaches `element`. Do not start it here.

Whichever you choose, the test is the probe above, in `text`, using only exported API. It
must fail if the protection is removed.

## 2. The key allocates on every lookup, which is the bug you just fixed

You did exactly the right thing to `shapeInput.key()` and then rebuilt it one level up:

```go
key := lineCacheKey{text: text, runs: styleRunsKey(runs), maxWidth: maxWidth}
if lines, ok := s.lineCache.lru.Get(key); ok {
```

`styleRunsKey` builds a `[]byte` and converts it to a string before the map is touched, so
a hit pays for it. That is all three of the allocations and all 272 bytes your own
benchmark reports on the hit path — the return does not copy, so nothing else is left to
account for them.

`ShapeLine` is called per text element per frame. Fifty text elements is fifty key builds
per frame that could be none.

The prompt's instruction was to intern once when the run is resolved and carry the value.
Same answer here: the caller's `[]StyleRun` is stable across frames, so the key is
computable when the runs are, not when they are looked up. If interning at that boundary
turns out to need something from `element`, that is a finding worth reporting rather than
working around — say what you need and I will decide it.

Target zero allocations on a line-cache hit, and put the number in the report.

## 3. Commit the work

Nothing is committed. `git log` at HEAD is my last commit; every file you describe is
untracked or modified in the shared working tree.

That is not a detail. Review here is a verdict on what a package committed — I had to
review `text/` in place and could only do it because nothing else contends that directory.
It also means your work is one `git checkout` by another agent from being gone, in a tree
where four other agents are working.

Conventional commits, one file per commit, staged by path. `git add text/lru.go`, never
`git add -A`.

You were right to revert `docs/packages.md` and leave `prompts/text.md` alone. Both are
mine, and I will write the entry when this lands.

## 4. Small

`SetMaxBytes(0)` means "hold nothing", not "unbounded" — `evict` runs while
`size > maxBytes`. That is a defensible choice and an easy one to get wrong from the
outside. Say it on all three setters.

## Done when

    go test ./text
    go test -tags facet_debug ./text

pass, with the aliasing probe in the package and failing when the protection is removed,
and with a line-cache hit at zero allocations.

Both are committed, one file per commit.

Report the hit-path numbers again. The 190x stands; I want to see what the copy costs and
what the key stops costing.
