//go:build darwin

package metal

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
)

const (
	mtlPixelFormatBGRA8Unorm          = 80
	mtlStorageModeShared              = 0
	mtlPrimitiveTypeTriangle          = 3
	mtlLoadActionClear                = 2
	mtlStoreActionStore               = 1
	mtlBlendOperationAdd              = 0
	mtlBlendFactorOne                 = 1
	mtlBlendFactorSourceAlpha         = 4
	mtlBlendFactorOneMinusSourceAlpha = 5
	mtlSamplerMinMagFilterNearest     = 0
	mtlSamplerAddressModeClampToEdge  = 2
)

type MTLOrigin struct {
	X uint64
	Y uint64
	Z uint64
}

type MTLSize struct {
	Width  uint64
	Height uint64
	Depth  uint64
}

type MTLRegion struct {
	Origin MTLOrigin
	Size   MTLSize
}

type CGSize struct {
	Width  float64
	Height float64
}

type MTLClearColor struct {
	Red   float64
	Green float64
	Blue  float64
	Alpha float64
}

type MTLViewport struct {
	OriginX float64
	OriginY float64
	Width   float64
	Height  float64
	Znear   float64
	Zfar    float64
}

type MTLScissorRect struct {
	X      uint64
	Y      uint64
	Width  uint64
	Height uint64
}

type viewUniforms struct {
	LogicalSize [2]float32
}

type gpuTexture struct {
	width   int
	height  int
	texture objc.ID
}

type Renderer2D struct {
	device         objc.ID
	commandQueue   objc.ID
	layer          objc.ID
	pipelineState  objc.ID
	samplerState   objc.ID
	vertexBuffer   objc.ID
	vertexCapacity int
	software       *softwareRenderer
	texturesBySrc  map[*softwareTexture]*gpuTexture
	targetWidth    int
	targetHeight   int
}

