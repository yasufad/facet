//go:build windows

package d3d11

import (
	"fmt"
	"unsafe"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/render"
	"github.com/yasufad/facet/scene"
)

type d3d11Renderer struct {
	hwnd             uintptr
	size             geometry.Size[geometry.DevicePixels]
	opts             render.Options
	device           *comObject
	context          *comObject
	swapChain        *comObject
	renderTargetView *comObject

	atlas         *atlasManager
	pipeline      *pipelineManager
	dynamicBuffer *dynamicBuffer
}

// New creates a Direct3D 11 backend renderer bound to the given native window surface.
func New(surface uintptr, size geometry.Size[geometry.DevicePixels], opts render.Options) (render.Renderer, error) {
	if surface == 0 {
		return nil, fmt.Errorf("invalid surface handle: 0")
	}

	w := uint32(size.Width)
	h := uint32(size.Height)
	if w == 0 {
		w = 1
	}
	if h == 0 {
		h = 1
	}

	flags := uintptr(d3d11CreateDeviceBgraSupport)

	var (
		device  *comObject
		context *comObject
	)

	// D3D11CreateDevice(pAdapter, DriverType, Software, Flags, pFeatureLevels, FeatureLevels, SDKVersion, ppDevice, pFeatureLevel, ppImmediateContext)
	hr, _, _ := procD3D11CreateDevice.Call(
		0,
		uintptr(d3dDriverTypeHardware),
		0,
		flags,
		0,
		0,
		uintptr(d3d11SdkVersion),
		uintptr(unsafe.Pointer(&device)),
		0,
		uintptr(unsafe.Pointer(&context)),
	)
	if int32(hr) < 0 || device == nil || context == nil {
		return nil, fmt.Errorf("D3D11CreateDevice failed: hr=0x%08x", uint32(hr))
	}

	r := &d3d11Renderer{
		hwnd:    surface,
		size:    size,
		opts:    opts,
		device:  device,
		context: context,
	}

	if err := r.initSwapChain(w, h); err != nil {
		r.Close()
		return nil, err
	}

	r.atlas = newAtlasManager(device, context)

	pipeline, err := newPipelineManager(device)
	if err != nil {
		r.Close()
		return nil, err
	}
	r.pipeline = pipeline

	r.dynamicBuffer = newDynamicBuffer(device, context)

	return r, nil
}

func (r *d3d11Renderer) initSwapChain(width, height uint32) error {
	var dxgiDevice *comObject
	if hr := r.device.QueryInterface(&iidIDXGIDevice, unsafe.Pointer(&dxgiDevice)); hr < 0 || dxgiDevice == nil {
		return fmt.Errorf("query IDXGIDevice: hr=0x%08x", uint32(hr))
	}
	defer dxgiDevice.Release()

	var adapter *comObject
	// IDXGIDevice::GetAdapter is vtbl index 7
	r1, _, _ := dxgiDevice.call(7, uintptr(unsafe.Pointer(&adapter)))
	if int32(r1) < 0 || adapter == nil {
		return fmt.Errorf("get IDXGIAdapter: hr=0x%08x", uint32(r1))
	}
	defer adapter.Release()

	var factory *comObject
	// IDXGIAdapter::GetParent (IID_IDXGIFactory2) is vtbl index 6
	r1, _, _ = adapter.call(6, uintptr(unsafe.Pointer(&iidIDXGIFactory2)), uintptr(unsafe.Pointer(&factory)))
	if int32(r1) < 0 || factory == nil {
		return fmt.Errorf("get IDXGIFactory2: hr=0x%08x", uint32(r1))
	}
	defer factory.Release()

	desc := dxgiSwapChainDesc1{
		Width:       width,
		Height:      height,
		Format:      dxgiFormatB8G8R8A8Unorm,
		Stereo:      0,
		SampleDesc:  dxgiSampleDesc{Count: 1, Quality: 0},
		BufferUsage: dxgiUsageRenderTargetOutput,
		BufferCount: 2,
		Scaling:     0,
		SwapEffect:  dxgiSwapEffectFlipDiscard,
		AlphaMode:   0,
		Flags:       0,
	}

	var swapChain *comObject
	// IDXGIFactory2::CreateSwapChainForHwnd is vtbl index 13
	r1, _, _ = factory.call(13,
		uintptr(unsafe.Pointer(r.device)),
		r.hwnd,
		uintptr(unsafe.Pointer(&desc)),
		0,
		0,
		uintptr(unsafe.Pointer(&swapChain)),
	)
	if int32(r1) < 0 || swapChain == nil {
		// Fallback to DXGI_SWAP_EFFECT_DISCARD for older Windows versions
		desc.SwapEffect = dxgiSwapEffectDiscard
		r1, _, _ = factory.call(13,
			uintptr(unsafe.Pointer(r.device)),
			r.hwnd,
			uintptr(unsafe.Pointer(&desc)),
			0,
			0,
			uintptr(unsafe.Pointer(&swapChain)),
		)
		if int32(r1) < 0 || swapChain == nil {
			return fmt.Errorf("CreateSwapChainForHwnd: hr=0x%08x", uint32(r1))
		}
	}
	r.swapChain = swapChain

	return r.createRenderTarget()
}

