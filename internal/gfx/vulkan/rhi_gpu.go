package vulkan

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
)

// ---------------------------------------------------------------------------
// ShaderModule
// ---------------------------------------------------------------------------

type shaderModule struct {
	device *device
	handle vkShaderModule
	stage  rhi.ShaderStage
}

func (d *device) CreateShaderModule(code []byte, stage rhi.ShaderStage) (rhi.ShaderModule, error) {
	if d == nil || d.handle == 0 {
		return nil, ErrNotImplemented
	}
	if len(code) == 0 || len(code)%4 != 0 {
		return nil, fmt.Errorf("%w: shader bytecode must be non-empty and aligned to 4 bytes", ErrCreateShaderModule)
	}
	createInfo := vkShaderModuleCreateInfo{
		SType:    vkStructureTypeShaderModuleCreateInfo,
		CodeSize: uintptr(len(code)),
		PCode:    (*uint32)(unsafe.Pointer(unsafe.SliceData(code))),
	}
	var handle vkShaderModule
	result := d.api.createShaderModule(d.handle, &createInfo, nil, &handle)
	runtime.KeepAlive(code)
	if result != vkSuccess {
		return nil, fmt.Errorf("%w: %s", ErrCreateShaderModule, result)
	}
	mod := &shaderModule{device: d, handle: handle, stage: stage}
	runtime.SetFinalizer(mod, func(m *shaderModule) { _ = m.Destroy() })
	return mod, nil
}

func (m *shaderModule) Backend() rhi.BackendKind { return rhi.BackendKindVulkan }
func (m *shaderModule) Stage() rhi.ShaderStage   { return m.stage }

func (m *shaderModule) Destroy() error {
	if m == nil || m.handle == 0 || m.device == nil {
		return nil
	}
	m.device.api.destroyShaderModule(m.device.handle, m.handle, nil)
	m.handle = 0
	runtime.SetFinalizer(m, nil)
	return nil
}

// Handle returns the underlying Vulkan shader module handle for internal use.
func (m *shaderModule) Handle() vkShaderModule { return m.handle }

// ---------------------------------------------------------------------------
// Pipeline (stub -- the actual Renderer2D builds its pipeline internally)
// ---------------------------------------------------------------------------

type pipeline struct {
	device         *device
	pipelineHandle vkPipeline
	layoutHandle   vkPipelineLayout
}

func (d *device) CreatePipeline(desc rhi.PipelineDescriptor) (rhi.Pipeline, error) {
	if d == nil || d.handle == 0 {
		return nil, ErrNotImplemented
	}
	normalized, err := rhi.NormalizePipelineDescriptor(desc)
	if err != nil {
		return nil, err
	}
	_ = normalized
	// TODO: Full pipeline creation from PipelineDescriptor.
	// The current Renderer2D hard-codes its pipeline. This stub validates
	// the descriptor and returns ErrNotImplemented until the renderer is
	// refactored to use RHI pipelines.
	return nil, fmt.Errorf("%w: generic pipeline creation from descriptor is not yet wired -- use Renderer2D.createGraphicsPipeline for now", ErrNotImplemented)
}

func (p *pipeline) Backend() rhi.BackendKind { return rhi.BackendKindVulkan }

func (p *pipeline) Destroy() error {
	if p == nil || p.device == nil {
		return nil
	}
	if p.pipelineHandle != 0 {
		p.device.api.destroyPipeline(p.device.handle, p.pipelineHandle, nil)
		p.pipelineHandle = 0
	}
	if p.layoutHandle != 0 {
		p.device.api.destroyPipelineLayout(p.device.handle, p.layoutHandle, nil)
		p.layoutHandle = 0
	}
	runtime.SetFinalizer(p, nil)
	return nil
}

// ---------------------------------------------------------------------------
// Buffer
// ---------------------------------------------------------------------------

type rhiBuffer struct {
	device *device
	inner  gpuBuffer
	usage  rhi.BufferUsage
	size   int
}

