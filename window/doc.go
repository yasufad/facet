// Package window drives the frame loop: it owns a native platform window,
// coordinates layout through paint, presents the scene, and routes input
// events into the dispatch tree.
//
// This is the package where all lower layers meet. platform produces raw input
// and lifecycle events, app produces notifications and reactive effects, layout
// solves flexbox geometries, text shapes lines and rasterises glyphs, scene
// batches primitives, render executes GPU draw calls, input resolves bindings,
// and element constructs the ephemeral UI hierarchy. window is the sole layer
// that imports all of them and connects their seams.
//
// # The Seven Frame Steps
//
// Every frame executes the following seven phases strictly in order:
//
//  1. Flush effects: drain reactive notifications in app, run observers, and
//     mark dirty views (app.Flush).
//  2. Request layout: evaluate the root view's element tree, building layout
//     nodes in the layout engine (layout.TaffyTree).
//  3. Layout: solve flexbox layout with the window's viewport bounds and derive
//     absolute window-relative bounds for all layout nodes.
//  4. Prepaint: elements commit their computed bounds, register input dispatch
//     nodes, and register hit regions in the in-flight frame.
//  5. Hit test: resolve the current pointer position against the hit regions
//     just registered in the in-flight frame. This allows hover and active
//     state queries (IsHovered, IsActive) to evaluate immediately during paint
//     with zero frame lag.
//  6. Paint: elements evaluate hover, active, and focus styles, and emit draw
//     primitives (quads, shadows, paths, text sprites) into the frame scene.
//  7. Present: swap the in-flight frame to become the on-screen rendered frame,
//     clear the next frame, reset the layout engine, and submit the scene to
//     the GPU renderer.
//
// Phase ordering is enforced strictly at runtime: calling RequestLayout outside
// step 2, RegisterHitRegion outside step 4, or IsHovered/InsertQuad outside step 6
// panics immediately.
//
// # Two Frames, and Two Hit Tests
//
// A window maintains two independent frame structures:
//
//	rendered   what is currently visible on screen; holds the presented scene,
//	           active hit regions, and the input dispatch tree.
//	next       what the current frame loop is constructing during prepaint and
//	           paint.
//
// Paint writes exclusively into next. At the end of presentation, rendered and
// next swap pointers and next is reset. This isolation guarantees that user
// input arriving between frames resolves against stable, fully formed geometry.
//
// The two hit tests serve distinct purposes and must not be conflated:
//
//   - Intra-frame hit test (Step 5): resolves the pointer against next.hitRegions
//     between prepaint and paint so that elements querying IsHovered or IsActive
//     during paint receive answers matching the current frame's geometry.
//   - Inter-frame event routing: resolves arriving platform input events against
//     rendered.hitRegions and dispatches through rendered.dispatchTree, because
//     that is the state the user is looking at and interacting with.
//
// # Architectural Decisions
//
// The following foundational decisions govern the window package:
//
//   - What schedules a frame: Two independent sources schedule frames — app state
//     mutations (effects flushing from updates) and platform events (native paint,
//     resize, scale change, or input). Invalidation marks the window dirty and
//     schedules a draw turn on the UI goroutine. A notification arriving mid-frame
//     buffers in the effect queue and flushes for the subsequent frame; it never
//     restarts the current frame in-place. When dirty is false, no frame runs,
//     preventing perpetual redraw loops.
//
//   - Presentation of unchanged frames: If a scheduled turn finds no dirty views
//     and no geometry changes, drawing and presentation are skipped. The idle
//     cost of an open Facet window is exactly 0 GPU draw calls and 0 present calls
//     per second. The native event loop sleeps waiting for OS messages.
//
//   - View invalidation and re-rendering: When a frame executes, the root view
//     re-renders unconditionally. Fine-grained per-view dirty tracking and
//     subtree caching are deferred to a future milestone; for now, when an
//     entity mutation or event marks the window dirty, the complete element tree
//     is rebuilt.
//
//   - Reactivity and entity observation: SetRootView observes the root view's
//     underlying entity directly. When that entity notifies (cx.Notify()), the
//     window marks itself dirty and schedules a redraw. Reactivity stops at the
//     root view: if a view reads secondary entities during Render, those entities
//     do not automatically trigger repaints when they change because there is no
//     automatic read tracking. Instead, the view explicitly observes any secondary
//     entity it depends on and forwards notifications to itself:
//
//     app.Observe(cx, childEntity, func(v *MyView, e app.Entity[Child], cx *app.Context[MyView]) {
//     cx.Notify()
//     })
//
//     This pattern makes dependencies explicit, avoiding the runtime overhead
//     and cache churn of per-frame read tracking.
//
//   - Per-element state lifetime: Elements are ephemeral value types rebuilt
//     fresh every frame and discarded after paint. Any state that must survive
//     the frame (scroll offsets, selection, cursor position) belongs in an
//     app.Entity[T]. There is no element-keyed state map on the frame, eliminating
//     memory leaks when elements are unmounted.
//
//   - Order of resize: When a platform.ResizeEvent arrives, the new logical size
//     is stored on the window and dirty is set to true. The GPU swapchain is not
//     resized eagerly inside the raw event handler; instead, swapchain resizing
//     occurs during the frame loop immediately before layout and paint. The
//     previously presented backbuffer remains displayed by the window manager
//     compositor until the new frame is painted and presented atomically in the
//     same turn, preventing visual blank or stretched flashes.
//
//   - Scale factor invalidation: When platform.ScaleChangedEvent arrives, the
//     window drops its CPU glyph raster cache (text.Atlas) and clears the GPU
//     monochrome texture atlas (render.Renderer.ClearAtlas). The scale factor is
//     updated, and a full relayout, swapchain resize, and repaint are triggered.
//
// # Threading and Invariants
//
// All window methods and frame loop executions run strictly on the single UI
// goroutine. Background operations marshal back to the UI goroutine via
// app.ForegroundExecutor, which window wires directly to platform.Dispatch.
//
// The sole exception to single-goroutine access is ScheduleFrame, which protects
// its frameScheduled flag with an internal mutex so background tasks or thread
// wakeups can safely request a redraw without racing with the UI loop.
package window
