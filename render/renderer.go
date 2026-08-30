package render

import (
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/scene"
)

// Options configures a [Renderer] at construction.
//
// The zero value is a valid, default configuration (immediate presentation
// without waiting for vertical blanking).
type Options struct {
	// VSync enables vertical synchronisation to prevent visual tearing.
	// When true, presentation synchronises with the display refresh rate.
	// When false, frames are presented immediately.
	VSync bool
}

// Renderer is the GPU drawing interface. It takes native surface handles
// from [platform.Window], manages the swapchain and GPU atlas textures, and
// draws batched primitives from [scene.Scene].
//
// Renderer is a layer boundary. Methods change by explicit decision, never
// as a side effect of a backend implementation.
type Renderer interface {
	// Resize updates the swapchain and render target sizes to match the window's
	// client area in device pixels. It is called when the window is resized or
	// when its display scale factor changes.
	Resize(size geometry.Size[geometry.DevicePixels]) error

	// Draw renders the primitives in the scene into the backbuffer. Primitives
	// are consumed from [scene.Scene.Batches] in draw order using instanced
	// draw calls.
	//
	// The scene must have been prepared with [scene.Scene.Finish] prior to
	// calling Draw.
	Draw(s *scene.Scene) error

	// Present presents the rendered backbuffer to the native surface. If
	// [Options.VSync] is enabled, it synchronises with the vertical blank.
	Present() error

	// Upload allocates a tile in an atlas texture of the requested kind and
	// uploads the provided pixel or mask data to the GPU.
	//
	// For [scene.TextureMonochrome], data must contain 1 byte per pixel
	// (8-bit coverage mask, row-major).
	// For [scene.TexturePolychrome], data must contain 4 bytes per pixel
	// (32-bit RGBA, row-major).
	// For [scene.TextureSubpixel], data contains subpixel coverage bytes.
	//
	// If size has zero width or height, an empty [scene.AtlasTile] is returned
	// without allocating GPU resources.
	Upload(kind scene.AtlasTextureKind, size geometry.Size[geometry.DevicePixels], data []byte) (scene.AtlasTile, error)

	// ClearAtlas releases all allocated tiles and resets the atlas allocator
	// for the given texture kind, freeing GPU memory.
	ClearAtlas(kind scene.AtlasTextureKind)

	// Size returns the current swapchain dimensions in device pixels.
	Size() geometry.Size[geometry.DevicePixels]

	// Close releases all GPU resources, textures, swapchains, and pipeline
	// state associated with the renderer. After Close is called, the renderer
	// is unusable.
	Close() error
}
