// Package element defines the core Element interface, the three-phase frame
// lifecycle, the element tree, the Frame capability interface implemented by
// window, and the Div container element.
//
// Elements are ephemeral values constructed fresh each frame and discarded after
// paint. Retained state that survives across frames belongs in entities in the
// app package.
//
// # The Three-Phase Lifecycle
//
// Every frame walks the element tree through three sequential phases:
//
//  1. RequestLayout: Elements request nodes in the flexbox layout engine with
//     their resolved styles and child identifiers. Returns a layout.NodeID.
//  2. Prepaint: Following the layout solve, computed bounds are committed, hit
//     regions are registered, and scroll/focus geometry is prepared.
//  3. Paint: Visual primitives (quads, shadows, text, paths, underlines) are
//     emitted into the frame's scene in draw order.
//
// The three phases execute strictly in order: RequestLayout -> Prepaint -> Paint.
// No phase may reach backwards or skip preceding steps. Calling phases out of
// order or invoking phase-restricted capabilities (such as requesting layout
// during prepaint or registering hit regions during paint) panics.
//
// # Per-Phase State Carriage
//
// GPUI associates per-phase state types with each element trait. In Go, generic
// associated types on interfaces do not exist. Two designs were weighed:
//
//   - Return any from RequestLayout/Prepaint and pass it back. This costs interface
//     boxing allocations and dynamic type assertions per element per phase every
//     frame.
//   - Keep per-phase state on the element struct itself with pointer receivers.
//     Because elements are built fresh each frame, struct fields (such as
//     childLayoutIDs and computed bounds) are filled by RequestLayout and read by
//     Prepaint and Paint.
//
// Facet adopts the second approach: pointer receivers and internal state fields.
// The compiler statically verifies types with zero runtime assertions and zero
// state boxing. The explicit consequence is that element instances become
// non-copyable during frame execution, and "an element is a value" means an
// ephemeral value addressed by pointer.
//
// # The Frame Interface
//
// Frame is declared by element and implemented by window. Elements never import
// window directly, breaking what would otherwise be an import cycle.
//
// Frame provides a narrow, capability-based contract:
//   - Request layout node creation and children association
//   - Read solved layout bounds in logical pixels
//   - Register hit regions for input routing
//   - Insert the six scene primitives (Quad, Shadow, Path, Underline,
//     MonochromeSprite, PolychromeSprite)
//   - Shape text lines at the window's scale factor
//   - Query the display scale factor and root rem size
//
// # Constructor Naming
//
// In Go, an exported type and an exported function in the same package cannot share
// the same identifier (e.g. Div). While GPUI provides div(), Facet exports the
// concrete struct Div and the constructor NewDiv(). This preserves standard Go
// naming conventions and clear documentation while enabling fluent chaining:
//
//	element.NewDiv().Flex().Bg(colour.Rgba{...}).Children(...)
//
// # Element Identity Across Frames
//
// State that outlives the frame belongs in an app.Entity[T], never in an
// unevicted map keyed by element identifier. Ephemeral trees avoid memory leaks
// in dynamic lists and scrolling views where unkeyed retained elements would
// otherwise accumulate indefinitely.
//
// # Child Erasure
//
// Container elements store children as []Element. Storing children behind the
// Element interface boxes each concrete pointer into two interface words. At
// standard UI tree scales (hundreds of elements), this allocation is negligible
// and enables heterogeneous composition without generic parameter explosion on
// container structs.
//
// # View Rendering and Type Erasure
//
// A view is an entity whose Render method transforms retained state into an
// element hierarchy. The Render[T] interface connects app entities to UI
// elements:
//
//	type Render[T any] interface {
//	    Render(cx *app.Context[T]) Element
//	}
//
// To allow window to manage root views without knowing the concrete entity type T,
// View[T] wraps app.Entity[T] and implements the type-erased AnyView interface:
//
//	type AnyView interface {
//	    Render(a *app.App) Element
//	}
//
// # Performance
//
// Because the element tree is rebuilt each frame, element construction sits on
// the per-frame hot path. Benchmark measurements for constructing an 11-node
// tree (a parent Div with ten styled children):
//
//	BenchmarkBuildTree10Children    ~5.3 µs/op    6896 B/op    15 allocs/op
package element
