# platform: the menu decision, then macOS

> **Onboarding.** You own `platform` and nothing else. If a change you need reaches into
> another package's exported API, stop and say so rather than editing it.
>
> `AGENTS.md` loads automatically and is the standard you are held to. The two sections
> that catch people out are **Commits** — one file per commit, committed by path as
> `git commit -m "..." -- <file>`, because the index is shared and staging then
> committing has put three agents' work under someone else's subject line — and
> **Working alongside other agents**, which is a list of incidents rather than advice.
> Read your entry in `docs/packages.md` before you start: it says what you may import
> and what you have to keep true.
>
> Other agents are working in this same checkout right now. `go build ./...` here
> reports their half-finished files alongside yours, which has misled a review three
> times, so verify in a worktree of your own before claiming anything is green:
>
>     git worktree add --detach <scratch>/check HEAD
>
> Do not push to origin — commit locally, the lead pushes in batches.
>
> When you are done, write `work/platform-01.md` the way **AGENTS.md → Reporting** describes.
> That file is what gets reviewed. A claim with no break test under it is read as
> unverified.


The save dialog is verified. I replaced the flags at `platform_windows.go:410` with
`FOS_FILEMUSTEXIST` in place of `FOS_OVERWRITEPROMPT` and both tests failed on both
counts — the save flag missing and the open flag present — including
`TestNewFileSaveDialogNoFilters`, which is the one that matters, because a zero-value
`SaveFileDialog{}` is where two earlier defects in this package survived five tests.

Setting `Directory` and asserting on the FOS bits is the pattern working as intended:
the assertion cannot pass because of the action that preceded it.

## One gap, and one thing that cannot be closed

`SetFileName` is called and never read back. `GetFileName` is declared in both vtable
structs — `shobjidl.go:107` and `:291` — and has no Go method, so nothing proves that
slot is the one being called. It is the same silent-failure class the whole vtable
cross-check exists for: a wrong ordinal is a legal call to the wrong method. Bind
`GetFileName` and round-trip `DefaultName` in `TestNewFileSaveDialogConfigures`.

`SetDefaultExtension` cannot be verified that way — `IFileDialog` has no getter for it.
Say so in a comment beside the call rather than leaving the asymmetry to be noticed.
A count of what is covered with no count of what is not reads as full coverage.

## The menu decision

All three, decided. Replace the proposal comment in `menu.go` with what survives.

### 1. Native menu bars, and `Window` does not change

The proposal pairs native with `Window.SetMenu` and rendered with no new method. That
pair is false. `windowsPlatform` already holds `windows map[w32.HWND]*windowsWindow`,
populated at `platform_windows.go:555` and cleared at `:564`. `SetApplicationMenu`
walks it and calls `SetMenu(hwnd, hmenu)` on each; `registerWindow` attaches the
current menu to windows created afterwards. That is a method on the unexported
`windowsWindow`, not on the `Window` interface.

Keeping it off the interface is the point. macOS cannot honour a per-window menu, so
`Window.SetMenu` would mean something on one backend and nothing on the other — the
`Div.Opacity` failure, which I have already caused once in this tree. `SetApplicationMenu`
is also what an application actually wants to call: one menu, set once.

The escape clause in its doc comment goes with this. "or is a no-op where window menus
are set per-window" is the sentence that let the Windows implementation be a TODO. It
is real on both backends now, or the method is a lie.

Undecorated windows get no menu bar, and that is correct rather than a limitation: a
window created with `Decorated: false` asked for no OS chrome, and a menu bar is OS
chrome. A rendered menu bar for custom-title-bar applications is a `ui` widget, later,
and it needs the mechanism the next section says does not exist.

### 2. Context menus: native, and asynchronous

    Window.ShowContextMenu(menu *Menu, at geometry.Point[geometry.Pixels])

Native rather than rendered because a rendered popup cannot leave the window, and a
context menu opened near the bottom edge has nowhere to go — a visible defect on an
ordinary case. Rendering one also needs a way for an element to escape its parent's
bounds and paint above later siblings. `scene` has `PushLayer`/`PopLayer`, but nothing
on `element.Frame` reaches them, so that mechanism is not built. Native costs less and
is better.

The part to get right is that it must not block. `TrackPopupMenu` runs a nested modal
message loop and does not return until the user dismisses the menu. Called from inside
an event handler, that handler is inside `app.UpdateEntity` with the entity checked
out, and the nested loop will pump further messages into the frame loop while it is
borrowed. So **`ShowContextMenu` records the request and returns immediately**, and the
backend runs `TrackPopupMenu` on a later turn of its own message loop, once the current
update has finished. `MenuItem.OnClick` fires on the platform thread, outside any borrow.

