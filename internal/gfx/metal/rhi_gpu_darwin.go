//go:build darwin

package metal

import (
	"github.com/ebitengine/purego/objc"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
)

// metalShaderModule wraps a compiled MTLFunction.
type metalShaderModule struct {
	library  objc.ID
	function objc.ID
	stage    rhi.ShaderStage
}

func (m *metalShaderModule) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (m *metalShaderModule) Stage() rhi.ShaderStage   { return m.stage }
func (m *metalShaderModule) Destroy() error {
	if m.function != 0 {
		m.function.Send(objc.RegisterName("release"))
		m.function = 0
	}
	if m.library != 0 {
		m.library.Send(objc.RegisterName("release"))
		m.library = 0
	}
	return nil
}

// metalPipeline wraps a MTLRenderPipelineState.
type metalPipeline struct {
	state objc.ID
}

func (p *metalPipeline) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (p *metalPipeline) Destroy() error {
	if p.state != 0 {
		p.state.Send(objc.RegisterName("release"))
		p.state = 0
	}
	return nil
}

// metalBuffer wraps a MTLBuffer.
type metalBuffer struct {
	buffer objc.ID
	size   int
	usage  rhi.BufferUsage
}

func (b *metalBuffer) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (b *metalBuffer) Size() int                { return b.size }
func (b *metalBuffer) Usage() rhi.BufferUsage   { return b.usage }
func (b *metalBuffer) Destroy() error {
	if b.buffer != 0 {
		b.buffer.Send(objc.RegisterName("release"))
		b.buffer = 0
	}
	return nil
}

// metalTexture wraps a MTLTexture.
type metalTexture struct {
	texture objc.ID
	width   int
	height  int
	format  rhi.PixelFormat
}

func (t *metalTexture) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (t *metalTexture) Width() int               { return t.width }
func (t *metalTexture) Height() int              { return t.height }
func (t *metalTexture) Format() rhi.PixelFormat  { return t.format }
func (t *metalTexture) Destroy() error {
	if t.texture != 0 {
		t.texture.Send(objc.RegisterName("release"))
		t.texture = 0
	}
	return nil
}

// metalDescriptorSet is a placeholder; Metal uses argument buffers or direct
// binding in the encoder, so traditional descriptor sets are not needed for
// the 2D renderer path.
type metalDescriptorSet struct{}

func (ds *metalDescriptorSet) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (ds *metalDescriptorSet) Destroy() error           { return nil }

// metalCommandBuffer wraps a MTLCommandBuffer.
type metalCommandBuffer struct {
	cmdBuffer objc.ID
	encoder   objc.ID
}

func (cb *metalCommandBuffer) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (cb *metalCommandBuffer) Reset() error {
	// Reset creates a fresh command buffer from the queue.
	// In practice the renderer creates command buffers directly,
	// so this is a no-op placeholder.
	return nil
}

// metalQueue wraps a MTLCommandQueue.
type metalQueue struct {
	queue objc.ID
	kind  rhi.QueueKind
}

func (q *metalQueue) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (q *metalQueue) Kind() rhi.QueueKind      { return q.kind }
func (q *metalQueue) Submit(cmds []rhi.CommandBuffer) error {
	selCommit := objc.RegisterName("commit")
	for _, c := range cmds {
		if mc, ok := c.(*metalCommandBuffer); ok && mc.cmdBuffer != 0 {
			mc.cmdBuffer.Send(selCommit)
		}
	}
	return nil
}
func (q *metalQueue) WaitIdle() error {
	// Create a dummy command buffer and wait for it.
	selCommandBuffer := objc.RegisterName("commandBuffer")
	selCommit := objc.RegisterName("commit")
	selWaitUntilCompleted := objc.RegisterName("waitUntilCompleted")
	cmdBuffer := objc.Send[objc.ID](q.queue, selCommandBuffer)
	if cmdBuffer == 0 {
		return nil
	}
	cmdBuffer.Send(selCommit)
	cmdBuffer.Send(selWaitUntilCompleted)
	return nil
}
