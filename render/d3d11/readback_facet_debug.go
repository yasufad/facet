//go:build windows && facet_debug

package d3d11

import (
	"fmt"
	"unsafe"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/render"
)

// ReadBackbuffer reads the entire swapchain backbuffer into a 2D grid of [colour.Rgba]
// pixels indexed by [y][x] coordinates. It is available only under the facet_debug build
// tag for test verification.
func ReadBackbuffer(r render.Renderer) ([][]colour.Rgba, error) {
	d, ok := r.(*d3d11Renderer)
	if !ok {
		return nil, fmt.Errorf("renderer is not *d3d11Renderer")
	}

	if d.swapChain == nil || d.device == nil || d.context == nil {
		return nil, fmt.Errorf("renderer is closed or not initialised")
	}

	w := int(d.size.Width)
	h := int(d.size.Height)
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("invalid renderer dimensions: %dx%d", w, h)
	}

	// 1. Get backbuffer texture from swapchain (IDXGISwapChain::GetBuffer is vtbl index 9)
	var backBuffer *comObject
	r1, _, _ := d.swapChain.call(9, 0, uintptr(unsafe.Pointer(&iidID3D11Texture2D)), uintptr(unsafe.Pointer(&backBuffer)))
	if int32(r1) < 0 || backBuffer == nil {
		return nil, fmt.Errorf("get swapchain backbuffer: hr=0x%08x", uint32(r1))
	}
	defer backBuffer.Release()

	// 2. Create staging texture with CPU read access
	desc := d3d11Texture2DDesc{
		Width:          uint32(w),
		Height:         uint32(h),
		MipLevels:      1,
		ArraySize:      1,
		Format:         dxgiFormatB8G8R8A8Unorm,
		SampleDesc:     dxgiSampleDesc{Count: 1, Quality: 0},
		Usage:          d3d11UsageStaging,
		BindFlags:      0,
		CPUAccessFlags: d3d11CpuAccessRead,
		MiscFlags:      0,
	}

	var stagingTex *comObject
	// ID3D11Device::CreateTexture2D is vtbl index 5
	r1, _, _ = d.device.call(5, uintptr(unsafe.Pointer(&desc)), 0, uintptr(unsafe.Pointer(&stagingTex)))
	if int32(r1) < 0 || stagingTex == nil {
		return nil, fmt.Errorf("create staging texture: hr=0x%08x", uint32(r1))
	}
	defer stagingTex.Release()

	// 3. Copy backbuffer to staging texture (ID3D11DeviceContext::CopyResource is vtbl index 47)
	d.context.call(47, uintptr(unsafe.Pointer(stagingTex)), uintptr(unsafe.Pointer(backBuffer)))

	// 4. Map staging texture for reading (ID3D11DeviceContext::Map is vtbl index 14)
	var mapped d3d11MappedSubresource
	r1, _, _ = d.context.call(14,
		uintptr(unsafe.Pointer(stagingTex)),
		0,
		uintptr(d3d11MapRead),
		0,
		uintptr(unsafe.Pointer(&mapped)),
	)
	if int32(r1) < 0 || mapped.PData == nil {
		return nil, fmt.Errorf("map staging texture: hr=0x%08x", uint32(r1))
	}
	// ID3D11DeviceContext::Unmap is vtbl index 15
	defer d.context.call(15, uintptr(unsafe.Pointer(stagingTex)), 0)

	rowPitch := int(mapped.RowPitch)
	// Sound because mapped.PData points to mapped GPU memory of at least rowPitch * h bytes.
	byteSlice := unsafe.Slice((*byte)(mapped.PData), rowPitch*h)

	pixels := make([][]colour.Rgba, h)
	for y := 0; y < h; y++ {
		row := make([]colour.Rgba, w)
		rowOffset := y * rowPitch
		for x := 0; x < w; x++ {
			pxOffset := rowOffset + x*4
			b := float32(byteSlice[pxOffset+0]) / 255.0
			g := float32(byteSlice[pxOffset+1]) / 255.0
			r := float32(byteSlice[pxOffset+2]) / 255.0
			a := float32(byteSlice[pxOffset+3]) / 255.0
			row[x] = colour.Rgba{R: r, G: g, B: b, A: a}
		}
		pixels[y] = row
	}

	return pixels, nil
}