macOS is the same shape — `popUpMenuPositioningItem:` also runs a nested tracking loop —
so this is the contract, not a Windows workaround. Put it in the method's doc comment:
it returns before the menu appears, and `OnClick` fires later.

#### The ordering, and I got it wrong the first time

I wrote that the interface line and `platform_other.go` land together. `platform.Window`
has four implementers, not two. Two of them are outside this package:
`stubPlatformWindow` in `window/window_test.go:63` and another in
`internal/integration/button_click_test.go`, twenty-four methods each, both passed to
`window.NewWithRenderer(pw platform.Window, ...)`. A twenty-fifth method on the interface
stops two packages you do not own from compiling.

That is the deletion incident inverted, and I nearly caused it again in a prompt that
cites the deletion incident.

**So the fix is not sequencing, it is `platform/platformtest`.** Ship an exported
`Window` double there, the way `element/elementtest` already serves `ui` — that package
is the precedent and `docs/packages.md` blesses it. It adds nothing and breaks nothing,
so it can land now, on its own.

Then `window` and whoever holds `internal/integration` drop their private stubs onto it.
That is their commit, not yours; raise it with me and I will put it in their prompts.
Only once both are migrated does `platform.Window` gain `ShowContextMenu`, and after that
it can grow again without a three-party handshake, which is the actual point — this
interface is going to keep growing, and the next method should not cost a round of
coordination.

Within this package, the order is unchanged: `windowsWindow` gets the method first, where
it satisfies nothing and breaks nothing, then the interface line and `platform_other.go`
together.

**Do the save dialog, `platformtest` and `SetApplicationMenu` now.** `SetApplicationMenu`
is already on the interface, so the menu bar costs no coordination at all.
`ShowContextMenu` waits for the migration.


### 3. Shortcuts go through `input`'s keymap. `Shortcut` is display text.

`RegisterHotKey` is out, and not narrowly. It registers system-globally: the chord
fires whenever it is pressed anywhere on the desktop, whichever application has focus.
For Ctrl+S that is not a near miss.

`input` already has the whole mechanism — `Keymap` with context predicates,
`ParseKeySequence` for chords, `Action` resolved with precedence along the focus path,
`NoAction` and `Unbind` for suppression. `RegisterHotKey` has none of them, so two
systems would disagree with no rule to appeal to. `input.Action`'s own doc comment
reads "dispatched in response to keybindings, menu selections or command palettes".
This was anticipated where it belongs.

So `MenuItem.Shortcut` is the text drawn beside the label, and the promise in its doc
comment — "where the platform supports it, registered as a global shortcut for the
item" — is withdrawn. The application binds the chord in its keymap; the menu item's
`OnClick` dispatches the same action. Both routes arrive at one place.

`platform` cannot hold an `input.Action`: `input` imports `platform` and the dependency
runs one way only. `OnClick func()` stays, and its doc should say what it is for — a
thin closure that dispatches an action, not the logic itself.

One consequence, worth writing down now rather than meeting it on macOS. An
`NSMenuItem` key equivalent fires natively once the item is installed, and consumes the
keystroke before it reaches our handler. A chord present in both the menu and the
keymap is therefore handled by the menu on macOS and never reaches the keymap. That is
harmless only because both dispatch the same action, which is why "same action on both
routes" above is a contract and not a preference.

## Then macOS

Unchanged, and it is the last thing in front of you. The first deliverable is
deliberately small: a window that opens, reports a real `NSWindow*`, and delivers
pointer and key events. No renderer, no drawing.

That is enough to answer the question everything else rests on — whether `objc_msgSend`
with struct returns is reachable through purego without cgo. If it is not, that is a
decision for `docs/architecture.md`, and the sooner it is recorded the less is built on
the assumption. Bring me that answer before building on either branch of it.

## Done when

`GetFileName` is bound and `DefaultName` round-trips; the `SetDefaultExtension` gap is
recorded beside the call.

`menu.go`'s proposal comment is replaced by the three decisions, stated as contracts
rather than as a record of the choice.

`SetApplicationMenu` attaches a real `HMENU` to every window the Windows backend owns,
present and future, and its doc comment no longer contains the words "no-op".

`platform/platformtest` exports a `Window` double, and you have told me it is there so I
can put the migration in `window`'s and `ui`'s prompts.

`ShowContextMenu` is **not** in this round. It lands after both stubs have moved, and its
own condition then is that it returns before the menu is shown, with a test that a
selection reaches `OnClick` with no entity borrow live — or, if that needs a human
clicking, the untestable part is exactly one call wide, the way you split
`newFileSaveDialog` out of `ShowSaveDialog`.
