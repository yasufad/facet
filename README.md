# Facet

A GUI framework for Go. You write Go, you get a native desktop application.

The design follows GPUI, the framework behind the Zed editor. State lives in an
entity map, invalidation is precise, and the element tree is rebuilt each frame.

## Status

`window`, the frame loop, is complete. Windows (Direct3D 11) is the first supported platform.

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
| `window`   | done        |
| `ui`       | not started |

`examples/quad` drives an element tree through the `window` frame loop and Direct3D 11.

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