var (
	selReleaseMetal                         = objc.RegisterName("release")
	selNew                                  = objc.RegisterName("new")
	selSetDrawableSize                      = objc.RegisterName("setDrawableSize:")
	selNextDrawable                         = objc.RegisterName("nextDrawable")
	selTexture                              = objc.RegisterName("texture")
	selCommandBuffer                        = objc.RegisterName("commandBuffer")
	selCommit                               = objc.RegisterName("commit")
	selPresentDrawable                      = objc.RegisterName("presentDrawable:")
	selNewTextureWithDescriptor             = objc.RegisterName("newTextureWithDescriptor:")
	selTexture2DDescriptorWithPixelFormat   = objc.RegisterName("texture2DDescriptorWithPixelFormat:width:height:mipmapped:")
	selSetStorageMode                       = objc.RegisterName("setStorageMode:")
	selReplaceRegion                        = objc.RegisterName("replaceRegion:mipmapLevel:withBytes:bytesPerRow:")
	selNewLibraryWithSource                 = objc.RegisterName("newLibraryWithSource:options:error:")
	selNewFunctionWithName                  = objc.RegisterName("newFunctionWithName:")
	selNewRenderPipelineStateWithDescriptor = objc.RegisterName("newRenderPipelineStateWithDescriptor:error:")
	selNewBufferWithLengthOptions           = objc.RegisterName("newBufferWithLength:options:")
	selContents                             = objc.RegisterName("contents")
	selRenderPassDescriptor                 = objc.RegisterName("renderPassDescriptor")
	selColorAttachments                     = objc.RegisterName("colorAttachments")
	selObjectAtIndexedSubscript             = objc.RegisterName("objectAtIndexedSubscript:")
	selSetTextureAttachment                 = objc.RegisterName("setTexture:")
	selSetLoadAction                        = objc.RegisterName("setLoadAction:")
	selSetStoreAction                       = objc.RegisterName("setStoreAction:")
	selSetClearColor                        = objc.RegisterName("setClearColor:")
	selRenderCommandEncoderWithDescriptor   = objc.RegisterName("renderCommandEncoderWithDescriptor:")
	selSetViewport                          = objc.RegisterName("setViewport:")
	selSetScissorRect                       = objc.RegisterName("setScissorRect:")
	selSetRenderPipelineState               = objc.RegisterName("setRenderPipelineState:")
	selSetVertexBufferOffsetAtIndex         = objc.RegisterName("setVertexBuffer:offset:atIndex:")
	selSetVertexBytesLengthAtIndex          = objc.RegisterName("setVertexBytes:length:atIndex:")
	selSetFragmentTextureAtIndex            = objc.RegisterName("setFragmentTexture:atIndex:")
	selSetFragmentSamplerStateAtIndex       = objc.RegisterName("setFragmentSamplerState:atIndex:")
	selDrawPrimitivesVertexStartVertexCount = objc.RegisterName("drawPrimitives:vertexStart:vertexCount:")
	selEndEncoding                          = objc.RegisterName("endEncoding")
	selSetVertexFunction                    = objc.RegisterName("setVertexFunction:")
	selSetFragmentFunction                  = objc.RegisterName("setFragmentFunction:")
	selSetBlendingEnabled                   = objc.RegisterName("setBlendingEnabled:")
	selSetRGBBlendOperation                 = objc.RegisterName("setRgbBlendOperation:")
	selSetAlphaBlendOperation               = objc.RegisterName("setAlphaBlendOperation:")
	selSetSourceRGBBlendFactor              = objc.RegisterName("setSourceRGBBlendFactor:")
	selSetDestinationRGBBlendFactor         = objc.RegisterName("setDestinationRGBBlendFactor:")
	selSetSourceAlphaBlendFactor            = objc.RegisterName("setSourceAlphaBlendFactor:")
	selSetDestinationAlphaBlendFactor       = objc.RegisterName("setDestinationAlphaBlendFactor:")
	selSetPixelFormat                       = objc.RegisterName("setPixelFormat:")
	selNewSamplerStateWithDescriptor        = objc.RegisterName("newSamplerStateWithDescriptor:")
	selSetMinFilter                         = objc.RegisterName("setMinFilter:")
	selSetMagFilter                         = objc.RegisterName("setMagFilter:")
	selSetSAddressMode                      = objc.RegisterName("setSAddressMode:")
	selSetTAddressMode                      = objc.RegisterName("setTAddressMode:")
	selLocalizedDescription                 = objc.RegisterName("localizedDescription")
	selUTF8String                           = objc.RegisterName("UTF8String")
	selStringWithUTF8String                 = objc.RegisterName("stringWithUTF8String:")
)

const metalShaderSource = `#include <metal_stdlib>
using namespace metal;

struct QuadVertex {
    float2 position;
    float2 uv;
    float4 color;
};

struct ViewUniforms {
    float2 logicalSize;
};

struct VertexOut {
    float4 position [[position]];
    float2 uv;
    float4 color;
};

vertex VertexOut whiskyVertex(uint vertexID [[vertex_id]],
                              constant QuadVertex *vertices [[buffer(0)]],
                              constant ViewUniforms &view [[buffer(1)]]) {
    QuadVertex inVertex = vertices[vertexID];
    float2 logicalSize = max(view.logicalSize, float2(1.0, 1.0));
    float2 ndc = float2(
        (inVertex.position.x / logicalSize.x) * 2.0 - 1.0,
        1.0 - (inVertex.position.y / logicalSize.y) * 2.0
    );

    VertexOut outVertex;
    outVertex.position = float4(ndc, 0.0, 1.0);
    outVertex.uv = inVertex.uv;
    outVertex.color = inVertex.color;
    return outVertex;
}

fragment float4 whiskyFragment(VertexOut inVertex [[stage_in]],
                               texture2d<float> texture0 [[texture(0)]],
                               sampler sampler0 [[sampler(0)]]) {
    return texture0.sample(sampler0, inVertex.uv) * inVertex.color;
}
`

