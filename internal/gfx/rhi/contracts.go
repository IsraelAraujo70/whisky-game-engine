package rhi

type Instance interface {
	Backend() BackendKind
	CreateSurface(target SurfaceTarget) (Surface, error)
	CreateDevice(surface Surface, opts DeviceOptions) (Device, error)
	Destroy() error
}

type Surface interface {
	Backend() BackendKind
	Target() SurfaceTarget
	Destroy() error
}

type Device interface {
	Backend() BackendKind
	CreateSwapchain(surface Surface, desc SwapchainDescriptor) (Swapchain, error)
	WaitIdle() error
	Destroy() error
}

type Swapchain interface {
	Backend() BackendKind
	Descriptor() SwapchainDescriptor
	Resize(width, height int) error
	Destroy() error
}