func (r *d3d11Renderer) createRenderTarget() error {
	var backBuffer *comObject
	// IDXGISwapChain::GetBuffer(0, IID_ID3D11Texture2D, &backBuffer) is vtbl index 9
	r1, _, _ := r.swapChain.call(9, 0, uintptr(unsafe.Pointer(&iidID3D11Texture2D)), uintptr(unsafe.Pointer(&backBuffer)))
	if int32(r1) < 0 || backBuffer == nil {
		return fmt.Errorf("get swapchain backbuffer: hr=0x%08x", uint32(r1))
	}
	defer backBuffer.Release()

	// ID3D11Device::CreateRenderTargetView is vtbl index 9
	r1, _, _ = r.device.call(9, uintptr(unsafe.Pointer(backBuffer)), 0, uintptr(unsafe.Pointer(&r.renderTargetView)))
	if int32(r1) < 0 || r.renderTargetView == nil {
		return fmt.Errorf("create render target view: hr=0x%08x", uint32(r1))
	}

	return nil
}

func (r *d3d11Renderer) Resize(size geometry.Size[geometry.DevicePixels]) error {
	if size.Width <= 0 || size.Height <= 0 {
		return nil
	}
	r.size = size

	if r.renderTargetView != nil {
		r.renderTargetView.Release()
		r.renderTargetView = nil
	}

	w := uint32(size.Width)
	h := uint32(size.Height)

	// IDXGISwapChain::ResizeBuffers(0, w, h, 0, 0) is vtbl index 13
	r1, _, _ := r.swapChain.call(13, 0, uintptr(w), uintptr(h), 0, 0)
	if int32(r1) < 0 {
		return fmt.Errorf("swapchain ResizeBuffers: hr=0x%08x", uint32(r1))
	}

	return r.createRenderTarget()
}

func (r *d3d11Renderer) Size() geometry.Size[geometry.DevicePixels] {
	return r.size
}

func (r *d3d11Renderer) Upload(kind scene.AtlasTextureKind, size geometry.Size[geometry.DevicePixels], data []byte) (scene.AtlasTile, error) {
	return r.atlas.upload(kind, size, data)
}

func (r *d3d11Renderer) ClearAtlas(kind scene.AtlasTextureKind) {
	r.atlas.clear(kind)
}

func (r *d3d11Renderer) setShader(s *shaderProgram) {
	// ID3D11DeviceContext::VSSetShader is vtbl index 11
	r.context.call(11, uintptr(unsafe.Pointer(s.vs)), 0, 0)
	// ID3D11DeviceContext::PSSetShader is vtbl index 9
	r.context.call(9, uintptr(unsafe.Pointer(s.ps)), 0, 0)
	// ID3D11DeviceContext::IASetInputLayout is vtbl index 17
	r.context.call(17, uintptr(unsafe.Pointer(s.inputLayout)))
}

func (r *d3d11Renderer) bindVertexBuffer(buf *comObject, stride uint32) {
	strideVal := stride
	offsetVal := uint32(0)
	bufPtr := uintptr(unsafe.Pointer(buf))
	// ID3D11DeviceContext::IASetVertexBuffers is vtbl index 18
	r.context.call(18, 0, 1, uintptr(unsafe.Pointer(&bufPtr)), uintptr(unsafe.Pointer(&strideVal)), uintptr(unsafe.Pointer(&offsetVal)))
}

