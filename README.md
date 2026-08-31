# Facet

A GUI framework for Go. You write Go, you get a native desktop application, with no
HTML, no CSS and no JavaScript in the programming model.

The design follows GPUI, the framework behind the Zed editor: application state lives
in an entity map rather than a pointer graph, invalidation is precise rather than
diff-based, and the element tree is rebuilt each frame from retained state.

## Status

Not usable yet. The layers below the frame loop are built and tested; nothing puts an
element tree on screen, because the package that would do it is unwritten. The
`examples/quad` programme draws a rounded, bordered quad through Direct3D 11 by
wiring `platform` and `render` together by hand, so the stack underneath is real.

Windows only so far. macOS and Linux need their own `platform` backend and their own
renderer, and neither has been started.

| Package    | State       | What it is                                          |
|------------|-------------|-----------------------------------------------------|
| `geometry` | done        | pixels, points, sizes, bounds, edges, corners        |
| `colour`   | done        | Rgba and Hsla                                        |
| `app`      | done        | entity map, contexts, effect queue, executors        |
| `layout`   | done        | Taffy's flexbox solver, ported                       |
| `scene`    | done        | the six paint primitives, draw ordering, batching    |
| `text`     | done        | font loading, shaping, line breaking, glyph raster   |
| `input`    | done        | keymaps, actions, focus tree, dispatch               |
| `style`    | done        | style properties and the refinement model            |
| `platform` | Windows     | windows, event loop, dispatch, raw input             |
| `render`   | Direct3D 11 | the Renderer interface and its GPU backends          |
| `element`  | in progress | Element, the three phases, `div`, the fluent builder |
| `window`   | not started | the frame loop, hit testing, input routing           |
| `ui`       | not started | buttons, labels, lists, text fields, scroll views    |

Roughly 30,000 lines of Go, plus 13,000 vendored. `window` is the keystone: it is the
only package that sees both a platform window and a renderer, and until it exists
none of the layers below it are connected to each other.

## Documentation

- [docs/architecture.md](docs/architecture.md): the layer stack, the seams between
  layers, and which decisions are still open
- [docs/packages.md](docs/packages.md): what each package owns, what it may import,
  and the invariants it holds
- [docs/sources.md](docs/sources.md): what we read, port and vendor, and from where
- [AGENTS.md](AGENTS.md): conventions for anyone working here, human or agent

## Licence

Apache-2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
