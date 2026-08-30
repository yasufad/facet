# Upstream sources

Two codebases sit alongside this one, and they serve opposite purposes.

    ../gpui     read it, port nothing
    ../wails    borrow the code

GPUI is a design document that happens to be written in Rust. None of it transfers
directly: different language, different memory model, different renderer. What
transfers is the model.

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

## What to borrow from Wails

- Window creation and the native event loop for Win32, Cocoa and GTK
- Application lifecycle
- Menus, tray, dialogs, notifications
- Screen and display information, clipboard
- The `wails3` build and packaging tooling

An agent writing a window event loop by hand has misread the brief.

## Where they meet

`platform/` is the only package both influences reach. Its interface is shaped by
GPUI's platform abstraction and implemented with Wails' code, which makes it the
package to change most carefully.

## What neither provides

- **The renderer.** GPUI's talks to Metal and Vulkan from Rust; Wails has none.
  Written from scratch in Go.
- **Flexbox.** Taffy is Rust. Ported or reimplemented.
- **Text shaping.** GPUI leans on CoreText and HarfBuzz. Likely `go-text/typesetting`.

Those three are the actual work.