func (d *device) CreateBuffer(desc rhi.BufferDescriptor) (rhi.Buffer, error) {
	if d == nil || d.handle == 0 {
		return nil, ErrNotImplemented
	}
	normalized, err := rhi.NormalizeBufferDescriptor(desc)
	if err != nil {
		return nil, err
	}

	vkUsage := mapBufferUsage(normalized.Usage)
	memProps := uint32(vkMemoryPropertyHostVisibleBit | vkMemoryPropertyHostCoherentBit)
	if normalized.Usage&rhi.BufferUsageStaging == 0 {
		// Non-staging buffers default to device-local; the caller is expected
		// to use a staging buffer for uploads. For now, we keep host-visible
		// because the engine currently maps vertex buffers directly.
		memProps = vkMemoryPropertyHostVisibleBit | vkMemoryPropertyHostCoherentBit
	}

	inner, err := createBufferOnDevice(d, vkDeviceSize(normalized.Size), vkUsage, memProps)
	if err != nil {
		return nil, err
	}

	buf := &rhiBuffer{device: d, inner: inner, usage: normalized.Usage, size: normalized.Size}
	runtime.SetFinalizer(buf, func(b *rhiBuffer) { _ = b.Destroy() })
	return buf, nil
}

func (b *rhiBuffer) Backend() rhi.BackendKind { return rhi.BackendKindVulkan }
func (b *rhiBuffer) Size() int                { return b.size }
func (b *rhiBuffer) Usage() rhi.BufferUsage   { return b.usage }

func (b *rhiBuffer) Destroy() error {
	if b == nil || b.device == nil {
		return nil
	}
	b.inner.destroy(b.device)
	runtime.SetFinalizer(b, nil)
	return nil
}

func mapBufferUsage(usage rhi.BufferUsage) uint32 {
	var flags uint32
	if usage&rhi.BufferUsageVertex != 0 {
		flags |= vkBufferUsageVertexBufferBit
	}
	if usage&rhi.BufferUsageIndex != 0 {
		flags |= 0x00000040 // VK_BUFFER_USAGE_INDEX_BUFFER_BIT
	}
	if usage&rhi.BufferUsageUniform != 0 {
		flags |= 0x00000010 // VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT
	}
	if usage&rhi.BufferUsageStaging != 0 {
		flags |= vkBufferUsageTransferSrcBit
	}
	if flags == 0 {
		flags = vkBufferUsageVertexBufferBit
	}
	return flags
}

// createBufferOnDevice is a device-level buffer creation helper extracted from
// Renderer2D.createBuffer so that both the renderer and the RHI path can share
// the same low-level allocation logic.
func createBufferOnDevice(d *device, size vkDeviceSize, usage uint32, properties uint32) (gpuBuffer, error) {
	createInfo := vkBufferCreateInfo{
		SType:       vkStructureTypeBufferCreateInfo,
		Size:        size,
		Usage:       usage,
		SharingMode: vkSharingModeExclusive,
	}
	var buffer vkBuffer
	if result := d.api.createBuffer(d.handle, &createInfo, nil, &buffer); result != vkSuccess {
		return gpuBuffer{}, fmt.Errorf("%w: %s", ErrCreateBuffer, result)
	}

	var requirements vkMemoryRequirements
	d.api.getBufferMemoryRequirements(d.handle, buffer, &requirements)
	memoryType, err := findMemoryType(d.memoryProperties, requirements.MemoryTypeBits, properties)
	if err != nil {
		d.api.destroyBuffer(d.handle, buffer, nil)
		return gpuBuffer{}, err
	}
	allocInfo := vkMemoryAllocateInfo{
		SType:           vkStructureTypeMemoryAllocateInfo,
		AllocationSize:  requirements.Size,
		MemoryTypeIndex: memoryType,
	}
	var memory vkDeviceMemory
	if result := d.api.allocateMemory(d.handle, &allocInfo, nil, &memory); result != vkSuccess {
		d.api.destroyBuffer(d.handle, buffer, nil)
		return gpuBuffer{}, fmt.Errorf("%w: %s", ErrAllocateMemory, result)
	}
	if result := d.api.bindBufferMemory(d.handle, buffer, memory, 0); result != vkSuccess {
		d.api.freeMemory(d.handle, memory, nil)
		d.api.destroyBuffer(d.handle, buffer, nil)
		return gpuBuffer{}, fmt.Errorf("%w: %s", ErrCreateBuffer, result)
	}
	return gpuBuffer{buffer: buffer, memory: memory, size: size}, nil
}

// ---------------------------------------------------------------------------
// Texture
// ---------------------------------------------------------------------------

type rhiTexture struct {
	device *device
	image  vkImage
	memory vkDeviceMemory
	view   vkImageView
	width  int
	height int
	format rhi.PixelFormat
}

