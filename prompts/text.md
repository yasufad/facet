# text: the cache memoises the shaping and redoes everything around it

Nothing above or below you moves for any of this, so it runs in parallel with the rest.

## 1. ShapeLine costs 37 µs on a cache hit

The measurement, from `element`'s benchmarks driving `Frame.ShapeLine`:

    BenchmarkTextMeasureSameWidth      6.13 ns      0 B/op    0 allocs/op
    BenchmarkTextMeasureVaryingWidth  37.52 µs  32075 B/op   92 allocs/op

The first number is `element` short-circuiting on its own cache. The second is what
`ShapeLine` actually costs, and it costs that even when `shapeCache` hits, because the
cache sits around the HarfBuzz call and nothing else:

```go
func (s *System) wrap(text string, runs []StyleRun, maxWidth fixed.Int26_6) ([]ShapedLine, error) {
    paragraph := []rune(text)
    runeToByte := make([]int, len(paragraph)+1)
    outs := make([]shaping.Output, 0, len(runs))
    ...
```

Three allocations before any shaping happens, then segmentation, then the line wrapper,
then building a `ShapedLine`. All of it repeats on every call. `shapeCache` saves the
shaping and nothing around it.

Cache the `ShapedLine`, not just the `shaping.Output`. Key it on the text and the runs —
the same things `cacheKey` already captures, one level up. A repeat `ShapeLine` for the
same string and style should be a map lookup and a copy, not a re-segmentation.

`element` will separately stop asking for a re-shape it does not need; that is on its
prompt. Both are worth doing, because a cheap call being made too often and an expensive
call being cheap are different problems and fixing one hides the other.

Say in the package doc what the cache is now keyed at, and what invalidates it.
`docs/packages.md` currently says "shaped output is cached by run, not by string", which
stops being the whole truth when there are two levels.

## 2. The cache key allocates on every lookup

```go
func (in shapeInput) key() cacheKey {
    return cacheKey{
        ...
        language: string(in.language),
        features: featuresKey(in.features),
    }
}
```

`string(in.language)` converts on every call, hit or miss. `featuresKey` builds a byte
slice and converts it. Both run before the map is touched, so a hit pays for them too.

Intern the language and the feature list once, when the run is resolved, and carry the
interned value on `shapeInput`. The key then becomes a comparison of values that already
exist.

## 3. Both caches only grow

`shapeCache.entries` accumulates an entry per distinct string per face per size for the
life of the process. `Atlas.entries` does the same for coverage masks, and `Clear`
replaces the map wholesale rather than evicting anything.

For a demo this is invisible. For an editor open across a day, with theme changes, font
size changes and several scripts, it is a slow leak by construction.

Put a ceiling on both, expressed in bytes rather than entries, with least-recently-used
eviction. Bytes because a mask for a 10 px glyph and one for a 96 px heading differ by two
orders of magnitude, and an entry count that is right for one is wrong for the other.

Make the ceiling settable and give it a default you can defend with a number: measure the
steady-state footprint of shaping a realistic document and pick from that, rather than
picking a round number and hoping.

Eviction has one hazard worth stating. `window` caches GPU tiles keyed by the same face,
glyph, size and subpixel offset, and holds them independently. Evicting a CPU mask must
not invalidate a GPU tile that is still correct — the two caches hold different things and
are allowed to disagree. If you find they cannot, that is a finding and the GPU side is
`render`'s and `window`'s to fix, not yours.

## Not in scope

Multi-line text at the element level — wrapping, bidi across lines, selection geometry,
mixed style runs within a paragraph — is `element` work, not yours. `WrapText` already
exists here and handles what it needs to.

## Done when

    go test ./text
    go test -tags facet_debug ./text

pass, with a benchmark showing the cost of a repeated `ShapeLine` for the same string, and
a test that fills a cache past its ceiling and asserts the oldest entry went and a recent
one stayed. Break the eviction and confirm that test fails; an LRU that never evicts
passes every test that only checks lookups.

Report the before and after numbers for `ShapeLine`, and the ceiling you chose with the
measurement behind it.
