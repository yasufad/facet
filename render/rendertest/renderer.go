// Package rendertest provides an exported test double for the render package,
// so packages above render (window, ui, internal/integration) can test against
// a render.Renderer without each holding a private stub that breaks every time
// render.Renderer gains a method.
//
// It is the render counterpart to platform/platformtest and
// element/elementtest: one exported double turns a new interface method from a
// three-party handshake across agents who do not own each other's files into one
// package's commit.
package rendertest

import (
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/render"
	"github.com/yasufad/facet/scene"
)

// UploadCall records one Upload invocation, so a test can assert what was
// uploaded and at what size.
type UploadCall struct {
	Kind scene.AtlasTextureKind
	Size geometry.Size[geometry.DevicePixels]
	Data []byte
}

// ReleasedTile records one ReleaseTile invocation, so a test can assert
// which tiles the caller asked the atlas to reclaim.
type ReleasedTile struct {
	Tile scene.AtlasTile
}

// Renderer is an exported test double implementing render.Renderer. The zero
// value is a usable renderer with a zero size; fields are exported so a test
// can set the values it wants to read back, and read what a method under test
// wrote.
//
// Draw captures the scene's quads into Quads and stores the scene pointer in
// LastScene. Present, Resize and ClearAtlas increment their counters and
// append to their slices. Upload records each call in Uploads. Close sets
// Closed; further calls after Close are not checked.
type Renderer struct {
	SizeVal        geometry.Size[geometry.DevicePixels]
	Quads          []scene.Quad
	Presents       int
	Resizes        int
	ClearedAtlases []scene.AtlasTextureKind
	LastScene      *scene.Scene
	Uploads        []UploadCall
	ReleasedTiles  []ReleasedTile
	Closed         bool
}

// Ensure Renderer implements render.Renderer.
var _ render.Renderer = (*Renderer)(nil)

// NewRenderer constructs a test Renderer with the given swapchain size.
func NewRenderer(size geometry.Size[geometry.DevicePixels]) *Renderer {
	return &Renderer{SizeVal: size}
}

func (r *Renderer) Resize(size geometry.Size[geometry.DevicePixels]) error {
	r.SizeVal = size
	r.Resizes++
	return nil
}

func (r *Renderer) Draw(s *scene.Scene) error {
	r.LastScene = s
	r.Quads = append([]scene.Quad(nil), s.Quads()...)
	return nil
}

func (r *Renderer) Present() error {
	r.Presents++
	return nil
}

func (r *Renderer) Upload(kind scene.AtlasTextureKind, size geometry.Size[geometry.DevicePixels], data []byte) (scene.AtlasTile, error) {
	r.Uploads = append(r.Uploads, UploadCall{Kind: kind, Size: size, Data: data})
	return scene.AtlasTile{}, nil
}

func (r *Renderer) ClearAtlas(kind scene.AtlasTextureKind) {
	r.ClearedAtlases = append(r.ClearedAtlases, kind)
}

func (r *Renderer) ReleaseTile(tile scene.AtlasTile) {
	r.ReleasedTiles = append(r.ReleasedTiles, ReleasedTile{Tile: tile})
}

func (r *Renderer) Size() geometry.Size[geometry.DevicePixels] {
	return r.SizeVal
}

func (r *Renderer) Close() error {
	r.Closed = true
	return nil
}