func (r *d3d11Renderer) Draw(s *scene.Scene) error {
	if s == nil || r.renderTargetView == nil {
		return nil
	}

	w := float32(r.size.Width)
	h := float32(r.size.Height)
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}

	// 1. Set Viewport
	vp := d3d11Viewport{
		TopLeftX: 0,
		TopLeftY: 0,
		Width:    w,
		Height:   h,
		MinDepth: 0.0,
		MaxDepth: 1.0,
	}
	// ID3D11DeviceContext::RSSetViewports is vtbl index 44
	r.context.call(44, 1, uintptr(unsafe.Pointer(&vp)))

	// 2. Set Render Target
	rtvPtr := uintptr(unsafe.Pointer(r.renderTargetView))
	// ID3D11DeviceContext::OMSetRenderTargets is vtbl index 33
	r.context.call(33, 1, uintptr(unsafe.Pointer(&rtvPtr)), 0)

	// 3. Clear Render Target View (transparent black)
	clearColor := [4]float32{0, 0, 0, 0}
	// ID3D11DeviceContext::ClearRenderTargetView is vtbl index 50
	r.context.call(50, rtvPtr, uintptr(unsafe.Pointer(&clearColor[0])))

	// 4. Set Blend, Rasterizer, Sampler State
	// ID3D11DeviceContext::OMSetBlendState is vtbl index 35
	r.context.call(35, uintptr(unsafe.Pointer(r.pipeline.blendState)), 0, 0xffffffff)
	// ID3D11DeviceContext::RSSetState is vtbl index 43
	r.context.call(43, uintptr(unsafe.Pointer(r.pipeline.rasterizerState)))
	// ID3D11DeviceContext::PSSetSamplers is vtbl index 10
	sampPtr := uintptr(unsafe.Pointer(r.pipeline.samplerLinear))
	r.context.call(10, 0, 1, uintptr(unsafe.Pointer(&sampPtr)))

	// 5. Update Constant Buffer
	cbData := [4]float32{w, h, float32(atlasPageWidth), float32(atlasPageHeight)}
	var mapped d3d11MappedSubresource
	// ID3D11DeviceContext::Map is vtbl index 14
	r1, _, _ := r.context.call(14,
		uintptr(unsafe.Pointer(r.pipeline.constantBuffer)),
		0,
		uintptr(d3d11MapWriteDiscard),
		0,
		uintptr(unsafe.Pointer(&mapped)),
	)
	if int32(r1) >= 0 && mapped.PData != nil {
		copy(unsafe.Slice((*float32)(mapped.PData), 4), cbData[:])
		// ID3D11DeviceContext::Unmap is vtbl index 15
		r.context.call(15, uintptr(unsafe.Pointer(r.pipeline.constantBuffer)), 0)
	}

	// Set Constant Buffers (VS index 7, PS index 16)
	cbPtr := uintptr(unsafe.Pointer(r.pipeline.constantBuffer))
	r.context.call(7, 0, 1, uintptr(unsafe.Pointer(&cbPtr)))
	r.context.call(16, 0, 1, uintptr(unsafe.Pointer(&cbPtr)))

	// Set Primitive Topology: D3D11_PRIMITIVE_TOPOLOGY_TRIANGLELIST (index 24)
	r.context.call(24, uintptr(d3d11PrimitiveTopologyTriangleList))

	// 6. Draw Batches
	for batch := range s.Batches() {
		switch batch.Kind {
		case scene.BatchShadows:
			if err := r.drawShadowBatch(s.Shadows()[batch.Range.Start:batch.Range.End]); err != nil {
				return err
			}
		case scene.BatchQuads:
			if err := r.drawQuadBatch(s.Quads()[batch.Range.Start:batch.Range.End]); err != nil {
				return err
			}
		case scene.BatchPaths:
			if err := r.drawPathBatch(s.Paths()[batch.Range.Start:batch.Range.End]); err != nil {
				return err
			}
		case scene.BatchUnderlines:
			if err := r.drawUnderlineBatch(s.Underlines()[batch.Range.Start:batch.Range.End]); err != nil {
				return err
			}
		case scene.BatchMonochromeSprites:
			if err := r.drawMonoSpriteBatch(s.MonochromeSprites()[batch.Range.Start:batch.Range.End], batch.TextureID); err != nil {
				return err
			}
		case scene.BatchPolychromeSprites:
			if err := r.drawPolySpriteBatch(s.PolychromeSprites()[batch.Range.Start:batch.Range.End], batch.TextureID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *d3d11Renderer) Present() error {
	if r.swapChain == nil {
		return nil
	}
	syncInterval := uintptr(0)
	if r.opts.VSync {
		syncInterval = 1
	}
	// IDXGISwapChain::Present is vtbl index 8
	r1, _, _ := r.swapChain.call(8, syncInterval, 0)
	if int32(r1) < 0 {
		return fmt.Errorf("swapchain Present: hr=0x%08x", uint32(r1))
	}
	return nil
}

func (r *d3d11Renderer) Close() error {
	if r.dynamicBuffer != nil {
		r.dynamicBuffer.release()
		r.dynamicBuffer = nil
	}
	if r.pipeline != nil {
		r.pipeline.release()
		r.pipeline = nil
	}
	if r.atlas != nil {
		r.atlas.release()
		r.atlas = nil
	}
	if r.renderTargetView != nil {
		r.renderTargetView.Release()
		r.renderTargetView = nil
	}
	if r.swapChain != nil {
		r.swapChain.Release()
		r.swapChain = nil
	}
	if r.context != nil {
		r.context.Release()
		r.context = nil
	}
	if r.device != nil {
		r.device.Release()
		r.device = nil
	}
	return nil
}
