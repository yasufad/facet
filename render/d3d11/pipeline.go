//go:build windows

package d3d11

import (
	"fmt"
	"unsafe"
)

const (
	dxgiFormatR32G32B32A32Float = 2
	dxgiFormatR32G32Float       = 16
	dxgiFormatR32Float          = 41
	dxgiFormatR32Uint           = 42
)

type shaderProgram struct {
	vs          *comObject
	ps          *comObject
	inputLayout *comObject
	stride      uint32
}

func (s *shaderProgram) release() {
	if s.inputLayout != nil {
		s.inputLayout.Release()
	}
	if s.ps != nil {
		s.ps.Release()
	}
	if s.vs != nil {
		s.vs.Release()
	}
}

type pipelineManager struct {
	device          *comObject
	constantBuffer  *comObject
	blendState      *comObject
	rasterizerState *comObject
	samplerLinear   *comObject

	quadShader      shaderProgram
	shadowShader    shaderProgram
	monoShader      shaderProgram
	polyShader      shaderProgram
	pathShader      shaderProgram
	underlineShader shaderProgram
}

func newPipelineManager(device *comObject) (*pipelineManager, error) {
	p := &pipelineManager{device: device}

	if err := p.initCommon(); err != nil {
		p.release()
		return nil, err
	}

	if err := p.initShaders(); err != nil {
		p.release()
		return nil, err
	}

	return p, nil
}

func (p *pipelineManager) initCommon() error {
	// Constant Buffer (16 bytes = 4 floats: viewport width, height, atlas width, height)
	cbDesc := d3d11BufferDesc{
		ByteWidth:      16,
		Usage:          d3d11UsageDynamic,
		BindFlags:      d3d11BindConstantBuffer,
		CPUAccessFlags: d3d11CpuAccessWrite,
	}
	// ID3D11Device::CreateBuffer is vtbl index 3
	r1, _, _ := p.device.call(3, uintptr(unsafe.Pointer(&cbDesc)), 0, uintptr(unsafe.Pointer(&p.constantBuffer)))
	if int32(r1) < 0 || p.constantBuffer == nil {
		return fmt.Errorf("create constant buffer: hr=0x%08x", uint32(r1))
	}

	// Blend State (Premultiplied alpha)
	blendDesc := d3d11BlendDesc{
		AlphaToCoverageEnable:  0,
		IndependentBlendEnable: 0,
	}
	blendDesc.RenderTarget[0] = d3d11RenderTargetBlendDesc{
		BlendEnable:           1,
		SrcBlend:              d3d11BlendOne,
		DestBlend:             d3d11BlendInvSrcAlpha,
		BlendOp:               d3d11BlendOpAdd,
		SrcBlendAlpha:         d3d11BlendOne,
		DestBlendAlpha:        d3d11BlendInvSrcAlpha,
		BlendOpAlpha:          d3d11BlendOpAdd,
		RenderTargetWriteMask: d3d11ColorWriteEnableAll,
	}
	// ID3D11Device::CreateBlendState is vtbl index 20
	r1, _, _ = p.device.call(20, uintptr(unsafe.Pointer(&blendDesc)), uintptr(unsafe.Pointer(&p.blendState)))
	if int32(r1) < 0 || p.blendState == nil {
		return fmt.Errorf("create blend state: hr=0x%08x", uint32(r1))
	}

	// Rasterizer State
	rastDesc := d3d11RasterizerDesc{
		FillMode:        d3d11FillSolid,
		CullMode:        d3d11CullNone,
		DepthClipEnable: 1,
	}
	// ID3D11Device::CreateRasterizerState is vtbl index 22
	r1, _, _ = p.device.call(22, uintptr(unsafe.Pointer(&rastDesc)), uintptr(unsafe.Pointer(&p.rasterizerState)))
	if int32(r1) < 0 || p.rasterizerState == nil {
		return fmt.Errorf("create rasterizer state: hr=0x%08x", uint32(r1))
	}

	// Sampler State (Linear Clamp)
	sampDesc := d3d11SamplerDesc{
		Filter:   d3d11FilterMinMagMipLinear,
		AddressU: d3d11TextureAddressClamp,
		AddressV: d3d11TextureAddressClamp,
		AddressW: d3d11TextureAddressClamp,
		MaxLOD:   1000.0,
	}
	// ID3D11Device::CreateSamplerState is vtbl index 23
	r1, _, _ = p.device.call(23, uintptr(unsafe.Pointer(&sampDesc)), uintptr(unsafe.Pointer(&p.samplerLinear)))
	if int32(r1) < 0 || p.samplerLinear == nil {
		return fmt.Errorf("create sampler state: hr=0x%08x", uint32(r1))
	}

	return nil
}

