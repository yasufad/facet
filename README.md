# Facet

A GUI framework for Go. You write Go, you get a native desktop application — no
HTML, no CSS and no JavaScript in the programming model.

The design follows GPUI, the framework behind the Zed editor: application state
lives in an entity map rather than a pointer graph, invalidation is precise rather
than diff-based, and the element tree is rebuilt each frame from retained state.

## Status

Early, and not usable. There is no working code yet — the design is being settled
first.

## Documentation

- [docs/architecture.md](docs/architecture.md) — the layer stack, the seams between
  layers, and which decisions are still open
- [docs/sources.md](docs/sources.md) — what we read, port and borrow, and from where
- [AGENTS.md](AGENTS.md) — conventions for anyone working here, human or agent

## Licence

Apache-2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
