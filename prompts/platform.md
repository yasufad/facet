# platform: the save dialog, and the gate is still shut

Window state is verified. I broke `stillFullscreen` to trust the flag blindly and
`TestWindowStateFullscreenFlagClearsOnMismatch` failed while the round-trip test correctly
kept passing — two tests that fail for different reasons, which is what the pair was for.

## The WPF_RESTORETOMAXIMIZED correction is mine

I told you Win32 models current-state-plus-restore-target through that flag and that you
should follow it. You checked the SDK, and it says the flag forces a restore to maximised
"regardless of whether it was maximized before it was minimized" — an override of the
default behaviour, not the thing producing it. My model was wrong.

Yours is better than either of the ones we started with. Minimise leaves
`rcNormalPosition` alone, so a fullscreen window's monitor-bounds rect survives with no
bookkeeping at all, and restore reproduces it even through a raw `ShowWindow` that never
went near `SetState`. My conclusion happened to be right; the mechanism I gave you for it
was invented. Building the simpler thing rather than binding a flag to match my
description was the right call, and retracting your own browser claim in the same report
rather than quietly dropping it is the standard this works at.

Both are recorded in `docs/packages.md`, the correction included, so the next person does
not re-derive it from my version.

## Recording the vtable method: yes

You asked whether it was worth a line. It was worth a paragraph, and it is in `platform`'s
entry now.

Microsoft Learn lists COM methods alphabetically, a vtable is ordinal, and a wrong slot is
a legal call to the wrong method rather than a crash — which is the same failure class as
the three D3D11 vtable indices that reached reviewed, green code. Cross-checking SDL and
Wine's IDL, then testing a real `CoCreateInstance` round trip anyway because two sources
agreeing is not the same as the call working, is a method that generalises to the Cocoa
and Vulkan work ahead. That is why it is written down rather than left in a commit
message.

Splitting `newFileOpenDialog` from `ShowOpenDialog` so the configuration is testable
without a human clicking a modal is the other thing worth naming. The untestable part is
now exactly one call wide.

## 1. The save dialog

Same shape, `IFileSaveDialog`. Verify its vtable the same way — it extends `IFileDialog`,
so the inherited slots come first and the additions follow, and that ordering is precisely
what an alphabetised page will not tell you.

Overwrite confirmation, a default filename and a default extension are the three things a
save dialog has that an open dialog does not, and each is a flag or a call that a wrong
constant would break silently. Test them the way you tested the open flags.

## 2. Then menus, not tray or notifications

You had menus waiting on window state. Window state has landed, so they are unblocked, and
they matter more than tray or notifications for the same reason the open dialog did: an
application with a custom title bar has no menu bar unless we give it one, and "Open
File…" now has something to call.

Raise the interface shape before implementing it. A menu crosses into `window` eventually
and I would rather decide that before there is code shaped around the wrong answer.

## The macOS gate is open

I said the gate was `examples/button` running and `internal/integration` passing, and that
when `ui` reported, macOS was yours. `ui` has reported and both conditions are met.

`internal/integration` exists and holds
`TestButtonClickInWindowMutatesEntityAndRendersNextFrame` — a real `ui.Button` clicked in
a real window, mutating entity state, with the next frame rendering it. I broke
`element.Listener` and it failed, so it is a test rather than a shape. `examples/button`
binds through `element.Listener` and builds. The whole tree is green: build, vet, gofmt
and every test.

So macOS is next after the save dialog, and the gate does not come back.

The first deliverable is unchanged and deliberately small: a window that opens, reports a
real `NSWindow*`, and delivers pointer and key events. No renderer, no drawing. That is
enough to answer the question everything else rests on — whether `objc_msgSend` with
struct returns is reachable through purego without cgo.

If it is not, that is a decision to record in `docs/architecture.md`, and the sooner it is
recorded the less is built on the assumption. Bring me that answer before building on
either branch of it.

Finish the save dialog first. It is a stub returning "not implemented" and it is the last
thing keeping the Windows backend from honestly matching what the README claims.

## Done when

`ShowSaveDialog` is implemented and its flags tested against a real `CoCreateInstance`
round trip, with the vtable cross-checked against two shipping implementations and the
sources named in the commit.

The menu interface shape is proposed, not built.