func NewRenderer2D(deviceValue rhi.Device, swapchainValue rhi.Swapchain) (*Renderer2D, error) {
	dev, ok := deviceValue.(*metalDevice)
	if !ok {
		return nil, fmt.Errorf("metal: expected *metalDevice, got %T", deviceValue)
	}
	swc, ok := swapchainValue.(*metalSwapchain)
	if !ok {
		return nil, fmt.Errorf("metal: expected *metalSwapchain, got %T", swapchainValue)
	}
	if dev.device == 0 {
		return nil, fmt.Errorf("metal: device is nil")
	}
	if swc.layer == 0 {
		return nil, fmt.Errorf("metal: swapchain layer is nil")
	}

	software, err := newSoftwareRenderer()
	if err != nil {
		return nil, err
	}

	renderer := &Renderer2D{
		device:        dev.device,
		commandQueue:  dev.commandQueue,
		layer:         swc.layer,
		software:      software,
		texturesBySrc: map[*softwareTexture]*gpuTexture{},
	}
	if err := renderer.createPipelineState(); err != nil {
		_ = renderer.Destroy()
		return nil, err
	}
	if err := renderer.createSamplerState(); err != nil {
		_ = renderer.Destroy()
		return nil, err
	}
	if _, err := renderer.ensureTextureUploaded(renderer.software.whiteTexture); err != nil {
		_ = renderer.Destroy()
		return nil, err
	}
	if renderer.software.debugFont != nil && renderer.software.debugFont.texture != nil {
		if _, err := renderer.ensureTextureUploaded(renderer.software.debugFont.texture); err != nil {
			_ = renderer.Destroy()
			return nil, err
		}
	}
	if err := renderer.resizeResources(); err != nil {
		_ = renderer.Destroy()
		return nil, err
	}
	runtime.SetFinalizer(renderer, func(r *Renderer2D) {
		_ = r.Destroy()
	})
	return renderer, nil
}

func (r *Renderer2D) LoadTexture(path string) (render.TextureID, int, int, error) {
	if r == nil || r.software == nil {
		return 0, 0, 0, fmt.Errorf("metal: renderer is nil")
	}
	return r.software.loadTexture(path)
}

func (r *Renderer2D) SetLogicalSize(width, height int, pixelPerfect bool) error {
	if r == nil || r.software == nil {
		return nil
	}
	r.software.setLogicalSize(width, height, pixelPerfect)
	return nil
}

func (r *Renderer2D) DrawFrame(clearColor geom.Color, cmds []render.DrawCmd, lines []string) error {
	if r == nil || r.software == nil || r.layer == 0 || r.commandQueue == 0 || r.pipelineState == 0 {
		return nil
	}
	if err := r.resizeResources(); err != nil {
		return err
	}
	if r.targetWidth <= 0 || r.targetHeight <= 0 {
		return nil
	}
	vertices, batches, logicalWidth, logicalHeight, err := r.software.buildDrawData(cmds, lines, r.targetWidth, r.targetHeight)
	if err != nil {
		return err
	}
	if err := r.uploadVertexData(vertices); err != nil {
		return err
	}
	return r.encodeFrame(clearColor, batches, logicalWidth, logicalHeight)
}

func (r *Renderer2D) Destroy() error {
	if r == nil {
		return nil
	}
	r.releaseTextures()
	if r.vertexBuffer != 0 {
		r.vertexBuffer.Send(selReleaseMetal)
		r.vertexBuffer = 0
	}
	if r.samplerState != 0 {
		r.samplerState.Send(selReleaseMetal)
		r.samplerState = 0
	}
	if r.pipelineState != 0 {
		r.pipelineState.Send(selReleaseMetal)
		r.pipelineState = 0
	}
	// layer, commandQueue and device are owned by RHI objects; do not release here.
	r.vertexCapacity = 0
	r.device = 0
	r.commandQueue = 0
	r.layer = 0
	r.software = nil
	runtime.SetFinalizer(r, nil)
	return nil
}

func (r *Renderer2D) resizeResources() error {
	if r == nil || r.layer == 0 {
		return nil
	}
	// Size is managed by the swapchain; the backend calls swapchain.Resize()
	// before DrawFrame when the window size changes. We just read the current
	// drawable size from the layer to stay in sync.
	selDrawableSize := objc.RegisterName("drawableSize")
	size := objc.Send[CGSize](r.layer, selDrawableSize)
	width := int(size.Width)
	height := int(size.Height)
	if width <= 0 || height <= 0 {
		r.targetWidth = 0
		r.targetHeight = 0
		return nil
	}
	if width != r.targetWidth || height != r.targetHeight {
		r.targetWidth = width
		r.targetHeight = height
	}
	return nil
}

