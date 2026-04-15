//go:build darwin

package metal

import "github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"

// TODO(rhi): Implement Metal backend for the generic RHI GPU contract.
//
// The following interfaces need concrete Metal implementations:
//   - rhi.ShaderModule   -- wraps MTLLibrary / MTLFunction
//   - rhi.Pipeline       -- wraps MTLRenderPipelineState
//   - rhi.Buffer         -- wraps MTLBuffer
//   - rhi.Texture        -- wraps MTLTexture
//   - rhi.DescriptorSet  -- Metal uses argument buffers; needs mapping
//   - rhi.CommandBuffer  -- wraps MTLCommandBuffer + MTLRenderCommandEncoder
//   - rhi.Queue          -- wraps MTLCommandQueue
//
// The existing Renderer2D in renderer_darwin.go already has the low-level
// Metal calls. The plan is to:
//   1. Extract resource creation into Device methods
//   2. Wrap each Objective-C handle behind the rhi interface
//   3. Keep Renderer2D working identically during the transition
//
// This file is a placeholder so the package compiles on darwin and other
// agents can see the intended structure.

// metalShaderModule is a placeholder for the future rhi.ShaderModule Metal impl.
type metalShaderModule struct {
	// library objc.ID  -- will hold the MTLLibrary
	stage rhi.ShaderStage
}

func (m *metalShaderModule) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (m *metalShaderModule) Stage() rhi.ShaderStage   { return m.stage }
func (m *metalShaderModule) Destroy() error           { return nil }

// metalPipeline is a placeholder for the future rhi.Pipeline Metal impl.
type metalPipeline struct{}

func (p *metalPipeline) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (p *metalPipeline) Destroy() error           { return nil }

// metalBuffer is a placeholder for the future rhi.Buffer Metal impl.
type metalBuffer struct {
	size  int
	usage rhi.BufferUsage
}

func (b *metalBuffer) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (b *metalBuffer) Size() int                { return b.size }
func (b *metalBuffer) Usage() rhi.BufferUsage   { return b.usage }
func (b *metalBuffer) Destroy() error           { return nil }

// metalTexture is a placeholder for the future rhi.Texture Metal impl.
type metalTexture struct {
	width  int
	height int
	format rhi.PixelFormat
}

func (t *metalTexture) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (t *metalTexture) Width() int               { return t.width }
func (t *metalTexture) Height() int              { return t.height }
func (t *metalTexture) Format() rhi.PixelFormat  { return t.format }
func (t *metalTexture) Destroy() error           { return nil }

// metalDescriptorSet is a placeholder for the future rhi.DescriptorSet Metal impl.
type metalDescriptorSet struct{}

func (ds *metalDescriptorSet) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (ds *metalDescriptorSet) Destroy() error           { return nil }

// metalCommandBuffer is a placeholder for the future rhi.CommandBuffer Metal impl.
type metalCommandBuffer struct{}

func (cb *metalCommandBuffer) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (cb *metalCommandBuffer) Reset() error             { return nil }

// metalQueue is a placeholder for the future rhi.Queue Metal impl.
type metalQueue struct {
	kind rhi.QueueKind
}

func (q *metalQueue) Backend() rhi.BackendKind              { return rhi.BackendKindMetal }
func (q *metalQueue) Kind() rhi.QueueKind                   { return q.kind }
func (q *metalQueue) Submit(_ []rhi.CommandBuffer) error     { return nil }
func (q *metalQueue) WaitIdle() error                       { return nil }
