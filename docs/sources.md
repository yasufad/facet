# Upstream sources

Three upstream codebases inform this one, each in a different way.

| Project | Where | What we do with it |
| --- | --- | --- |
| GPUI | [zed-industries/zed](https://github.com/zed-industries/zed), `crates/gpui` | read it; port a self-contained algorithm where that beats rewriting |
| Taffy | [DioxusLabs/taffy](https://github.com/DioxusLabs/taffy) | port it, algorithm for algorithm |
| Wails v3 | [wailsapp/wails](https://github.com/wailsapp/wails), `v3/` on master | vendor the shell, drop the webview |

Clone them wherever suits you; nothing here assumes a path.

GPUI is mostly a design document that happens to be written in Rust: different
language, different memory model, different renderer, so what usually transfers is
the model rather than the code. Where a self-contained algorithm is worth having
exactly — the R-tree that assigns draw order, for instance — port it and attribute
it like any other port.

### Licences in the Zed checkout

Zed is two licences in one repository, and our checkout holds crates from both
sides. Check `license` in a crate's `Cargo.toml` before porting from it.

    crates/gpui and its siblings      Apache-2.0, portable with attribution
    crates/gpui_shared_string         no licence stated — read only
    crates/gpui_util                  no licence stated — read only
    the Zed application itself        GPL, and deliberately not in our checkout

`crates/ztracing` is GPL-3.0-or-later and was removed from `upstream.pins` for that
reason. Facet is Apache-2.0; a crate nobody may port from should not sit in the tree
beside nine they can.

Taffy is a flexbox implementation whose behaviour we want exactly. It is ported,
which is a different act from reading — the algorithm and its edge cases come across
intact, only the language changes.

Wails v3 is a working per-OS shell already written in Go. That is code we take.

## What to read in GPUI

Start with the two pages its authors wrote about the parts that are hardest to infer
from the source:

- `crates/gpui/docs/contexts.md` — how `App`, `Context` and `AsyncApp` relate, and
  which of them may touch what
- `crates/gpui/docs/key_dispatch.md` — the focus tree and how a keystroke finds its
  action

The checkout also carries the ten crates gpui depends on. `refineable` underpins
style refinement, `scheduler` is the executor, and `sum_tree` is the B-tree behind
lists and text; the rest are support.

| Layer      | Read about                                  | Take                                              |
| ---------- | ------------------------------------------- | ------------------------------------------------- |
| `app/`     | the app, the entity map, subscriptions      | entity handles, the update cycle, effect ordering |
| `element/` | the element trait and the `div` element     | the three-phase lifecycle and why it splits so    |
| `style/`   | style and the styled trait                  | style refinement and the fluent builder shape     |
| `layout/`  | the Taffy integration                       | which flexbox subset is actually exercised        |
| `scene/`   | scene and geometry                          | the primitive set and how draw order is decided   |
| `window/`  | window, key dispatch, actions               | focus tree, hit testing, action dispatch          |
| `text/`    | the text system                             | shaping to line layout to glyph atlas             |

## What to port from Taffy

The flexbox solver, into `layout/`. Port the algorithm rather than reinterpreting
it: the ordering of the sizing passes, the treatment of `auto` margins, min and max
clamping, baseline alignment and the intrinsic-size caching are where a
reimplementation goes wrong. Taffy's own test suite is generated from browser
behaviour and should come across with the code.

Grid is out of scope. Port the flexbox path and the shared tree plumbing it needs.

## What to vendor from Wails

Wails is MIT, so vendoring is clean provided the notices travel. It is vendored
rather than imported because its window is inseparable from its webview — see the
rendering section of `docs/architecture.md`.

Three pieces come across almost untouched:

- `v3/pkg/w32` — Win32 bindings, around thirteen thousand lines, standalone
- `v3/pkg/mac` — Cocoa helpers
- `v3/pkg/application/mainthread_*.go` — dispatching work onto the platform main
  thread. Small, and it already handles the case that catches everyone: on Windows
  a modal inner loop swallows thread-queued messages, so the dispatch goes through
  a hidden window instead. Facet's foreground executor sits directly on this.

Above them, take the shell and leave the webview behind:

- Window creation and the native event loop for Win32, Cocoa and GTK
- Application lifecycle and single-instance handling
- Menus, tray, dialogs, notifications
- Screen and display information, clipboard
- The `wails3` build and packaging tooling

The per-platform window files mix window and webview concerns in one type, so this
part is surgery rather than a copy. Wails' side of the split is still the bulk of
the value, and an agent writing a window event loop from scratch has misread the
brief.

Their file layout is worth keeping: a common interface in one place, per-OS
implementations in `<feature>_{darwin,linux,windows}.go` behind build tags, and the
GTK cgo bridge isolated in its own file. It survives contact with four platforms,
which is more than most such schemes manage.

## Where they meet

`platform/` is the only package both GPUI and Wails reach. Its interface is shaped
by GPUI's platform abstraction and implemented with Wails' code, which makes it the
package to change most carefully.

## Text

[go-text/typesetting](https://github.com/go-text/typesetting) supplies font loading and matching, script and bidi
segmentation, and HarfBuzz-equivalent shaping, in pure Go. `text/` wraps it; nothing
above `text/` knows it exists.

Outstanding: rasterising the shaped outlines into the glyph atlas. `typesetting`
stops at outlines, so that is ours to choose or write.

## What none of them provides

- **The renderer.** GPUI's talks to Metal and Vulkan from Rust; Taffy and Wails have
  no opinion. Written from scratch in Go.
- **Client-area input.** Wails leaves pointer, wheel and key input to the webview.
  With the webview gone, `platform/` reads them from the native event stream on each
  platform.
