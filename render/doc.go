// Package render defines the graphics layer boundary of Facet.
//
// It consumes a [scene.Scene] of ordered, batched primitives and draws it onto
// a native drawing surface provided by the [platform] package. Backends own the
// GPU device, the swapchain, atlas textures and shaders for a specific graphics
// API:
//
//	render/d3d11       Windows (Direct3D 11)
//	render/metal       macOS   (Metal)
//	render/vulkan      Linux   (Vulkan)
//
// # The layer boundary
//
// [Renderer] is one of Facet's three layer boundaries (along with [platform.Platform]
// and [element.Element]). Backends implement this interface; layers above
// (primarily the window package) interact solely through it. The interface
// changes by explicit decision, never as a side effect of a backend.
//
// Backends see nothing above the [scene] package — no elements, no styles, no
// entities, and no reactive machinery.
//
// # The atlas split
//
// Three packages touch texture atlases, with responsibilities strictly separated:
//
//   - text: rasterises glyphs into coverage masks and caches them in CPU memory
//     by face, size, and subpixel offset. It never packs or uploads masks.
//   - scene: carries an [scene.AtlasTile] reference (texture ID, tile ID, and bounds)
//     inside sprite primitives. It does not own or allocate GPU textures.
//   - render: owns GPU atlas textures, allocates tile slots within them, uploads
//     mask and pixel data, and resolves tile references at draw time.
//
// The text and render packages do not import each other. The window package sits
// above both and coordinates them: it requests masks from text, uploads them
// through [Renderer.Upload], and places the resulting [scene.AtlasTile] into
// scene primitives.
//
// Two texture kinds are used:
//   - [scene.TextureMonochrome]: single-channel 8-bit coverage maps for glyphs
//   - [scene.TexturePolychrome]: full-colour 32-bit RGBA maps for images and emoji
//   - [scene.TextureSubpixel]: reserved for multi-channel subpixel text
//
// # Shaders
//
// Shaders are precompiled to bytecode ahead of time and embedded with go:embed.
// A user's build never invokes fxc, dxc, metal, or glslc, ensuring pure Go
// builds without requiring native graphics SDKs installed.
//
// # Backend differences
//
// While the [Renderer] interface is uniform, underlying graphics backends differ:
//
//   - Native surfaces: Direct3D 11 binds its swapchain to an HWND. Metal binds to
//     a CAMetalLayer hosted by an NSView. Vulkan binds to a VkSurfaceKHR
//     created from a Wayland or X11 surface.
//   - Coordinate spaces: Direct3D 11 and Metal use a [0, 1] clip-space depth range;
//     Vulkan uses [0, 1] with an inverted Y viewport convention.
//   - Colour and alpha: Primitives carry straight alpha colours from the colour
//     package; backends premultiply alpha when writing GPU instance buffers
//     and write to sRGB/linear render targets.
//   - Presentation: Direct3D 11 uses DXGI swapchain presentation intervals; Metal
//     uses CAMetalLayer displaySyncEnabled or presentDrawable; Vulkan uses FIFO
//     or mailbox presentation modes.
//
// # Invariants
//
//   - Imports only geometry, colour, scene, and platform, enforced by the layering
//     test.
//   - No cgo. All backend bindings use syscall, purego, or vendored bindings.
//   - The UI runs on a single goroutine; Renderer methods are called from that
//     goroutine.
package render
