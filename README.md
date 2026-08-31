# Facet

A GUI framework for Go. You write Go, you get a native desktop application.

The design follows GPUI, the framework behind the Zed editor. State lives in an
entity map, and the element tree is rebuilt each frame from retained state.

## Status

The frame loop is complete and draws, on Windows through Direct3D 11. A program can
open a window, layout boxes and text, and put a styled element tree on screen.

| Package    | State       |
|------------|-------------|
| `geometry` | done        |
| `colour`   | done        |
| `app`      | done        |
| `layout`   | done        |
| `scene`    | done        |
| `text`     | done        |
| `input`    | in progress |
| `style`    | done        |
| `platform` | Windows     |
| `render`   | Direct3D 11 |
| `element`  | done        |
| `window`   | done        |
| `ui`       | in progress |

`examples/quad` drives an element tree through the `window` frame loop and
Direct3D 11.

## How this is built

Most of the code is written by AI agents, one package at a time, working to the
assignments in `prompts/` and the conventions in [AGENTS.md](AGENTS.md). A human sets
the architecture, decides anything that crosses a layer boundary, and reviews what
lands.

## Documentation

- [docs/architecture.md](docs/architecture.md), the layer stack and the open decisions
- [docs/packages.md](docs/packages.md), what each package owns and may import
- [docs/sources.md](docs/sources.md), what we read, port and vendor
- [AGENTS.md](AGENTS.md), conventions for anyone working here

## Licence

Apache-2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