func (p *pipelineManager) createShader(vsBytecode, psBytecode []byte, elements []d3d11InputElementDesc, stride uint32) (shaderProgram, error) {
	var prog shaderProgram
	prog.stride = stride

	vsPtr := uintptr(0)
	if len(vsBytecode) > 0 {
		vsPtr = uintptr(unsafe.Pointer(&vsBytecode[0]))
	}
	// ID3D11Device::CreateVertexShader is vtbl index 12
	r1, _, _ := p.device.call(12, vsPtr, uintptr(len(vsBytecode)), 0, uintptr(unsafe.Pointer(&prog.vs)))
	if int32(r1) < 0 || prog.vs == nil {
		return prog, fmt.Errorf("create vertex shader: hr=0x%08x", uint32(r1))
	}

	psPtr := uintptr(0)
	if len(psBytecode) > 0 {
		psPtr = uintptr(unsafe.Pointer(&psBytecode[0]))
	}
	// ID3D11Device::CreatePixelShader is vtbl index 15
	r1, _, _ = p.device.call(15, psPtr, uintptr(len(psBytecode)), 0, uintptr(unsafe.Pointer(&prog.ps)))
	if int32(r1) < 0 || prog.ps == nil {
		prog.release()
		return prog, fmt.Errorf("create pixel shader: hr=0x%08x", uint32(r1))
	}

	elemPtr := uintptr(0)
	if len(elements) > 0 {
		elemPtr = uintptr(unsafe.Pointer(&elements[0]))
	}
	// ID3D11Device::CreateInputLayout is vtbl index 11
	r1, _, _ = p.device.call(11, elemPtr, uintptr(len(elements)), vsPtr, uintptr(len(vsBytecode)), uintptr(unsafe.Pointer(&prog.inputLayout)))
	if int32(r1) < 0 || prog.inputLayout == nil {
		prog.release()
		return prog, fmt.Errorf("create input layout: hr=0x%08x", uint32(r1))
	}

	return prog, nil
}