func (r *Renderer2D) createPipelineState() error {
	shaderSource := objc.ID(objc.GetClass("NSString")).Send(selStringWithUTF8String, metalShaderSource)
	if shaderSource == 0 {
		return fmt.Errorf("metal: failed to allocate shader source string")
	}
	var shaderError objc.ID
	library := objc.Send[objc.ID](r.device, selNewLibraryWithSource, shaderSource, objc.ID(0), &shaderError)
	if library == 0 {
		return fmt.Errorf("metal: compile shader library: %s", nsErrorDescription(shaderError))
	}
	defer library.Send(selReleaseMetal)

	vertexName := objc.ID(objc.GetClass("NSString")).Send(selStringWithUTF8String, "whiskyVertex")
	fragmentName := objc.ID(objc.GetClass("NSString")).Send(selStringWithUTF8String, "whiskyFragment")
	vertexFunction := objc.Send[objc.ID](library, selNewFunctionWithName, vertexName)
	if vertexFunction == 0 {
		return fmt.Errorf("metal: failed to create vertex function")
	}
	defer vertexFunction.Send(selReleaseMetal)
	fragmentFunction := objc.Send[objc.ID](library, selNewFunctionWithName, fragmentName)
	if fragmentFunction == 0 {
		return fmt.Errorf("metal: failed to create fragment function")
	}
	defer fragmentFunction.Send(selReleaseMetal)

	descriptor := objc.ID(objc.GetClass("MTLRenderPipelineDescriptor")).Send(selNew)
	if descriptor == 0 {
		return fmt.Errorf("metal: failed to create pipeline descriptor")
	}
	defer descriptor.Send(selReleaseMetal)
	descriptor.Send(selSetVertexFunction, vertexFunction)
	descriptor.Send(selSetFragmentFunction, fragmentFunction)
	attachments := objc.Send[objc.ID](descriptor, selColorAttachments)
	attachment := objc.Send[objc.ID](attachments, selObjectAtIndexedSubscript, uintptr(0))
	if attachment == 0 {
		return fmt.Errorf("metal: failed to access pipeline color attachment")
	}
	attachment.Send(selSetPixelFormat, uintptr(mtlPixelFormatBGRA8Unorm))
	attachment.Send(selSetBlendingEnabled, true)
	attachment.Send(selSetRGBBlendOperation, uintptr(mtlBlendOperationAdd))
	attachment.Send(selSetAlphaBlendOperation, uintptr(mtlBlendOperationAdd))
	attachment.Send(selSetSourceRGBBlendFactor, uintptr(mtlBlendFactorSourceAlpha))
	attachment.Send(selSetDestinationRGBBlendFactor, uintptr(mtlBlendFactorOneMinusSourceAlpha))
	attachment.Send(selSetSourceAlphaBlendFactor, uintptr(mtlBlendFactorOne))
	attachment.Send(selSetDestinationAlphaBlendFactor, uintptr(mtlBlendFactorOneMinusSourceAlpha))

	var pipelineError objc.ID
	pipelineState := objc.Send[objc.ID](r.device, selNewRenderPipelineStateWithDescriptor, descriptor, &pipelineError)
	if pipelineState == 0 {
		return fmt.Errorf("metal: create pipeline state: %s", nsErrorDescription(pipelineError))
	}
	r.pipelineState = pipelineState
	return nil
}

func (r *Renderer2D) createSamplerState() error {
	descriptor := objc.ID(objc.GetClass("MTLSamplerDescriptor")).Send(selNew)
	if descriptor == 0 {
		return fmt.Errorf("metal: failed to create sampler descriptor")
	}
	defer descriptor.Send(selReleaseMetal)
	descriptor.Send(selSetMinFilter, uintptr(mtlSamplerMinMagFilterNearest))
	descriptor.Send(selSetMagFilter, uintptr(mtlSamplerMinMagFilterNearest))
	descriptor.Send(selSetSAddressMode, uintptr(mtlSamplerAddressModeClampToEdge))
	descriptor.Send(selSetTAddressMode, uintptr(mtlSamplerAddressModeClampToEdge))
	samplerState := objc.Send[objc.ID](r.device, selNewSamplerStateWithDescriptor, descriptor)
	if samplerState == 0 {
		return fmt.Errorf("metal: failed to create sampler state")
	}
	r.samplerState = samplerState
	return nil
}

