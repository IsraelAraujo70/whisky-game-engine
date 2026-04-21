//go:build darwin

package metal

import (
	"fmt"

	"github.com/ebitengine/purego/objc"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
)

type metalDevice struct {
	device       objc.ID
	commandQueue objc.ID
}

func (d *metalDevice) Backend() rhi.BackendKind { return rhi.BackendKindMetal }

func (d *metalDevice) CreateSwapchain(surface rhi.Surface, desc rhi.SwapchainDescriptor) (rhi.Swapchain, error) {
	ms, ok := surface.(*metalSurface)
	if !ok {
		return nil, fmt.Errorf("metal: expected *metalSurface, got %T", surface)
	}
	if ms.layer == 0 {
		return nil, fmt.Errorf("metal: surface layer is nil")
	}

	selSetDrawableSize := objc.RegisterName("setDrawableSize:")
	ms.layer.Send(selSetDrawableSize, CGSize{
		Width:  float64(desc.Extent.Width),
		Height: float64(desc.Extent.Height),
	})

	return &metalSwapchain{
		layer:      ms.layer,
		descriptor: desc,
	}, nil
}

func (d *metalDevice) CreateShaderModule(code []byte, stage rhi.ShaderStage) (rhi.ShaderModule, error) {
	selNewLibraryWithSource := objc.RegisterName("newLibraryWithSource:options:error:")
	selNewFunctionWithName := objc.RegisterName("newFunctionWithName:")
	selStringWithUTF8String := objc.RegisterName("stringWithUTF8String:")
	selLocalizedDescription := objc.RegisterName("localizedDescription")
	selUTF8String := objc.RegisterName("UTF8String")

	source := objc.ID(objc.GetClass("NSString")).Send(selStringWithUTF8String, string(code))
	if source == 0 {
		return nil, fmt.Errorf("metal: failed to create shader source string")
	}

	options := objc.ID(objc.GetClass("MTLCompileOptions")).Send(objc.RegisterName("new"))
	if options != 0 {
		defer options.Send(objc.RegisterName("release"))
	}

	var compileError objc.ID
	library := objc.Send[objc.ID](d.device, selNewLibraryWithSource, source, options, &compileError)
	if library == 0 {
		desc := "unknown"
		if compileError != 0 {
			nsDesc := objc.Send[objc.ID](compileError, selLocalizedDescription)
			if nsDesc != 0 {
				desc = objc.Send[string](nsDesc, selUTF8String)
			}
		}
		return nil, fmt.Errorf("metal: shader compilation failed: %s", desc)
	}

	funcName := "whiskyVertex"
	if stage == rhi.ShaderStageFragment {
		funcName = "whiskyFragment"
	}
	fnName := objc.ID(objc.GetClass("NSString")).Send(selStringWithUTF8String, funcName)
	function := objc.Send[objc.ID](library, selNewFunctionWithName, fnName)
	if function == 0 {
		library.Send(objc.RegisterName("release"))
		return nil, fmt.Errorf("metal: function %q not found in library", funcName)
	}

	return &metalShaderModule{
		library:  library,
		function: function,
		stage:    stage,
	}, nil
}

func (d *metalDevice) CreatePipeline(desc rhi.PipelineDescriptor) (rhi.Pipeline, error) {
	selNew := objc.RegisterName("new")
	selSetVertexFunction := objc.RegisterName("setVertexFunction:")
	selSetFragmentFunction := objc.RegisterName("setFragmentFunction:")
	selColorAttachments := objc.RegisterName("colorAttachments")
	selObjectAtIndexedSubscript := objc.RegisterName("objectAtIndexedSubscript:")
	selSetPixelFormat := objc.RegisterName("setPixelFormat:")
	selSetBlendingEnabled := objc.RegisterName("setBlendingEnabled:")
	selSetRGBBlendOperation := objc.RegisterName("setRgbBlendOperation:")
	selSetAlphaBlendOperation := objc.RegisterName("setAlphaBlendOperation:")
	selSetSourceRGBBlendFactor := objc.RegisterName("setSourceRGBBlendFactor:")
	selSetDestinationRGBBlendFactor := objc.RegisterName("setDestinationRGBBlendFactor:")
	selSetSourceAlphaBlendFactor := objc.RegisterName("setSourceAlphaBlendFactor:")
	selSetDestinationAlphaBlendFactor := objc.RegisterName("setDestinationAlphaBlendFactor:")
	selNewRenderPipelineStateWithDescriptor := objc.RegisterName("newRenderPipelineStateWithDescriptor:error:")
	selLocalizedDescription := objc.RegisterName("localizedDescription")
	selUTF8String := objc.RegisterName("UTF8String")
	selRelease := objc.RegisterName("release")

	vertexShader, ok := desc.VertexShader.(*metalShaderModule)
	if !ok {
		return nil, fmt.Errorf("metal: vertex shader is not a metal shader module")
	}
	fragmentShader, ok := desc.FragmentShader.(*metalShaderModule)
	if !ok {
		return nil, fmt.Errorf("metal: fragment shader is not a metal shader module")
	}

	pipelineDesc := objc.ID(objc.GetClass("MTLRenderPipelineDescriptor")).Send(selNew)
	if pipelineDesc == 0 {
		return nil, fmt.Errorf("metal: failed to create pipeline descriptor")
	}
	defer pipelineDesc.Send(selRelease)

	pipelineDesc.Send(selSetVertexFunction, vertexShader.function)
	pipelineDesc.Send(selSetFragmentFunction, fragmentShader.function)

	attachments := objc.Send[objc.ID](pipelineDesc, selColorAttachments)
	attachment := objc.Send[objc.ID](attachments, selObjectAtIndexedSubscript, uintptr(0))
	if attachment != 0 {
		attachment.Send(selSetPixelFormat, uintptr(mtlPixelFormatBGRA8Unorm))
		if desc.Blend == rhi.BlendAlpha {
			attachment.Send(selSetBlendingEnabled, true)
			attachment.Send(selSetRGBBlendOperation, uintptr(mtlBlendOperationAdd))
			attachment.Send(selSetAlphaBlendOperation, uintptr(mtlBlendOperationAdd))
			attachment.Send(selSetSourceRGBBlendFactor, uintptr(mtlBlendFactorSourceAlpha))
			attachment.Send(selSetDestinationRGBBlendFactor, uintptr(mtlBlendFactorOneMinusSourceAlpha))
			attachment.Send(selSetSourceAlphaBlendFactor, uintptr(mtlBlendFactorOne))
			attachment.Send(selSetDestinationAlphaBlendFactor, uintptr(mtlBlendFactorOneMinusSourceAlpha))
		}
	}

	var pipelineError objc.ID
	state := objc.Send[objc.ID](d.device, selNewRenderPipelineStateWithDescriptor, pipelineDesc, &pipelineError)
	if state == 0 {
		desc := "unknown"
		if pipelineError != 0 {
			nsDesc := objc.Send[objc.ID](pipelineError, selLocalizedDescription)
			if nsDesc != 0 {
				desc = objc.Send[string](nsDesc, selUTF8String)
			}
		}
		return nil, fmt.Errorf("metal: create pipeline state: %s", desc)
	}

	return &metalPipeline{state: state}, nil
}

