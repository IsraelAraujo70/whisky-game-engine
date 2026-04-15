package rhi

// Instance represents a GPU API instance (Vulkan instance, Metal device, etc.).
type Instance interface {
	Backend() BackendKind
	CreateSurface(target SurfaceTarget) (Surface, error)
	CreateDevice(surface Surface, opts DeviceOptions) (Device, error)
	Destroy() error
}

// Surface represents a window surface that can be presented to.
type Surface interface {
	Backend() BackendKind
	Target() SurfaceTarget
	Destroy() error
}

// Device is the logical GPU device and the main factory for GPU resources.
type Device interface {
	Backend() BackendKind
	CreateSwapchain(surface Surface, desc SwapchainDescriptor) (Swapchain, error)
	CreateShaderModule(code []byte, stage ShaderStage) (ShaderModule, error)
	CreatePipeline(desc PipelineDescriptor) (Pipeline, error)
	CreateBuffer(desc BufferDescriptor) (Buffer, error)
	CreateTexture(desc TextureDescriptor) (Texture, error)
	GetQueue(kind QueueKind) (Queue, error)
	WaitIdle() error
	Destroy() error
}

// Swapchain manages the presentation surface image chain.
type Swapchain interface {
	Backend() BackendKind
	Descriptor() SwapchainDescriptor
	Resize(width, height int) error
	Destroy() error
}

// ShaderModule holds compiled shader bytecode.
type ShaderModule interface {
	Backend() BackendKind
	Stage() ShaderStage
	Destroy() error
}

// Pipeline represents a complete GPU rendering pipeline state.
type Pipeline interface {
	Backend() BackendKind
	Destroy() error
}

// Buffer is a GPU memory buffer (vertex, index, uniform, staging).
type Buffer interface {
	Backend() BackendKind
	Size() int
	Usage() BufferUsage
	Destroy() error
}

// Texture is a GPU texture (sampled, render target, etc.).
type Texture interface {
	Backend() BackendKind
	Width() int
	Height() int
	Format() PixelFormat
	Destroy() error
}

// DescriptorSet binds a set of resources (buffers, textures) to shader stages.
type DescriptorSet interface {
	Backend() BackendKind
	Destroy() error
}

// CommandBuffer records GPU commands for later submission.
type CommandBuffer interface {
	Backend() BackendKind
	// Reset prepares the command buffer for recording a new set of commands.
	Reset() error
}

// Queue represents a GPU submission queue.
type Queue interface {
	Backend() BackendKind
	Kind() QueueKind
	// Submit submits recorded command buffers for execution.
	Submit(cmds []CommandBuffer) error
	WaitIdle() error
}