func (d *device) CreateTexture(desc rhi.TextureDescriptor) (rhi.Texture, error) {
	if d == nil || d.handle == 0 {
		return nil, ErrNotImplemented
	}
	normalized, err := rhi.NormalizeTextureDescriptor(desc)
	if err != nil {
		return nil, err
	}
	_ = normalized
	// TODO: Full texture creation from TextureDescriptor.
	// The current Renderer2D creates images through createTextureFromRGBA
	// which also uploads pixel data. This stub validates the descriptor
	// but returns ErrNotImplemented until we refactor the image upload path.
	return nil, fmt.Errorf("%w: generic texture creation from descriptor is not yet wired -- use Renderer2D.createTextureFromRGBA for now", ErrNotImplemented)
}

func (t *rhiTexture) Backend() rhi.BackendKind { return rhi.BackendKindVulkan }
func (t *rhiTexture) Width() int               { return t.width }
func (t *rhiTexture) Height() int              { return t.height }
func (t *rhiTexture) Format() rhi.PixelFormat  { return t.format }

func (t *rhiTexture) Destroy() error {
	if t == nil || t.device == nil {
		return nil
	}
	if t.view != 0 {
		t.device.api.destroyImageView(t.device.handle, t.view, nil)
		t.view = 0
	}
	if t.image != 0 {
		t.device.api.destroyImage(t.device.handle, t.image, nil)
		t.image = 0
	}
	if t.memory != 0 {
		t.device.api.freeMemory(t.device.handle, t.memory, nil)
		t.memory = 0
	}
	runtime.SetFinalizer(t, nil)
	return nil
}

// ---------------------------------------------------------------------------
// DescriptorSet (stub)
// ---------------------------------------------------------------------------

type rhiDescriptorSet struct {
	device *device
	handle vkDescriptorSet
}

func (ds *rhiDescriptorSet) Backend() rhi.BackendKind { return rhi.BackendKindVulkan }
func (ds *rhiDescriptorSet) Destroy() error           { return nil } // pooled -- destroyed with pool

// ---------------------------------------------------------------------------
// CommandBuffer (stub)
// ---------------------------------------------------------------------------

type rhiCommandBuffer struct {
	device *device
	handle vkCommandBuffer
}

func (cb *rhiCommandBuffer) Backend() rhi.BackendKind { return rhi.BackendKindVulkan }

func (cb *rhiCommandBuffer) Reset() error {
	if cb == nil || cb.handle == 0 || cb.device == nil {
		return nil
	}
	if result := cb.device.api.resetCommandBuffer(cb.handle, 0); result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrQueueSubmit, result)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Queue
// ---------------------------------------------------------------------------

type rhiQueue struct {
	device *device
	handle vkQueue
	kind   rhi.QueueKind
}

func (d *device) GetQueue(kind rhi.QueueKind) (rhi.Queue, error) {
	if d == nil || d.handle == 0 {
		return nil, ErrNotImplemented
	}
	switch kind {
	case rhi.QueueGraphics:
		if d.graphicsQueue == 0 {
			return nil, fmt.Errorf("%w: graphics queue not available", ErrNoQueueFamily)
		}
		return &rhiQueue{device: d, handle: d.graphicsQueue, kind: kind}, nil
	default:
		return nil, fmt.Errorf("%w: queue kind %d not supported yet", ErrNotImplemented, kind)
	}
}

func (q *rhiQueue) Backend() rhi.BackendKind { return rhi.BackendKindVulkan }
func (q *rhiQueue) Kind() rhi.QueueKind      { return q.kind }

func (q *rhiQueue) Submit(cmds []rhi.CommandBuffer) error {
	if q == nil || q.handle == 0 || q.device == nil {
		return ErrNotImplemented
	}
	// TODO: batch multiple command buffers into a single vkQueueSubmit.
	for _, cmd := range cmds {
		vkCmd, ok := cmd.(*rhiCommandBuffer)
		if !ok || vkCmd.handle == 0 {
			return fmt.Errorf("%w: expected Vulkan command buffer", ErrQueueSubmit)
		}
		submitInfo := vkSubmitInfo{
			SType:              vkStructureTypeSubmitInfo,
			CommandBufferCount: 1,
			PCommandBuffers:    &vkCmd.handle,
		}
		if result := q.device.api.queueSubmit(q.handle, 1, &submitInfo, 0); result != vkSuccess {
			return fmt.Errorf("%w: %s", ErrQueueSubmit, result)
		}
	}
	return nil
}

func (q *rhiQueue) WaitIdle() error {
	if q == nil || q.handle == 0 || q.device == nil {
		return nil
	}
	if result := q.device.api.queueWaitIdle(q.handle); result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrQueueSubmit, result)
	}
	return nil
}