func (d *metalDevice) CreateBuffer(desc rhi.BufferDescriptor) (rhi.Buffer, error) {
	selNewBufferWithLengthOptions := objc.RegisterName("newBufferWithLength:options:")
	buffer := objc.Send[objc.ID](d.device, selNewBufferWithLengthOptions, uintptr(desc.Size), uintptr(mtlStorageModeShared))
	if buffer == 0 {
		return nil, fmt.Errorf("metal: failed to create buffer")
	}
	return &metalBuffer{buffer: buffer, size: desc.Size, usage: desc.Usage}, nil
}

func (d *metalDevice) CreateTexture(desc rhi.TextureDescriptor) (rhi.Texture, error) {
	selTexture2DDescriptorWithPixelFormat := objc.RegisterName("texture2DDescriptorWithPixelFormat:width:height:mipmapped:")
	selSetStorageMode := objc.RegisterName("setStorageMode:")
	selNewTextureWithDescriptor := objc.RegisterName("newTextureWithDescriptor:")

	mtlFormat := mapPixelFormatToMetal(desc.Format)
	if mtlFormat < 0 {
		return nil, fmt.Errorf("metal: unsupported pixel format %q", desc.Format)
	}

	textureDesc := objc.ID(objc.GetClass("MTLTextureDescriptor")).Send(
		selTexture2DDescriptorWithPixelFormat,
		uintptr(mtlFormat),
		uintptr(desc.Width),
		uintptr(desc.Height),
		false,
	)
	if textureDesc == 0 {
		return nil, fmt.Errorf("metal: failed to create texture descriptor")
	}
	textureDesc.Send(selSetStorageMode, uintptr(mtlStorageModeShared))

	texture := objc.Send[objc.ID](d.device, selNewTextureWithDescriptor, textureDesc)
	if texture == 0 {
		return nil, fmt.Errorf("metal: failed to create texture")
	}
	return &metalTexture{texture: texture, width: desc.Width, height: desc.Height, format: desc.Format}, nil
}

func (d *metalDevice) GetQueue(kind rhi.QueueKind) (rhi.Queue, error) {
	_ = kind
	return &metalQueue{queue: d.commandQueue, kind: kind}, nil
}

func (d *metalDevice) WaitIdle() error {
	// Metal has no explicit device wait; we can create a command buffer,
	// commit it, and wait until completed.
	selCommandBuffer := objc.RegisterName("commandBuffer")
	selCommit := objc.RegisterName("commit")
	selWaitUntilCompleted := objc.RegisterName("waitUntilCompleted")
	cmdBuffer := objc.Send[objc.ID](d.commandQueue, selCommandBuffer)
	if cmdBuffer == 0 {
		return nil
	}
	cmdBuffer.Send(selCommit)
	cmdBuffer.Send(selWaitUntilCompleted)
	return nil
}

func (d *metalDevice) Destroy() error {
	if d.commandQueue != 0 {
		selRelease := objc.RegisterName("release")
		d.commandQueue.Send(selRelease)
		d.commandQueue = 0
	}
	return nil
}

type metalSwapchain struct {
	layer      objc.ID
	descriptor rhi.SwapchainDescriptor
}

func (s *metalSwapchain) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (s *metalSwapchain) Descriptor() rhi.SwapchainDescriptor { return s.descriptor }

func (s *metalSwapchain) Resize(width, height int) error {
	s.descriptor.Extent = rhi.Extent2D{Width: width, Height: height}
	if s.layer != 0 {
		selSetDrawableSize := objc.RegisterName("setDrawableSize:")
		s.layer.Send(selSetDrawableSize, CGSize{Width: float64(width), Height: float64(height)})
	}
	return nil
}

func (s *metalSwapchain) Destroy() error {
	return nil
}

func mapPixelFormatToMetal(pf rhi.PixelFormat) int {
	switch pf {
	case rhi.PixelFormatBGRA8Unorm:
		return mtlPixelFormatBGRA8Unorm
	default:
		return -1
	}
}
