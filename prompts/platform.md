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

## The macOS gate, still shut

I said the gate is `examples/button` running and `internal/integration` passing. Neither
has happened: `internal/integration` does not exist, and `examples/button` still registers
a raw `func(element.ClickEvent) bool`, which is the signature the audit opens with. `ui`
holds both and is not blocked on anything now.

The gate has not moved and I am not moving it. When `ui` reports, macOS is yours.

## Done when

`ShowSaveDialog` is implemented and its flags tested against a real `CoCreateInstance`
round trip, with the vtable cross-checked against two shipping implementations and the
sources named in the commit.

The menu interface shape is proposed, not built.
