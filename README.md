# Facet

A GUI framework for Go. You write Go, you get a native desktop application.

The design follows GPUI, the framework behind the Zed editor. State lives in an
entity map, and the element tree is rebuilt each frame from retained state.

## Status

The frame loop draws. On Windows, through Direct3D 11, a program can open a window,
lay out boxes and text, and put a styled element tree on screen.

It does not yet respond correctly to input. A view cannot mutate its own state from a
click, there is no pointer capture, and hit testing ignores the clip stack.
[docs/audit.md](docs/audit.md) has the reproductions and the order the remaining
decisions come in; the packages below marked reopened each have an assignment in
`prompts/`.

| Package    | State                   |
|------------|-------------------------|
| `geometry` | done                    |
| `colour`   | done                    |
| `app`      | reopened                |
| `layout`   | done                    |
| `scene`    | done                    |
| `text`     | reopened                |
| `input`    | in progress             |
| `style`    | done                    |
| `platform` | Windows, reopened       |
| `render`   | Direct3D 11            |
| `element`  | reopened                |
| `window`   | reopened                |
| `ui`       | in progress             |

`examples/quad` drives an element tree through the `window` frame loop and
Direct3D 11. `examples/button` does not work; it is written the way the framework
intends and trips the defect the audit opens with.

## How this is built

Most of the code is written by AI agents, one package at a time, working to the
assignments in `prompts/` and the conventions in [AGENTS.md](AGENTS.md). A human sets
the architecture, decides anything that crosses a layer boundary, and reviews what
lands.

## Documentation

- [docs/audit.md](docs/audit.md), what works, what does not, and what to do about it
- [docs/architecture.md](docs/architecture.md), the layer stack and the open decisions
- [docs/packages.md](docs/packages.md), what each package owns and may import
- [docs/sources.md](docs/sources.md), what we read, port and vendor
- [AGENTS.md](AGENTS.md), conventions for anyone working here

## Licence

Apache-2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