func (r *Renderer2D) ensureTextureUploaded(source *softwareTexture) (*gpuTexture, error) {
	if source == nil || source.rgba == nil {
		return nil, fmt.Errorf("metal: source texture is nil")
	}
	if texture, ok := r.texturesBySrc[source]; ok && texture != nil && texture.texture != 0 {
		return texture, nil
	}
	descriptor := objc.ID(objc.GetClass("MTLTextureDescriptor")).Send(
		selTexture2DDescriptorWithPixelFormat,
		uintptr(mtlPixelFormatBGRA8Unorm),
		uintptr(source.width),
		uintptr(source.height),
		false,
	)
	if descriptor == 0 {
		return nil, fmt.Errorf("metal: failed to create texture descriptor")
	}
	descriptor.Send(selSetStorageMode, uintptr(mtlStorageModeShared))
	textureID := objc.Send[objc.ID](r.device, selNewTextureWithDescriptor, descriptor)
	if textureID == 0 {
		return nil, fmt.Errorf("metal: failed to create GPU texture")
	}
	region := MTLRegion{Origin: MTLOrigin{}, Size: MTLSize{Width: uint64(source.width), Height: uint64(source.height), Depth: 1}}
	textureID.Send(selReplaceRegion, region, uintptr(0), unsafe.Pointer(&source.rgba.Pix[0]), uintptr(source.rgba.Stride))
	texture := &gpuTexture{width: source.width, height: source.height, texture: textureID}
	r.texturesBySrc[source] = texture
	return texture, nil
}

func (r *Renderer2D) releaseTextures() {
	for source, texture := range r.texturesBySrc {
		if texture != nil && texture.texture != 0 {
			texture.texture.Send(selReleaseMetal)
			texture.texture = 0
		}
		delete(r.texturesBySrc, source)
	}
}

func (r *Renderer2D) uploadVertexData(vertices []quadVertex) error {
	if len(vertices) == 0 {
		return nil
	}
	if err := r.ensureVertexBufferCapacity(len(vertices)); err != nil {
		return err
	}
	contents := objc.Send[unsafe.Pointer](r.vertexBuffer, selContents)
	if contents == nil {
		return fmt.Errorf("metal: vertex buffer contents is nil")
	}
	byteLen := len(vertices) * int(unsafe.Sizeof(quadVertex{}))
	dst := unsafe.Slice((*byte)(contents), byteLen)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&vertices[0])), byteLen)
	copy(dst, src)
	return nil
}

func (r *Renderer2D) ensureVertexBufferCapacity(vertexCount int) error {
	if vertexCount <= r.vertexCapacity && r.vertexBuffer != 0 {
		return nil
	}
	capacity := nextVertexCapacity(r.vertexCapacity, vertexCount)
	buffer := objc.Send[objc.ID](r.device, selNewBufferWithLengthOptions, uintptr(capacity*int(unsafe.Sizeof(quadVertex{}))), uintptr(mtlStorageModeShared))
	if buffer == 0 {
		return fmt.Errorf("metal: failed to create vertex buffer")
	}
	if r.vertexBuffer != 0 {
		r.vertexBuffer.Send(selReleaseMetal)
	}
	r.vertexBuffer = buffer
	r.vertexCapacity = capacity
	return nil
}

func nextVertexCapacity(current, required int) int {
	if required <= 0 {
		return current
	}
	capacity := current
	if capacity < 256 {
		capacity = 256
	}
	for capacity < required {
		capacity *= 2
	}
	return capacity
}

