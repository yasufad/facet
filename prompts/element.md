# element: review of the first milestone

Everything asked for is here and correct. `NewDiv()` resolves the name collision and
`doc.go` says why. `RemSize` is on `Frame`. The phases enforce their order. The view
erasure works. The fake frame records what reached it and the tests assert exact
values — quad backgrounds, origins and sizes, not that a quad exists.

And you benchmarked the thing that mattered. The number is the finding.

## The tree costs more than everything below it combined

    Div                        584 bytes
    NewDiv()                   420 ns    640 B    1 alloc
    eleven-node tree          5300 ns   7536 B   16 allocs

`Div` embeds `style.Refinement` by value, and that is 504 bytes since the property
list landed. So every element in the tree is a separate 640-byte heap allocation
costing ~420 ns, and eleven of them account for 7,040 of the 7,536 bytes. Preallocating
the children slice changes nothing — I measured that too.

At a thousand elements a frame:

    ~420 µs      per frame, construction alone
    ~640 KB      garbage per frame
    ~38 MB/s     allocation rate at 60 fps

The wall-clock is survivable — 420 µs against a 16 ms budget, on top of `style`'s
~400 µs. The allocation rate is the problem, and it does not show up as a slower
average. It shows up as GC pauses landing inside frames, which is a stutter, and
stutter is the thing a UI framework cannot have.

This is the measured price of "elements are values built fresh each frame". That
sentence has been in `docs/packages.md` since before anything existed, and until now
nobody knew what it cost. Recording it is worth as much as the code.

## What to decide now, and what not to

**Do not restructure `Div` yet.** Splitting the refinement out of line trades one
allocation for two on any styled element, which is most of them. It is not the fix.

GPUI's fix is a per-frame arena — `window.rs` enters an `ElementArenaScope` in `draw`
and every element allocates from it, with the whole thing reset at frame end.
Allocation becomes a bump and a pointer, and there is no garbage. That arena belongs
to `window`, which owns the frame lifecycle, so it is not yours to build and I have
put it in `prompts/window.md`.

**But the constructor signature is yours, and it decides whether an arena is possible
at all.** `NewDiv()` takes no arguments, so it cannot reach an arena — there is
nowhere for one to come from. Once an arena exists the options are:

- `f.NewDiv()` on `Frame`, or a similar carrier — explicit, and changes every call
  site users write
- a package-level "current arena" set by `window` for the duration of a frame —
  keeps `NewDiv()` as it is, works because the UI is single-goroutine, and is the
  kind of hidden global that is hard to remove later
- accept the GC cost and keep `NewDiv()` as it is

Pick one now and say so in `doc.go`, with the numbers above as the reason. Changing
it after `ui` is written means changing every widget and every example. This is
exactly the sort of decision that is free today and expensive in a fortnight.

I lean towards the second, because `div().Flex()` is the expression the framework
exists to make read well and threading a frame through it damages that — but the
hidden global is a real cost and I would rather you weigh it than take my word.

## Smaller

Sixteen allocations for eleven nodes means five are not elements. Worth knowing what
they are before the arena lands, because the arena will not absorb them.

`ShapeLine(str string, ...)` — good, the shadowing is gone.

## Done when

The construction numbers are in `doc.go`, labelled as the per-frame floor they are,
next to `style`'s.

The constructor decision is made and written down with its reasoning.

Then the second milestone: the rest of `div`, and interactivity — `input` is
available to you, and `Interactivity` is most of GPUI's 5,200-line `div.rs`, so scope
it before starting.

## Worth carrying

You measured the constructor because the prompt asked for a number, and the number
turned out to be the most important thing anyone learned this week. The benchmark
that tells you something you did not already believe is the one worth writing.
