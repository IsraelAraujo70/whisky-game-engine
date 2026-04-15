package rhi

import platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"

type BackendKind string

const (
	BackendKindUnknown BackendKind = ""
	BackendKindVulkan  BackendKind = "vulkan"
	BackendKindD3D12   BackendKind = "d3d12"
	BackendKindMetal   BackendKind = "metal"
)

type PixelFormat string

const (
	PixelFormatUnknown         PixelFormat = ""
	PixelFormatBGRA8Unorm      PixelFormat = "bgra8_unorm"
	PixelFormatRGBA8Unorm      PixelFormat = "rgba8_unorm"
	PixelFormatBGRA8SRGB       PixelFormat = "bgra8_srgb"
	PixelFormatRGBA8SRGB       PixelFormat = "rgba8_srgb"
	PixelFormatDepth24Stencil8 PixelFormat = "depth24_stencil8"
)

type PresentMode string

const (
	PresentModeImmediate PresentMode = "immediate"
	PresentModeMailbox   PresentMode = "mailbox"
	PresentModeFIFO      PresentMode = "fifo"
)

type Extent2D struct {
	Width  int
	Height int
}

type SurfaceTarget struct {
	Window platformapi.NativeWindowHandle
	Extent Extent2D
}

type DeviceOptions struct {
	PreferDiscreteGPU bool
	EnableValidation  bool
}

type SwapchainDescriptor struct {
	Extent      Extent2D
	Format      PixelFormat
	PresentMode PresentMode
	BufferCount int
}

// ---------------------------------------------------------------------------
// Buffer
// ---------------------------------------------------------------------------

type BufferUsage uint32

const (
	BufferUsageVertex  BufferUsage = 1 << iota // vertex data
	BufferUsageIndex                           // index data
	BufferUsageUniform                         // uniform / constant data
	BufferUsageStaging                         // CPU-visible staging for upload
)

type BufferDescriptor struct {
	Size  int
	Usage BufferUsage
	Label string // optional debug label
}

// ---------------------------------------------------------------------------
// Texture
// ---------------------------------------------------------------------------

type TextureUsage uint32

const (
	TextureUsageSampled     TextureUsage = 1 << iota // can be sampled in a shader
	TextureUsageStorage                              // read/write in a compute shader
	TextureUsageRenderTarget                         // can be used as a color attachment
	TextureUsageTransferSrc                          // can be used as a blit/copy source
	TextureUsageTransferDst                          // can be used as a blit/copy destination
)

type TextureDescriptor struct {
	Width  int
	Height int
	Format PixelFormat
	Usage  TextureUsage
	Label  string // optional debug label
}

// ---------------------------------------------------------------------------
// Vertex layout
// ---------------------------------------------------------------------------

type VertexFormat uint32

const (
	VertexFormatFloat32x2 VertexFormat = iota // 2 x float32  (8 bytes)
	VertexFormatFloat32x3                     // 3 x float32 (12 bytes)
	VertexFormatFloat32x4                     // 4 x float32 (16 bytes)
)

type VertexAttribute struct {
	Location uint32
	Format   VertexFormat
	Offset   uint32
}

type VertexLayout struct {
	Stride     uint32
	Attributes []VertexAttribute
}

// ---------------------------------------------------------------------------
// Blend state
// ---------------------------------------------------------------------------

type BlendState uint32

const (
	BlendOpaque    BlendState = iota // no blending
	BlendAlpha                       // src-alpha / one-minus-src-alpha
	BlendAdditive                    // one / one
)

// ---------------------------------------------------------------------------
// Primitive topology
// ---------------------------------------------------------------------------

type PrimitiveTopology uint32

const (
	PrimitiveTopologyTriangleList  PrimitiveTopology = iota
	PrimitiveTopologyTriangleStrip
	PrimitiveTopologyLineList
	PrimitiveTopologyPointList
)

// ---------------------------------------------------------------------------
// Compare operation (depth / stencil)
// ---------------------------------------------------------------------------

type CompareOp uint32

const (
	CompareOpNever        CompareOp = iota
	CompareOpLess
	CompareOpEqual
	CompareOpLessEqual
	CompareOpGreater
	CompareOpNotEqual
	CompareOpGreaterEqual
	CompareOpAlways
)

// ---------------------------------------------------------------------------
// Shader stage
// ---------------------------------------------------------------------------

type ShaderStage uint32

const (
	ShaderStageVertex   ShaderStage = 1 << iota
	ShaderStageFragment
)

// ---------------------------------------------------------------------------
// Pipeline descriptor
// ---------------------------------------------------------------------------

type PipelineDescriptor struct {
	VertexShader   ShaderModule
	FragmentShader ShaderModule
	VertexLayout   VertexLayout
	Topology       PrimitiveTopology
	Blend          BlendState
	DepthTest      bool
	DepthWrite     bool
	DepthCompare   CompareOp
	Label          string // optional debug label
}

// ---------------------------------------------------------------------------
// Queue kind
// ---------------------------------------------------------------------------

type QueueKind uint32

const (
	QueueGraphics QueueKind = iota
	QueueCompute
	QueueTransfer
)