func (p *pipelineManager) initShaders() error {
	var err error

	// 1. Quad
	semBounds := []byte("INST_BOUNDS\x00")
	semMask := []byte("INST_MASK\x00")
	semBgCol := []byte("INST_BG_COLOR\x00")
	semBorderCol := []byte("INST_BORDER_COL\x00")
	semRadii := []byte("INST_RADII\x00")
	semBorderW := []byte("INST_BORDER_W\x00")
	semBorderStyle := []byte("INST_BORDER_STYLE\x00")
	semPad0 := []byte("INST_PAD0\x00")
	semPad1 := []byte("INST_PAD1\x00")
	semPad2 := []byte("INST_PAD2\x00")

	quadElems := []d3d11InputElementDesc{
		{&semBounds[0], 0, dxgiFormatR32G32B32A32Float, 0, 0, d3d11InputPerInstanceData, 1},
		{&semMask[0], 0, dxgiFormatR32G32B32A32Float, 0, 16, d3d11InputPerInstanceData, 1},
		{&semBgCol[0], 0, dxgiFormatR32G32B32A32Float, 0, 32, d3d11InputPerInstanceData, 1},
		{&semBorderCol[0], 0, dxgiFormatR32G32B32A32Float, 0, 48, d3d11InputPerInstanceData, 1},
		{&semRadii[0], 0, dxgiFormatR32G32B32A32Float, 0, 64, d3d11InputPerInstanceData, 1},
		{&semBorderW[0], 0, dxgiFormatR32G32B32A32Float, 0, 80, d3d11InputPerInstanceData, 1},
		{&semBorderStyle[0], 0, dxgiFormatR32Uint, 0, 96, d3d11InputPerInstanceData, 1},
		{&semPad0[0], 0, dxgiFormatR32Uint, 0, 100, d3d11InputPerInstanceData, 1},
		{&semPad1[0], 0, dxgiFormatR32Uint, 0, 104, d3d11InputPerInstanceData, 1},
		{&semPad2[0], 0, dxgiFormatR32Uint, 0, 108, d3d11InputPerInstanceData, 1},
	}
	p.quadShader, err = p.createShader(quadVSBytecode, quadPSBytecode, quadElems, 112)
	if err != nil {
		return fmt.Errorf("quad shader: %w", err)
	}

	// 2. Shadow
	semColor := []byte("INST_COLOR\x00")
	semElemBounds := []byte("INST_ELEM_BOUNDS\x00")
	semElemRadii := []byte("INST_ELEM_RADII\x00")
	semBlur := []byte("INST_BLUR\x00")
	semInset := []byte("INST_INSET\x00")

	shadowElems := []d3d11InputElementDesc{
		{&semBounds[0], 0, dxgiFormatR32G32B32A32Float, 0, 0, d3d11InputPerInstanceData, 1},
		{&semRadii[0], 0, dxgiFormatR32G32B32A32Float, 0, 16, d3d11InputPerInstanceData, 1},
		{&semMask[0], 0, dxgiFormatR32G32B32A32Float, 0, 32, d3d11InputPerInstanceData, 1},
		{&semColor[0], 0, dxgiFormatR32G32B32A32Float, 0, 48, d3d11InputPerInstanceData, 1},
		{&semElemBounds[0], 0, dxgiFormatR32G32B32A32Float, 0, 64, d3d11InputPerInstanceData, 1},
		{&semElemRadii[0], 0, dxgiFormatR32G32B32A32Float, 0, 80, d3d11InputPerInstanceData, 1},
		{&semBlur[0], 0, dxgiFormatR32Float, 0, 96, d3d11InputPerInstanceData, 1},
		{&semInset[0], 0, dxgiFormatR32Uint, 0, 100, d3d11InputPerInstanceData, 1},
		{&semPad0[0], 0, dxgiFormatR32Uint, 0, 104, d3d11InputPerInstanceData, 1},
		{&semPad1[0], 0, dxgiFormatR32Uint, 0, 108, d3d11InputPerInstanceData, 1},
	}
	p.shadowShader, err = p.createShader(shadowVSBytecode, shadowPSBytecode, shadowElems, 112)
	if err != nil {
		return fmt.Errorf("shadow shader: %w", err)
	}

	// 3. Mono Sprite
	semTile := []byte("INST_TILE\x00")
	semTransMat := []byte("INST_TRANS_MAT\x00")
	semTransTx := []byte("INST_TRANS_TX\x00")

	monoElems := []d3d11InputElementDesc{
		{&semBounds[0], 0, dxgiFormatR32G32B32A32Float, 0, 0, d3d11InputPerInstanceData, 1},
		{&semMask[0], 0, dxgiFormatR32G32B32A32Float, 0, 16, d3d11InputPerInstanceData, 1},
		{&semColor[0], 0, dxgiFormatR32G32B32A32Float, 0, 32, d3d11InputPerInstanceData, 1},
		{&semTile[0], 0, dxgiFormatR32G32B32A32Float, 0, 48, d3d11InputPerInstanceData, 1},
		{&semTransMat[0], 0, dxgiFormatR32G32B32A32Float, 0, 64, d3d11InputPerInstanceData, 1},
		{&semTransTx[0], 0, dxgiFormatR32G32Float, 0, 80, d3d11InputPerInstanceData, 1},
		{&semPad0[0], 0, dxgiFormatR32G32Float, 0, 88, d3d11InputPerInstanceData, 1},
	}
	p.monoShader, err = p.createShader(monoSpriteVSBytecode, monoSpritePSBytecode, monoElems, 96)
	if err != nil {
		return fmt.Errorf("mono sprite shader: %w", err)
	}

	// 4. Poly Sprite
	semOpacity := []byte("INST_OPACITY\x00")
	semGrayscale := []byte("INST_GRAYSCALE\x00")

	polyElems := []d3d11InputElementDesc{
		{&semBounds[0], 0, dxgiFormatR32G32B32A32Float, 0, 0, d3d11InputPerInstanceData, 1},
		{&semMask[0], 0, dxgiFormatR32G32B32A32Float, 0, 16, d3d11InputPerInstanceData, 1},
		{&semTile[0], 0, dxgiFormatR32G32B32A32Float, 0, 32, d3d11InputPerInstanceData, 1},
		{&semRadii[0], 0, dxgiFormatR32G32B32A32Float, 0, 48, d3d11InputPerInstanceData, 1},
		{&semOpacity[0], 0, dxgiFormatR32Float, 0, 64, d3d11InputPerInstanceData, 1},
		{&semGrayscale[0], 0, dxgiFormatR32Uint, 0, 68, d3d11InputPerInstanceData, 1},
		{&semPad0[0], 0, dxgiFormatR32G32Float, 0, 72, d3d11InputPerInstanceData, 1},
	}
	p.polyShader, err = p.createShader(polySpriteVSBytecode, polySpritePSBytecode, polyElems, 80)
	if err != nil {
		return fmt.Errorf("poly sprite shader: %w", err)
	}

	// 5. Path (Per-Vertex)
	semPos := []byte("POSITION\x00")
	semTex := []byte("TEXCOORD\x00")
	semContentMask := []byte("CONTENT_MASK\x00")
	semCol := []byte("COLOR\x00")

	pathElems := []d3d11InputElementDesc{
		{&semPos[0], 0, dxgiFormatR32G32Float, 0, 0, d3d11InputPerVertexData, 0},
		{&semTex[0], 0, dxgiFormatR32G32Float, 0, 8, d3d11InputPerVertexData, 0},
		{&semContentMask[0], 0, dxgiFormatR32G32B32A32Float, 0, 16, d3d11InputPerVertexData, 0},
		{&semCol[0], 0, dxgiFormatR32G32B32A32Float, 0, 32, d3d11InputPerVertexData, 0},
	}
	p.pathShader, err = p.createShader(pathVSBytecode, pathPSBytecode, pathElems, 48)
	if err != nil {
		return fmt.Errorf("path shader: %w", err)
	}

	// 6. Underline
	semThickness := []byte("INST_THICKNESS\x00")
	semWavy := []byte("INST_WAVY\x00")

	underlineElems := []d3d11InputElementDesc{
		{&semBounds[0], 0, dxgiFormatR32G32B32A32Float, 0, 0, d3d11InputPerInstanceData, 1},
		{&semMask[0], 0, dxgiFormatR32G32B32A32Float, 0, 16, d3d11InputPerInstanceData, 1},
		{&semColor[0], 0, dxgiFormatR32G32B32A32Float, 0, 32, d3d11InputPerInstanceData, 1},
		{&semThickness[0], 0, dxgiFormatR32Float, 0, 48, d3d11InputPerInstanceData, 1},
		{&semWavy[0], 0, dxgiFormatR32Uint, 0, 52, d3d11InputPerInstanceData, 1},
		{&semPad0[0], 0, dxgiFormatR32G32Float, 0, 56, d3d11InputPerInstanceData, 1},
	}
	p.underlineShader, err = p.createShader(underlineVSBytecode, underlinePSBytecode, underlineElems, 64)
	if err != nil {
		return fmt.Errorf("underline shader: %w", err)
	}

	return nil
}

func (p *pipelineManager) release() {
	p.quadShader.release()
	p.shadowShader.release()
	p.monoShader.release()
	p.polyShader.release()
	p.pathShader.release()
	p.underlineShader.release()

	if p.samplerLinear != nil {
		p.samplerLinear.Release()
	}
	if p.rasterizerState != nil {
		p.rasterizerState.Release()
	}
	if p.blendState != nil {
		p.blendState.Release()
	}
	if p.constantBuffer != nil {
		p.constantBuffer.Release()
	}
}