func (r *Renderer2D) encodeFrame(clearColor geom.Color, batches []drawBatch, logicalWidth, logicalHeight int) error {
	drawable := objc.Send[objc.ID](r.layer, selNextDrawable)
	if drawable == 0 {
		return nil
	}
	drawableTexture := objc.Send[objc.ID](drawable, selTexture)
	if drawableTexture == 0 {
		return nil
	}
	commandBuffer := objc.Send[objc.ID](r.commandQueue, selCommandBuffer)
	if commandBuffer == 0 {
		return fmt.Errorf("metal: failed to allocate command buffer")
	}
	passDescriptor := objc.ID(objc.GetClass("MTLRenderPassDescriptor")).Send(selRenderPassDescriptor)
	if passDescriptor == 0 {
		return fmt.Errorf("metal: failed to create render pass descriptor")
	}
	attachments := objc.Send[objc.ID](passDescriptor, selColorAttachments)
	attachment := objc.Send[objc.ID](attachments, selObjectAtIndexedSubscript, uintptr(0))
	if attachment == 0 {
		return fmt.Errorf("metal: failed to access render pass attachment")
	}
	attachment.Send(selSetTextureAttachment, drawableTexture)
	attachment.Send(selSetLoadAction, uintptr(mtlLoadActionClear))
	attachment.Send(selSetStoreAction, uintptr(mtlStoreActionStore))
	attachment.Send(selSetClearColor, MTLClearColor{Red: float64(clearColor.R), Green: float64(clearColor.G), Blue: float64(clearColor.B), Alpha: float64(clearColor.A)})

	encoder := objc.Send[objc.ID](commandBuffer, selRenderCommandEncoderWithDescriptor, passDescriptor)
	if encoder == 0 {
		return fmt.Errorf("metal: failed to create render command encoder")
	}
	viewport, scissor := r.computeViewport(logicalWidth, logicalHeight)
	encoder.Send(selSetViewport, viewport)
	encoder.Send(selSetScissorRect, scissor)
	encoder.Send(selSetRenderPipelineState, r.pipelineState)
	uniforms := viewUniforms{LogicalSize: [2]float32{float32(maxInt(logicalWidth, 1)), float32(maxInt(logicalHeight, 1))}}
	encoder.Send(selSetVertexBytesLengthAtIndex, unsafe.Pointer(&uniforms), uintptr(unsafe.Sizeof(uniforms)), uintptr(1))
	if len(batches) > 0 {
		encoder.Send(selSetVertexBufferOffsetAtIndex, r.vertexBuffer, uintptr(0), uintptr(0))
		encoder.Send(selSetFragmentSamplerStateAtIndex, r.samplerState, uintptr(0))
		var boundTexture objc.ID
		for _, batch := range batches {
			if batch.texture == nil || batch.vertexCount == 0 {
				continue
			}
			texture, err := r.ensureTextureUploaded(batch.texture)
			if err != nil {
				encoder.Send(selEndEncoding)
				return err
			}
			if texture.texture != boundTexture {
				encoder.Send(selSetFragmentTextureAtIndex, texture.texture, uintptr(0))
				boundTexture = texture.texture
			}
			encoder.Send(selDrawPrimitivesVertexStartVertexCount, uintptr(mtlPrimitiveTypeTriangle), uintptr(batch.firstVertex), uintptr(batch.vertexCount))
		}
	}
	encoder.Send(selEndEncoding)
	commandBuffer.Send(selPresentDrawable, drawable)
	commandBuffer.Send(selCommit)
	return nil
}

func (r *Renderer2D) computeViewport(logicalWidth, logicalHeight int) (MTLViewport, MTLScissorRect) {
	layout := computePresentationLayout(r.targetWidth, r.targetHeight, logicalWidth, logicalHeight, r.software.pixelPerfect)
	return MTLViewport{
			OriginX: layout.viewportX,
			OriginY: layout.viewportY,
			Width:   layout.viewportWidth,
			Height:  layout.viewportHeight,
			Znear:   0,
			Zfar:    1,
		}, MTLScissorRect{
			X:      layout.scissorX,
			Y:      layout.scissorY,
			Width:  layout.scissorWidth,
			Height: layout.scissorHeight,
		}
}

func nsErrorDescription(errObj objc.ID) string {
	if errObj == 0 {
		return "unknown error"
	}
	description := objc.Send[objc.ID](errObj, selLocalizedDescription)
	if description == 0 {
		return "unknown error"
	}
	message := objc.Send[string](description, selUTF8String)
	if message == "" {
		return "unknown error"
	}
	return message
}
