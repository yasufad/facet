//go:build windows

package d3d11

import (
	"fmt"
	"unsafe"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/scene"
)

const (
	atlasPageWidth  = 2048
	atlasPageHeight = 2048
)

// shelfPacker packs rectangular tiles into a fixed-size texture page using a
// simple shelf (row-based) bin packing algorithm.
type shelfPacker struct {
	width       int
	height      int
	cursorX     int
	cursorY     int
	shelfHeight int
}

func newShelfPacker(w, h int) *shelfPacker {
	return &shelfPacker{
		width:  w,
		height: h,
	}
}

func (p *shelfPacker) reset() {
	p.cursorX = 0
	p.cursorY = 0
	p.shelfHeight = 0
}

func (p *shelfPacker) allocate(w, h int) (int, int, bool) {
	if w <= 0 || h <= 0 || w > p.width || h > p.height {
		return 0, 0, false
	}

	// If it doesn't fit on the current shelf, start a new shelf
	if p.cursorX+w > p.width {
		p.cursorY += p.shelfHeight
		p.cursorX = 0
		p.shelfHeight = 0
	}

	// If it doesn't fit vertically, this page is full
	if p.cursorY+h > p.height {
		return 0, 0, false
	}

	x := p.cursorX
	y := p.cursorY

	p.cursorX += w
	if h > p.shelfHeight {
		p.shelfHeight = h
	}

	return x, y, true
}

type atlasPage struct {
	index   uint32
	kind    scene.AtlasTextureKind
	texture *comObject
	srv     *comObject
	packer  *shelfPacker
}

type atlasManager struct {
	device     *comObject
	context    *comObject
	nextTileID uint32
	monoPages  []*atlasPage
	polyPages  []*atlasPage
}

func newAtlasManager(device, context *comObject) *atlasManager {
	return &atlasManager{
		device:     device,
		context:    context,
		nextTileID: 1,
	}
}

func (m *atlasManager) createPage(kind scene.AtlasTextureKind, index uint32) (*atlasPage, error) {
	var format uint32
	if kind == scene.TextureMonochrome {
		format = dxgiFormatR8Unorm
	} else {
		format = dxgiFormatR8G8B8A8Unorm
	}

	desc := d3d11Texture2DDesc{
		Width:          atlasPageWidth,
		Height:         atlasPageHeight,
		MipLevels:      1,
		ArraySize:      1,
		Format:         format,
		SampleDesc:     dxgiSampleDesc{Count: 1, Quality: 0},
		Usage:          d3d11UsageDefault,
		BindFlags:      d3d11BindShaderResource,
		CPUAccessFlags: 0,
		MiscFlags:      0,
	}

	var texture *comObject
	// Sound because desc is on the stack and texture will receive the created COM object.
	r1, _, _ := m.device.call(5, uintptr(unsafe.Pointer(&desc)), 0, uintptr(unsafe.Pointer(&texture)))
	if int32(r1) < 0 || texture == nil {
		return nil, fmt.Errorf("create atlas texture: hr=0x%08x", uint32(r1))
	}

	var srv *comObject
	// Sound because srv is on the stack and receives the created shader resource view.
	r1, _, _ = m.device.call(7, uintptr(unsafe.Pointer(texture)), 0, uintptr(unsafe.Pointer(&srv)))
	if int32(r1) < 0 || srv == nil {
		texture.Release()
		return nil, fmt.Errorf("create atlas SRV: hr=0x%08x", uint32(r1))
	}

	return &atlasPage{
		index:   index,
		kind:    kind,
		texture: texture,
		srv:     srv,
		packer:  newShelfPacker(atlasPageWidth, atlasPageHeight),
	}, nil
}

func (m *atlasManager) upload(kind scene.AtlasTextureKind, size geometry.Size[geometry.DevicePixels], data []byte) (scene.AtlasTile, error) {
	w := int(size.Width)
	h := int(size.Height)
	if w <= 0 || h <= 0 || len(data) == 0 {
		return scene.AtlasTile{}, nil
	}

	var pages *[]*atlasPage
	if kind == scene.TextureMonochrome {
		pages = &m.monoPages
	} else {
		pages = &m.polyPages
	}

	// Try existing pages
	var targetPage *atlasPage
	var x, y int
	var ok bool

	for _, page := range *pages {
		x, y, ok = page.packer.allocate(w, h)
		if ok {
			targetPage = page
			break
		}
	}

	// Allocate a new page if none fit
	if targetPage == nil {
		newPage, err := m.createPage(kind, uint32(len(*pages)))
		if err != nil {
			return scene.AtlasTile{}, err
		}
		*pages = append(*pages, newPage)
		targetPage = newPage
		x, y, ok = targetPage.packer.allocate(w, h)
		if !ok {
			return scene.AtlasTile{}, fmt.Errorf("tile size %dx%d exceeds atlas maximum %dx%d", w, h, atlasPageWidth, atlasPageHeight)
		}
	}

	// Upload subresource box
	box := d3d11Box{
		Left:   uint32(x),
		Top:    uint32(y),
		Front:  0,
		Right:  uint32(x + w),
		Bottom: uint32(y + h),
		Back:   1,
	}

	var rowPitch uint32
	if kind == scene.TextureMonochrome {
		rowPitch = uint32(w)
	} else {
		rowPitch = uint32(w * 4)
	}

	// Sound because data is kept live during SyscallN and UpdateSubresource copies immediately.
	dataPtr := uintptr(0)
	if len(data) > 0 {
		dataPtr = uintptr(unsafe.Pointer(&data[0]))
	}

	// ID3D11DeviceContext::UpdateSubresource is vtbl index 48
	m.context.call(48,
		uintptr(unsafe.Pointer(targetPage.texture)),
		0,
		uintptr(unsafe.Pointer(&box)),
		dataPtr,
		uintptr(rowPitch),
		0,
	)

	tileID := m.nextTileID
	m.nextTileID++

	return scene.AtlasTile{
		TextureID: scene.AtlasTextureID{Index: targetPage.index, Kind: kind},
		TileID:    scene.TileID(tileID),
		Bounds: geometry.Bounds[geometry.DevicePixels]{
			Origin: geometry.NewPoint(geometry.DevicePixels(x), geometry.DevicePixels(y)),
			Size:   geometry.NewSize(geometry.DevicePixels(w), geometry.DevicePixels(h)),
		},
	}, nil
}

func (m *atlasManager) clear(kind scene.AtlasTextureKind) {
	var pages []*atlasPage
	if kind == scene.TextureMonochrome {
		pages = m.monoPages
	} else {
		pages = m.polyPages
	}
	for _, page := range pages {
		page.packer.reset()
	}
}

func (m *atlasManager) getSRV(id scene.AtlasTextureID) *comObject {
	var pages []*atlasPage
	if id.Kind == scene.TextureMonochrome {
		pages = m.monoPages
	} else {
		pages = m.polyPages
	}
	if int(id.Index) < len(pages) {
		return pages[id.Index].srv
	}
	return nil
}

func (m *atlasManager) release() {
	for _, page := range m.monoPages {
		if page.srv != nil {
			page.srv.Release()
		}
		if page.texture != nil {
			page.texture.Release()
		}
	}
	for _, page := range m.polyPages {
		if page.srv != nil {
			page.srv.Release()
		}
		if page.texture != nil {
			page.texture.Release()
		}
	}
	m.monoPages = nil
	m.polyPages = nil
}
