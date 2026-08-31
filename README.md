# Facet

A GUI framework for Go. You write Go, you get a native desktop application.

The design follows GPUI, the framework behind the Zed editor. State lives in an
entity map, invalidation is precise, and the element tree is rebuilt each frame.

## Status

Not usable yet. `window`, the frame loop, is unwritten, so nothing puts an element
tree on screen. Windows is the only platform with a backend.

| Package    | State       |
|------------|-------------|
| `geometry` | done        |
| `colour`   | done        |
| `app`      | done        |
| `layout`   | done        |
| `scene`    | done        |
| `text`     | done        |
| `input`    | done        |
| `style`    | done        |
| `platform` | Windows     |
| `render`   | Direct3D 11 |
| `element`  | done        |
| `window`   | not started |
| `ui`       | not started |

`examples/quad` draws through Direct3D 11, wiring `platform` and `render` by hand.

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
