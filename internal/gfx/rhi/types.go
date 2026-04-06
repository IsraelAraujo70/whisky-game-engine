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
