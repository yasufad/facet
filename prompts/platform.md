# platform: window has reported, so take the second half

All three items verified. The examples and `platform` cross-compile clean for Linux and
macOS, `SetCursor` was added to `platform.Window` without touching `Platform` as
instructed, and the rune conversion is right — I broke the low-surrogate test in
`runeOffsetFromUTF16` and `TestRuneOffsetFromUTF16` failed on the astral-plane case with
exact values, which is the assertion doing real work.

The layering of the IME tests is the part worth naming. The conversion has an exact unit
test; the window-level test drives the message plumbing and says in its own comment that
it asserts a sane cursor rather than IMM32's behaviour, because no real IME is attached.
That is a bounded coverage claim recorded beside the code, which is what `AGENTS.md` asks
for and what usually gets skipped. Relaxing the assertion when the real behaviour
contradicted your guess — rather than changing the code to match the guess — was the right
call in the right direction.

## 1. SetCursor, second half — unblocked now

`window` has landed and reported. `82ab7e5` rewrote `resolveNextHitTest` to return the
whole hit region, and the function has settled; nothing else is queued against it. The
contention I told you to wait for is gone.

So take it: remove `SetCursor` from `Platform`, switch `window.resolveNextHitTest` to the
window method, and update `Platform`'s doc comment — which still promises a cursor over
the active window and stops being true when the method moves. One commit, because `window`
is the only caller and splitting it leaves `main` unbuildable.

This is the stated exception to staying inside your package. Commit by path
(`git commit -m "..." -- <file> <file>`), because the index is shared and three agents have
had another's staged file swept into their commit.

That the failure to tell you window had reported was mine. I said it in `window`'s prompt
and in my report to Yasu and never wrote it here, which is exactly the kind of gap that
leaves work parked for a round.

## 2. docs/packages.md — mine, and I am waiting on you

You were right to leave it alone and right to offer. I will write the `SetCursor` split
into `platform`'s entry when the second half lands, because the reason the split falls
where it does is only true once it has fallen — recording it now would describe an
intention rather than the code, which is the failure mode the audit found throughout this
repository.

I will record the IME production at the same time. If you think the entry should say
something you know and I do not — particularly about what `ImmGetCompositionStringW`'s
length semantics cost you — put it in your report and I will use your words.

## 3. Then macOS

After the above, and only after reporting it.

The first deliverable is a window that opens, reports a real `NSWindow*`, and delivers
pointer and key events. No renderer, no drawing. That is enough to prove the purego path
and to answer the question everything else rests on: whether `objc_msgSend` with struct
returns is reachable without cgo.

If it is not, that is a decision to record in `docs/architecture.md`, and the sooner it is
recorded the less is built on the assumption. Bring me the answer before building on
either branch of it.

Do not start this inside the current round.

## Not yours, for the record

`tools/compile_shaders` is now the only thing stopping `go build ./...` on Linux and
macOS — one missing `//go:build windows`. It belongs to `prompts/build.md` and it is
assigned. Leave it; I would rather it landed with the CI that proves it.

## Done when

    go doc ./platform

shows `SetCursor` on `Window` and not on `Platform`, `Platform`'s doc comment no longer
promises something it does not do, and the tree builds with `window` calling the window
method.

Report before starting anything in the macOS section.
