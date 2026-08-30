//go:build windows

package d3d11

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modD3D11 = syscall.NewLazyDLL("d3d11.dll")
	modDXGI  = syscall.NewLazyDLL("dxgi.dll")

	procD3D11CreateDevice = modD3D11.NewProc("D3D11CreateDevice")
)

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	iidIDXGIDevice = GUID{
		Data1: 0x54ec77fa, Data2: 0x1377, Data3: 0x44e6,
		Data4: [8]byte{0x8c, 0x32, 0x88, 0xfd, 0x5f, 0x44, 0xc8, 0x4c},
	}
	iidIDXGIFactory2 = GUID{
		Data1: 0x50c83a1c, Data2: 0xe072, Data3: 0x4c48,
		Data4: [8]byte{0x87, 0xb0, 0x36, 0x30, 0xfa, 0x36, 0xa6, 0xd0},
	}
	iidID3D11Texture2D = GUID{
		Data1: 0x6f15aaf2, Data2: 0xd208, Data3: 0x4e89,
		Data4: [8]byte{0x9a, 0xb4, 0x48, 0x95, 0x35, 0xd3, 0x4f, 0x9c},
	}
)

const (
	d3d11SdkVersion = 7

	d3dDriverTypeHardware = 1

	d3d11CreateDeviceBgraSupport = 0x20
	d3d11CreateDeviceDebug       = 0x2

	dxgiFormatR8G8B8A8Unorm = 28
	dxgiFormatB8G8R8A8Unorm = 87
	dxgiFormatR8Unorm       = 61

	dxgiUsageRenderTargetOutput = 1 << 5
	dxgiUsageShaderInput        = 1 << 4

	dxgiSwapEffectDiscard        = 0
	dxgiSwapEffectFlipSequential = 3
	dxgiSwapEffectFlipDiscard    = 4

	d3d11UsageDefault   = 0
	d3d11UsageImmutable = 1
	d3d11UsageDynamic   = 2
	d3d11UsageStaging   = 3

	d3d11BindVertexBuffer   = 0x1
	d3d11BindIndexBuffer    = 0x2
	d3d11BindConstantBuffer = 0x4
	d3d11BindShaderResource = 0x8
	d3d11BindRenderTarget   = 0x20

	d3d11CpuAccessWrite = 0x10000
	d3d11CpuAccessRead  = 0x20000

	d3d11MapWriteDiscard = 4

	d3d11PrimitiveTopologyTriangleList  = 4
	d3d11PrimitiveTopologyTriangleStrip = 5

	d3d11CullNone  = 1
	d3d11FillSolid = 3

	d3d11BlendZero        = 1
	d3d11BlendOne         = 2
	d3d11BlendSrcAlpha    = 5
	d3d11BlendInvSrcAlpha = 6
	d3d11BlendOpAdd       = 1

	d3d11ColorWriteEnableAll = 0xf

	d3d11FilterMinMagMipLinear = 0x15
	d3d11TextureAddressClamp   = 3
)

type comObject struct {
	vtbl *uintptr
}

func (o *comObject) call(methodIndex int, args ...uintptr) (uintptr, uintptr, error) {
	if o == nil || o.vtbl == nil {
		return 0, 0, fmt.Errorf("call on nil COM object")
	}
	// Sound because vtbl points to the interface function table.
	entry := *(*uintptr)(unsafe.Pointer(uintptr(unsafe.Pointer(o.vtbl)) + uintptr(methodIndex)*unsafe.Sizeof(uintptr(0))))
	allArgs := make([]uintptr, len(args)+1)
	allArgs[0] = uintptr(unsafe.Pointer(o))
	copy(allArgs[1:], args)
	r1, r2, err := syscall.SyscallN(entry, allArgs...)
	return r1, r2, err
}

func (o *comObject) Release() uint32 {
	if o == nil || o.vtbl == nil {
		return 0
	}
	r1, _, _ := o.call(2)
	return uint32(r1)
}

func (o *comObject) QueryInterface(riid *GUID, ppv unsafe.Pointer) int32 {
	// Sound because riid and ppv are pointers provided by the caller on the Go stack.
	r1, _, _ := o.call(0, uintptr(unsafe.Pointer(riid)), uintptr(ppv))
	return int32(r1)
}

// DXGI structs
type dxgiSampleDesc struct {
	Count   uint32
	Quality uint32
}

type dxgiSwapChainDesc1 struct {
	Width       uint32
	Height      uint32
	Format      uint32
	Stereo      int32
	SampleDesc  dxgiSampleDesc
	BufferUsage uint32
	BufferCount uint32
	Scaling     uint32
	SwapEffect  uint32
	AlphaMode   uint32
	Flags       uint32
}

// D3D11 structs
type d3d11BufferDesc struct {
	ByteWidth           uint32
	Usage               uint32
	BindFlags           uint32
	CPUAccessFlags      uint32
	MiscFlags           uint32
	StructureByteStride uint32
}

type d3d11SubresourceData struct {
	PSysMem          unsafe.Pointer
	SysMemPitch      uint32
	SysMemSlicePitch uint32
}

type d3d11Texture2DDesc struct {
	Width          uint32
	Height         uint32
	MipLevels      uint32
	ArraySize      uint32
	Format         uint32
	SampleDesc     dxgiSampleDesc
	Usage          uint32
	BindFlags      uint32
	CPUAccessFlags uint32
	MiscFlags      uint32
}

type d3d11Box struct {
	Left   uint32
	Top    uint32
	Front  uint32
	Right  uint32
	Bottom uint32
	Back   uint32
}

type d3d11RenderTargetBlendDesc struct {
	BlendEnable           int32
	SrcBlend              uint32
	DestBlend             uint32
	BlendOp               uint32
	SrcBlendAlpha         uint32
	DestBlendAlpha        uint32
	BlendOpAlpha          uint32
	RenderTargetWriteMask uint8
	Pad                   [3]uint8
}

type d3d11BlendDesc struct {
	AlphaToCoverageEnable  int32
	IndependentBlendEnable int32
	RenderTarget           [8]d3d11RenderTargetBlendDesc
}

type d3d11RasterizerDesc struct {
	FillMode              uint32
	CullMode              uint32
	FrontCounterClockwise int32
	DepthBias             int32
	DepthBiasClamp        float32
	SlopeScaledDepthBias  float32
	DepthClipEnable       int32
	ScissorEnable         int32
	MultisampleEnable     int32
	AntialiasedLineEnable int32
}

type d3d11SamplerDesc struct {
	Filter         uint32
	AddressU       uint32
	AddressV       uint32
	AddressW       uint32
	MipLODBias     float32
	MaxAnisotropy  uint32
	ComparisonFunc uint32
	BorderColor    [4]float32
	MinLOD         float32
	MaxLOD         float32
}

type d3d11Viewport struct {
	TopLeftX float32
	TopLeftY float32
	Width    float32
	Height   float32
	MinDepth float32
	MaxDepth float32
}

type d3d11MappedSubresource struct {
	PData      unsafe.Pointer
	RowPitch   uint32
	DepthPitch uint32
}

type d3d11InputElementDesc struct {
	SemanticName         *byte
	SemanticIndex        uint32
	Format               uint32
	InputSlot            uint32
	AlignedByteOffset    uint32
	InputSlotClass       uint32
	InstanceDataStepRate uint32
}

const (
	d3d11InputPerVertexData   = 0
	d3d11InputPerInstanceData = 1
	d3d11AppendAlignedElement = 0xffffffff
)
