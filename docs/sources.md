# Upstream sources

Three codebases sit alongside this one, and they serve different purposes.

    ../gpui     read it, port nothing
    ../taffy    port it, algorithm for algorithm
    ../wails    borrow the code

GPUI is a design document that happens to be written in Rust. None of it transfers
directly: different language, different memory model, different renderer. What
transfers is the model.

Taffy is a flexbox implementation whose behaviour we want exactly. It is ported,
which is a different act from reading — the algorithm and its edge cases come across
intact, only the language changes.

Wails v3 is a working per-OS shell already written in Go. That is code we take.

## What to read in GPUI

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

## What to borrow from Wails

- Window creation and the native event loop for Win32, Cocoa and GTK
- Application lifecycle
- Menus, tray, dialogs, notifications
- Screen and display information, clipboard
- The `wails3` build and packaging tooling

An agent writing a window event loop by hand has misread the brief.

## Where they meet

`platform/` is the only package both GPUI and Wails reach. Its interface is shaped
by GPUI's platform abstraction and implemented with Wails' code, which makes it the
package to change most carefully.

## Text

`go-text/typesetting` supplies font loading and matching, script and bidi
segmentation, and HarfBuzz-equivalent shaping, in pure Go. `text/` wraps it; nothing
above `text/` knows it exists.

Outstanding: rasterising the shaped outlines into the glyph atlas. `typesetting`
stops at outlines, so that is ours to choose or write.

## What none of them provides

The renderer. GPUI's talks to Metal and Vulkan from Rust; Taffy and Wails have no
opinion. Written from scratch in Go.
