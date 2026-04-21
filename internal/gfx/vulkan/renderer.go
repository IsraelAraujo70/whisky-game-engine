package vulkan

import (
	_ "embed"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
)

//go:embed shaders/quad.vert.spv
var quadVertexSPV []byte

//go:embed shaders/quad.frag.spv
var quadFragmentSPV []byte

const (
	rendererDescriptorCapacity = 1024
	initialVertexCapacity      = 6 * 2048
	textOverlayMargin          = 3
	textOverlayPadding         = 2
	textOverlayScale           = 0.75
	textOverlayMaxWidthRatio   = 0.52

	vkFormatR32G32Sfloat       = 103
	vkFormatR32G32B32A32Sfloat = 109

	vkVertexInputRateVertex = 0
)

type Renderer2D struct {
	device    *device
	swapchain *swapchain

	texturesByPath map[string]*gpuTexture
	texturesByID   map[render.TextureID]*gpuTexture
	nextTextureID  render.TextureID
	whiteTexture   *gpuTexture
	debugFont      *bitmapFont

	descriptorSetLayout vkDescriptorSetLayout
	descriptorPool      vkDescriptorPool
	sampler             vkSampler

	commandPool    vkCommandPool
	commandBuffers []vkCommandBuffer

	imageAvailable []vkSemaphore
	renderFinished []vkSemaphore
	inFlightFences []vkFence
	imagesInFlight []vkFence
	currentFrame   int

	swapchainImages      []vkImage
	swapchainImageViews  []vkImageView
	renderPass           vkRenderPass
	framebuffers         []vkFramebuffer
	pipelineLayout       vkPipelineLayout
	pipeline             vkPipeline
	vertexBuffers        []gpuBuffer
	swapchainImageFormat int32
	swapchainExtent      vkExtent2D

	virtualWidth  int
	virtualHeight int
	pixelPerfect  bool
}

type gpuTexture struct {
	id            render.TextureID
	width         int
	height        int
	image         vkImage
	memory        vkDeviceMemory
	view          vkImageView
	descriptorSet vkDescriptorSet
}

type gpuBuffer struct {
	buffer    vkBuffer
	memory    vkDeviceMemory
	size      vkDeviceSize
	capacity  int
	mappedPtr unsafe.Pointer
}

type quadVertex struct {
	Position [2]float32
	UV       [2]float32
	Color    [4]float32
}

type drawBatch struct {
	texture     *gpuTexture
	firstVertex uint32
	vertexCount uint32
}

func NewRenderer2D(deviceValue rhi.Device, swapchainValue rhi.Swapchain) (*Renderer2D, error) {
	device, err := requireDevice(deviceValue)
	if err != nil {
		return nil, err
	}
	swapchain, err := requireSwapchain(swapchainValue)
	if err != nil {
		return nil, err
	}

	renderer := &Renderer2D{
		device:         device,
		swapchain:      swapchain,
		texturesByPath: make(map[string]*gpuTexture),
		texturesByID:   make(map[render.TextureID]*gpuTexture),
	}

	if err := renderer.createDescriptorResources(); err != nil {
		renderer.Destroy()
		return nil, err
	}
	if err := renderer.createSwapchainResources(); err != nil {
		renderer.Destroy()
		return nil, err
	}
	whiteTexture, err := renderer.createTextureFromRGBA(onePixelWhite(), 1, 1)
	if err != nil {
		renderer.Destroy()
		return nil, err
	}
	renderer.whiteTexture = whiteTexture
	debugFont, err := renderer.createDebugFont()
	if err != nil {
		renderer.Destroy()
		return nil, err
	}
	renderer.debugFont = debugFont

	runtime.SetFinalizer(renderer, func(r *Renderer2D) {
		_ = r.Destroy()
	})
	return renderer, nil
}

func (r *Renderer2D) SetLogicalSize(width, height int, pixelPerfect bool) {
	r.virtualWidth = width
	r.virtualHeight = height
	r.pixelPerfect = pixelPerfect
}

func (r *Renderer2D) LoadTexture(path string) (render.TextureID, int, int, error) {
	if r == nil {
		return 0, 0, 0, fmt.Errorf("%w: renderer is nil", ErrCreateImage)
	}
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return 0, 0, 0, err
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return 0, 0, 0, err
	}
	defer file.Close()

	src, err := png.Decode(file)
	if err != nil {
		return 0, 0, 0, err
	}
	rgba := imageToRGBA(src)

	if old, ok := r.texturesByPath[cleanPath]; ok {
		// Re-upload: replace GPU memory in-place preserving TextureID.
		newTex, err := r.createTextureFromRGBA(rgba.Pix, rgba.Bounds().Dx(), rgba.Bounds().Dy())
		if err != nil {
			return 0, 0, 0, err
		}
		newTex.id = old.id
		newTex.width = rgba.Bounds().Dx()
		newTex.height = rgba.Bounds().Dy()
		old.destroy(r.device, r.descriptorPool)
		r.texturesByPath[cleanPath] = newTex
		r.texturesByID[newTex.id] = newTex
		return newTex.id, newTex.width, newTex.height, nil
	}

	texture, err := r.createTextureFromRGBA(rgba.Pix, rgba.Bounds().Dx(), rgba.Bounds().Dy())
	if err != nil {
		return 0, 0, 0, err
	}
	r.nextTextureID++
	texture.id = r.nextTextureID
	r.texturesByPath[cleanPath] = texture
	r.texturesByID[texture.id] = texture
	return texture.id, texture.width, texture.height, nil
}

func (r *Renderer2D) DrawFrame(clearColor geom.Color, cmds []render.DrawCmd, lines []string) error {
	if r == nil {
		return nil
	}
	if err := r.ensureSwapchainResources(); err != nil {
		return err
	}

	vertices, batches := r.buildDrawData(cmds, lines)
	frameIndex := r.currentFrame
	if err := r.waitForFence(r.inFlightFences[frameIndex]); err != nil {
		return err
	}

	var imageIndex uint32
	result := r.device.api.acquireNextImageKHR(r.device.handle, r.swapchain.handle, ^uint64(0), r.imageAvailable[frameIndex], 0, &imageIndex)
	switch result {
	case vkSuccess:
	case vkSuboptimalKHR, vkErrorOutOfDateKHR:
		return r.createSwapchainResources()
	default:
		return fmt.Errorf("%w: %s", ErrAcquireImage, result)
	}

	if imageIndex < uint32(len(r.imagesInFlight)) && r.imagesInFlight[imageIndex] != 0 {
		if err := r.waitForFence(r.imagesInFlight[imageIndex]); err != nil {
			return err
		}
	}
	r.imagesInFlight[imageIndex] = r.inFlightFences[frameIndex]

	if err := r.resetFence(r.inFlightFences[frameIndex]); err != nil {
		return err
	}
	if err := r.ensureVertexBuffer(frameIndex, len(vertices)); err != nil {
		return err
	}
	if err := r.uploadVertices(frameIndex, vertices); err != nil {
		return err
	}
	if err := r.recordCommandBuffer(r.commandBuffers[frameIndex], imageIndex, clearColor, batches); err != nil {
		return err
	}
	if err := r.submitFrame(frameIndex); err != nil {
		return err
	}
	if err := r.presentFrame(frameIndex, imageIndex); err != nil {
		if err == errSwapchainNeedsRebuild {
			return r.createSwapchainResources()
		}
		return err
	}

	r.currentFrame = (r.currentFrame + 1) % len(r.commandBuffers)
	return nil
}

func (r *Renderer2D) Destroy() error {
	if r == nil {
		return nil
	}
	if r.device != nil {
		_ = r.device.WaitIdle()
	}

	r.destroyTextures()
	r.destroySwapchainResources()

	if r.sampler != 0 {
		r.device.api.destroySampler(r.device.handle, r.sampler, nil)
		r.sampler = 0
	}
	if r.descriptorPool != 0 {
		r.device.api.destroyDescriptorPool(r.device.handle, r.descriptorPool, nil)
		r.descriptorPool = 0
	}
	if r.descriptorSetLayout != 0 {
		r.device.api.destroyDescriptorSetLayout(r.device.handle, r.descriptorSetLayout, nil)
		r.descriptorSetLayout = 0
	}

	runtime.SetFinalizer(r, nil)
	return nil
}

func (r *Renderer2D) ensureSwapchainResources() error {
	desc := r.swapchain.desc
	format, err := mapPixelFormatToVulkan(desc.Format)
	if err != nil {
		return err
	}
	extent := vkExtent2D{Width: uint32(desc.Extent.Width), Height: uint32(desc.Extent.Height)}
	if len(r.framebuffers) == len(r.swapchainImages) && len(r.framebuffers) > 0 &&
		r.swapchainImageFormat == format &&
		r.swapchainExtent == extent {
		return nil
	}
	return r.createSwapchainResources()
}

func (r *Renderer2D) createDescriptorResources() error {
	if err := r.createDescriptorSetLayout(); err != nil {
		return err
	}
	if err := r.createDescriptorPool(); err != nil {
		return err
	}
	if err := r.createSampler(); err != nil {
		return err
	}
	return nil
}

func (r *Renderer2D) createSwapchainResources() error {
	if r.device == nil || r.swapchain == nil {
		return nil
	}
	if err := r.device.WaitIdle(); err != nil {
		return err
	}

	r.destroySwapchainResources()

	images, err := r.getSwapchainImages()
	if err != nil {
		return err
	}
	r.swapchainImages = images
	r.imagesInFlight = make([]vkFence, len(images))

	format, err := mapPixelFormatToVulkan(r.swapchain.desc.Format)
	if err != nil {
		return err
	}
	r.swapchainImageFormat = format
	r.swapchainExtent = vkExtent2D{Width: uint32(r.swapchain.desc.Extent.Width), Height: uint32(r.swapchain.desc.Extent.Height)}

	if err := r.createImageViews(); err != nil {
		return err
	}
	if err := r.createRenderPass(); err != nil {
		return err
	}
	if err := r.createFramebuffers(); err != nil {
		return err
	}
	if err := r.createCommandPool(); err != nil {
		return err
	}
	if err := r.allocateCommandBuffers(); err != nil {
		return err
	}
	if err := r.createSyncObjects(); err != nil {
		return err
	}
	if err := r.createVertexBuffers(); err != nil {
		return err
	}
	if err := r.createGraphicsPipeline(); err != nil {
		return err
	}

	r.currentFrame = 0
	return nil
}

func (r *Renderer2D) destroySwapchainResources() {
	r.destroyVertexBuffers()
	r.destroySyncObjects()

	if r.commandPool != 0 {
		r.device.api.destroyCommandPool(r.device.handle, r.commandPool, nil)
		r.commandPool = 0
	}
	r.commandBuffers = nil

	for _, framebuffer := range r.framebuffers {
		if framebuffer != 0 {
			r.device.api.destroyFramebuffer(r.device.handle, framebuffer, nil)
		}
	}
	r.framebuffers = nil

	if r.pipeline != 0 {
		r.device.api.destroyPipeline(r.device.handle, r.pipeline, nil)
		r.pipeline = 0
	}
	if r.pipelineLayout != 0 {
		r.device.api.destroyPipelineLayout(r.device.handle, r.pipelineLayout, nil)
		r.pipelineLayout = 0
	}
	if r.renderPass != 0 {
		r.device.api.destroyRenderPass(r.device.handle, r.renderPass, nil)
		r.renderPass = 0
	}
	for _, view := range r.swapchainImageViews {
		if view != 0 {
			r.device.api.destroyImageView(r.device.handle, view, nil)
		}
	}
	r.swapchainImageViews = nil
	r.swapchainImages = nil
	r.imagesInFlight = nil
}

func (r *Renderer2D) createDescriptorSetLayout() error {
	binding := vkDescriptorSetLayoutBinding{
		Binding:         0,
		DescriptorType:  vkDescriptorTypeCombinedImageSampler,
		DescriptorCount: 1,
		StageFlags:      vkShaderStageFragmentBit,
	}
	createInfo := vkDescriptorSetLayoutCreateInfo{
		SType:        vkStructureTypeDescriptorSetLayoutCreateInfo,
		BindingCount: 1,
		PBindings:    &binding,
	}
	result := r.device.api.createDescriptorSetLayout(r.device.handle, &createInfo, nil, &r.descriptorSetLayout)
	if result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrCreateDescriptorLayout, result)
	}
	return nil
}

func (r *Renderer2D) createDescriptorPool() error {
	poolSize := vkDescriptorPoolSize{
		Type:            vkDescriptorTypeCombinedImageSampler,
		DescriptorCount: rendererDescriptorCapacity,
	}
	createInfo := vkDescriptorPoolCreateInfo{
		SType:         vkStructureTypeDescriptorPoolCreateInfo,
		Flags:         vkDescriptorPoolCreateFreeDescriptorSetBit,
		MaxSets:       rendererDescriptorCapacity,
		PoolSizeCount: 1,
		PPoolSizes:    &poolSize,
	}
	result := r.device.api.createDescriptorPool(r.device.handle, &createInfo, nil, &r.descriptorPool)
	if result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrCreateDescriptorPool, result)
	}
	return nil
}

func (r *Renderer2D) createSampler() error {
	createInfo := vkSamplerCreateInfo{
		SType:                   vkStructureTypeSamplerCreateInfo,
		MagFilter:               vkFilterNearest,
		MinFilter:               vkFilterNearest,
		MipmapMode:              vkSamplerMipmapModeNearest,
		AddressModeU:            vkSamplerAddressModeClampToEdge,
		AddressModeV:            vkSamplerAddressModeClampToEdge,
		AddressModeW:            vkSamplerAddressModeClampToEdge,
		MaxAnisotropy:           1,
		CompareOp:               7,
		MaxLod:                  0,
		BorderColor:             vkBorderColorIntOpaqueBlack,
		UnnormalizedCoordinates: 0,
	}
	result := r.device.api.createSampler(r.device.handle, &createInfo, nil, &r.sampler)
	if result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrCreateSampler, result)
	}
	return nil
}

func (r *Renderer2D) getSwapchainImages() ([]vkImage, error) {
	var count uint32
	result := r.device.api.getSwapchainImagesKHR(r.device.handle, r.swapchain.handle, &count, nil)
	if result != vkSuccess {
		return nil, fmt.Errorf("%w: swapchain image count: %s", ErrCreateSwapchain, result)
	}
	if count == 0 {
		return nil, fmt.Errorf("%w: swapchain reported zero images", ErrCreateSwapchain)
	}
	images := make([]vkImage, count)
	result = r.device.api.getSwapchainImagesKHR(r.device.handle, r.swapchain.handle, &count, &images[0])
	if result != vkSuccess && result != vkIncomplete {
		return nil, fmt.Errorf("%w: swapchain images: %s", ErrCreateSwapchain, result)
	}
	return images[:count], nil
}

func (r *Renderer2D) createImageViews() error {
	r.swapchainImageViews = make([]vkImageView, len(r.swapchainImages))
	for index, image := range r.swapchainImages {
		view, err := r.createImageView(image, r.swapchainImageFormat)
		if err != nil {
			return err
		}
		r.swapchainImageViews[index] = view
	}
	return nil
}

func (r *Renderer2D) createRenderPass() error {
	colorAttachment := vkAttachmentDescription{
		Format:         r.swapchainImageFormat,
		Samples:        vkSampleCount1Bit,
		LoadOp:         vkAttachmentLoadOpClear,
		StoreOp:        vkAttachmentStoreOpStore,
		StencilLoadOp:  vkAttachmentLoadOpDontCare,
		StencilStoreOp: vkAttachmentStoreOpDontCare,
		InitialLayout:  vkImageLayoutUndefined,
		FinalLayout:    vkImageLayoutPresentSrcKHR,
	}
	colorRef := vkAttachmentReference{
		Attachment: 0,
		Layout:     vkImageLayoutColorAttachmentOptimal,
	}
	subpass := vkSubpassDescription{
		PipelineBindPoint:    vkPipelineBindPointGraphics,
		ColorAttachmentCount: 1,
		PColorAttachments:    &colorRef,
	}
	dependency := vkSubpassDependency{
		SrcSubpass:      vkSubpassExternal,
		DstSubpass:      0,
		SrcStageMask:    vkPipelineStageColorAttachmentOutputBit,
		DstStageMask:    vkPipelineStageColorAttachmentOutputBit,
		DstAccessMask:   vkAccessColorAttachmentReadBit | vkAccessColorAttachmentWriteBit,
		DependencyFlags: vkDependencyByRegionBit,
	}
	createInfo := vkRenderPassCreateInfo{
		SType:           vkStructureTypeRenderPassCreateInfo,
		AttachmentCount: 1,
		PAttachments:    &colorAttachment,
		SubpassCount:    1,
		PSubpasses:      &subpass,
		DependencyCount: 1,
		PDependencies:   &dependency,
	}
	result := r.device.api.createRenderPass(r.device.handle, &createInfo, nil, &r.renderPass)
	if result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrCreateRenderPass, result)
	}
	return nil
}

func (r *Renderer2D) createFramebuffers() error {
	r.framebuffers = make([]vkFramebuffer, len(r.swapchainImageViews))
	for index, view := range r.swapchainImageViews {
		attachment := view
		createInfo := vkFramebufferCreateInfo{
			SType:           vkStructureTypeFramebufferCreateInfo,
			RenderPass:      r.renderPass,
			AttachmentCount: 1,
			PAttachments:    &attachment,
			Width:           r.swapchainExtent.Width,
			Height:          r.swapchainExtent.Height,
			Layers:          1,
		}
		result := r.device.api.createFramebuffer(r.device.handle, &createInfo, nil, &r.framebuffers[index])
		if result != vkSuccess {
			return fmt.Errorf("%w: %s", ErrCreateFramebuffer, result)
		}
	}
	return nil
}

func (r *Renderer2D) createCommandPool() error {
	createInfo := vkCommandPoolCreateInfo{
		SType:            vkStructureTypeCommandPoolCreateInfo,
		Flags:            vkCommandPoolCreateResetCommandBufferBit,
		QueueFamilyIndex: r.device.graphicsQueueIndex,
	}
	result := r.device.api.createCommandPool(r.device.handle, &createInfo, nil, &r.commandPool)
	if result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrCreateCommandPool, result)
	}
	return nil
}

func (r *Renderer2D) allocateCommandBuffers() error {
	count := len(r.swapchainImages)
	r.commandBuffers = make([]vkCommandBuffer, count)
	createInfo := vkCommandBufferAllocateInfo{
		SType:              vkStructureTypeCommandBufferAllocateInfo,
		CommandPool:        r.commandPool,
		Level:              vkCommandBufferLevelPrimary,
		CommandBufferCount: uint32(count),
	}
	result := r.device.api.allocateCommandBuffers(r.device.handle, &createInfo, &r.commandBuffers[0])
	if result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrAllocateCommandBuffer, result)
	}
	return nil
}

func (r *Renderer2D) createSyncObjects() error {
	count := len(r.commandBuffers)
	r.imageAvailable = make([]vkSemaphore, count)
	r.renderFinished = make([]vkSemaphore, count)
	r.inFlightFences = make([]vkFence, count)

	semaphoreInfo := vkSemaphoreCreateInfo{SType: vkStructureTypeSemaphoreCreateInfo}
	fenceInfo := vkFenceCreateInfo{SType: vkStructureTypeFenceCreateInfo, Flags: vkFenceCreateSignaledBit}
	for index := 0; index < count; index++ {
		if result := r.device.api.createSemaphore(r.device.handle, &semaphoreInfo, nil, &r.imageAvailable[index]); result != vkSuccess {
			return fmt.Errorf("%w: %s", ErrCreateSemaphore, result)
		}
		if result := r.device.api.createSemaphore(r.device.handle, &semaphoreInfo, nil, &r.renderFinished[index]); result != vkSuccess {
			return fmt.Errorf("%w: %s", ErrCreateSemaphore, result)
		}
		if result := r.device.api.createFence(r.device.handle, &fenceInfo, nil, &r.inFlightFences[index]); result != vkSuccess {
			return fmt.Errorf("%w: %s", ErrCreateFence, result)
		}
	}
	return nil
}

func (r *Renderer2D) destroySyncObjects() {
	for _, fence := range r.inFlightFences {
		if fence != 0 {
			r.device.api.destroyFence(r.device.handle, fence, nil)
		}
	}
	for _, semaphore := range r.imageAvailable {
		if semaphore != 0 {
			r.device.api.destroySemaphore(r.device.handle, semaphore, nil)
		}
	}
	for _, semaphore := range r.renderFinished {
		if semaphore != 0 {
			r.device.api.destroySemaphore(r.device.handle, semaphore, nil)
		}
	}
	r.inFlightFences = nil
	r.imageAvailable = nil
	r.renderFinished = nil
}

func (r *Renderer2D) createVertexBuffers() error {
	r.vertexBuffers = make([]gpuBuffer, len(r.commandBuffers))
	for index := range r.vertexBuffers {
		buffer, err := r.createBuffer(vertexBufferSize(initialVertexCapacity), vkBufferUsageVertexBufferBit, vkMemoryPropertyHostVisibleBit|vkMemoryPropertyHostCoherentBit)
		if err != nil {
			return err
		}
		buffer.capacity = initialVertexCapacity
		if result := r.device.api.mapMemory(r.device.handle, buffer.memory, 0, buffer.size, 0, &buffer.mappedPtr); result != vkSuccess {
			buffer.destroy(r.device)
			return fmt.Errorf("%w: %s", ErrMapMemory, result)
		}
		r.vertexBuffers[index] = buffer
	}
	return nil
}

func (r *Renderer2D) destroyVertexBuffers() {
	for _, buffer := range r.vertexBuffers {
		if buffer.mappedPtr != nil {
			r.device.api.unmapMemory(r.device.handle, buffer.memory)
		}
		buffer.destroy(r.device)
	}
	r.vertexBuffers = nil
}

func (r *Renderer2D) createGraphicsPipeline() error {
	vertexModule, err := r.createShaderModule(quadVertexSPV)
	if err != nil {
		return err
	}
	defer r.device.api.destroyShaderModule(r.device.handle, vertexModule, nil)

	fragmentModule, err := r.createShaderModule(quadFragmentSPV)
	if err != nil {
		return err
	}
	defer r.device.api.destroyShaderModule(r.device.handle, fragmentModule, nil)

	entryStorage, entryName := cString("main")
	stages := []vkPipelineShaderStageCreateInfo{
		{
			SType:  vkStructureTypePipelineShaderStageCreateInfo,
			Stage:  vkShaderStageVertexBit,
			Module: vertexModule,
			PName:  entryName,
		},
		{
			SType:  vkStructureTypePipelineShaderStageCreateInfo,
			Stage:  vkShaderStageFragmentBit,
			Module: fragmentModule,
			PName:  entryName,
		},
	}

	binding := vkVertexInputBindingDescription{
		Binding:   0,
		Stride:    uint32(unsafe.Sizeof(quadVertex{})),
		InputRate: vkVertexInputRateVertex,
	}
	attributes := []vkVertexInputAttributeDescription{
		{Location: 0, Binding: 0, Format: vkFormatR32G32Sfloat, Offset: 0},
		{Location: 1, Binding: 0, Format: vkFormatR32G32Sfloat, Offset: 8},
		{Location: 2, Binding: 0, Format: vkFormatR32G32B32A32Sfloat, Offset: 16},
	}
	vertexInput := vkPipelineVertexInputStateCreateInfo{
		SType:                           vkStructureTypePipelineVertexInputStateCreateInfo,
		VertexBindingDescriptionCount:   1,
		PVertexBindingDescriptions:      &binding,
		VertexAttributeDescriptionCount: uint32(len(attributes)),
		PVertexAttributeDescriptions:    &attributes[0],
	}
	inputAssembly := vkPipelineInputAssemblyStateCreateInfo{
		SType:    vkStructureTypePipelineInputAssemblyStateCreateInfo,
		Topology: vkPrimitiveTopologyTriangleList,
	}
	viewportState := vkPipelineViewportStateCreateInfo{
		SType:         vkStructureTypePipelineViewportStateCreateInfo,
		ViewportCount: 1,
		ScissorCount:  1,
	}
	rasterization := vkPipelineRasterizationStateCreateInfo{
		SType:       vkStructureTypePipelineRasterizationStateCreateInfo,
		PolygonMode: vkPolygonModeFill,
		CullMode:    vkCullModeNone,
		FrontFace:   vkFrontFaceCounterClockwise,
		LineWidth:   1,
	}
	multisample := vkPipelineMultisampleStateCreateInfo{
		SType:                vkStructureTypePipelineMultisampleStateCreateInfo,
		RasterizationSamples: vkSampleCount1Bit,
	}
	colorBlendAttachment := vkPipelineColorBlendAttachmentState{
		BlendEnable:         1,
		SrcColorBlendFactor: vkBlendFactorSrcAlpha,
		DstColorBlendFactor: vkBlendFactorOneMinusSrcAlpha,
		ColorBlendOp:        vkBlendOpAdd,
		SrcAlphaBlendFactor: vkBlendFactorOne,
		DstAlphaBlendFactor: vkBlendFactorOneMinusSrcAlpha,
		AlphaBlendOp:        vkBlendOpAdd,
		ColorWriteMask:      vkColorComponentRBit | vkColorComponentGBit | vkColorComponentBBit | vkColorComponentABit,
	}
	colorBlend := vkPipelineColorBlendStateCreateInfo{
		SType:           vkStructureTypePipelineColorBlendStateCreateInfo,
		AttachmentCount: 1,
		PAttachments:    &colorBlendAttachment,
	}
	dynamicStates := []uint32{vkDynamicStateViewport, vkDynamicStateScissor}
	dynamicState := vkPipelineDynamicStateCreateInfo{
		SType:             vkStructureTypePipelineDynamicStateCreateInfo,
		DynamicStateCount: uint32(len(dynamicStates)),
		PDynamicStates:    &dynamicStates[0],
	}
	layoutInfo := vkPipelineLayoutCreateInfo{
		SType:          vkStructureTypePipelineLayoutCreateInfo,
		SetLayoutCount: 1,
		PSetLayouts:    &r.descriptorSetLayout,
	}
	if result := r.device.api.createPipelineLayout(r.device.handle, &layoutInfo, nil, &r.pipelineLayout); result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrCreatePipelineLayout, result)
	}

	pipelineInfo := vkGraphicsPipelineCreateInfo{
		SType:               vkStructureTypeGraphicsPipelineCreateInfo,
		StageCount:          uint32(len(stages)),
		PStages:             &stages[0],
		PVertexInputState:   &vertexInput,
		PInputAssemblyState: &inputAssembly,
		PViewportState:      &viewportState,
		PRasterizationState: &rasterization,
		PMultisampleState:   &multisample,
		PColorBlendState:    &colorBlend,
		PDynamicState:       &dynamicState,
		Layout:              r.pipelineLayout,
		RenderPass:          r.renderPass,
	}
	result := r.device.api.createGraphicsPipelines(r.device.handle, 0, 1, &pipelineInfo, nil, &r.pipeline)
	runtime.KeepAlive(entryStorage)
	if result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrCreatePipeline, result)
	}
	return nil
}

func (r *Renderer2D) createShaderModule(code []byte) (vkShaderModule, error) {
	if len(code) == 0 || len(code)%4 != 0 {
		return 0, fmt.Errorf("%w: shader bytecode must be aligned to 4 bytes", ErrCreateShaderModule)
	}
	createInfo := vkShaderModuleCreateInfo{
		SType:    vkStructureTypeShaderModuleCreateInfo,
		CodeSize: uintptr(len(code)),
		PCode:    (*uint32)(unsafe.Pointer(unsafe.SliceData(code))),
	}
	var module vkShaderModule
	result := r.device.api.createShaderModule(r.device.handle, &createInfo, nil, &module)
	runtime.KeepAlive(code)
	if result != vkSuccess {
		return 0, fmt.Errorf("%w: %s", ErrCreateShaderModule, result)
	}
	return module, nil
}

func (r *Renderer2D) waitForFence(fence vkFence) error {
	if result := r.device.api.waitForFences(r.device.handle, 1, &fence, 1, ^uint64(0)); result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrQueueSubmit, result)
	}
	return nil
}

func (r *Renderer2D) resetFence(fence vkFence) error {
	if result := r.device.api.resetFences(r.device.handle, 1, &fence); result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrQueueSubmit, result)
	}
	return nil
}

func (r *Renderer2D) ensureVertexBuffer(frameIndex int, vertexCount int) error {
	if vertexCount == 0 {
		return nil
	}
	buffer := &r.vertexBuffers[frameIndex]
	if buffer.capacity >= vertexCount {
		return nil
	}
	if buffer.mappedPtr != nil {
		r.device.api.unmapMemory(r.device.handle, buffer.memory)
		buffer.mappedPtr = nil
	}
	buffer.destroy(r.device)
	nextCapacity := vertexCount
	if nextCapacity < initialVertexCapacity {
		nextCapacity = initialVertexCapacity
	}
	created, err := r.createBuffer(vertexBufferSize(nextCapacity), vkBufferUsageVertexBufferBit, vkMemoryPropertyHostVisibleBit|vkMemoryPropertyHostCoherentBit)
	if err != nil {
		return err
	}
	created.capacity = nextCapacity
	if result := r.device.api.mapMemory(r.device.handle, created.memory, 0, created.size, 0, &created.mappedPtr); result != vkSuccess {
		created.destroy(r.device)
		return fmt.Errorf("%w: %s", ErrMapMemory, result)
	}
	*buffer = created
	return nil
}

func (r *Renderer2D) uploadVertices(frameIndex int, vertices []quadVertex) error {
	if len(vertices) == 0 {
		return nil
	}
	buffer := r.vertexBuffers[frameIndex]
	if buffer.mappedPtr == nil {
		return fmt.Errorf("%w: vertex buffer not persistently mapped", ErrMapMemory)
	}
	target := unsafe.Slice((*quadVertex)(buffer.mappedPtr), buffer.capacity)
	copy(target, vertices)
	return nil
}

func (r *Renderer2D) recordCommandBuffer(commandBuffer vkCommandBuffer, imageIndex uint32, clearColor geom.Color, batches []drawBatch) error {
	if result := r.device.api.resetCommandBuffer(commandBuffer, 0); result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrQueueSubmit, result)
	}

	beginInfo := vkCommandBufferBeginInfo{SType: vkStructureTypeCommandBufferBeginInfo}
	if result := r.device.api.beginCommandBuffer(commandBuffer, &beginInfo); result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrQueueSubmit, result)
	}

	clearValue := vkClearValue{
		Color: vkClearColorValue{
			Float32: [4]float32{clearColor.R, clearColor.G, clearColor.B, clearColor.A},
		},
	}
	renderArea := vkRect2D{Extent: r.swapchainExtent}
	renderPassInfo := vkRenderPassBeginInfo{
		SType:           vkStructureTypeRenderPassBeginInfo,
		RenderPass:      r.renderPass,
		Framebuffer:     r.framebuffers[imageIndex],
		RenderArea:      renderArea,
		ClearValueCount: 1,
		PClearValues:    &clearValue,
	}
	r.device.api.cmdBeginRenderPass(commandBuffer, &renderPassInfo, vkSubpassContentsInline)

	viewport, scissor := r.computeViewport()
	r.device.api.cmdSetViewport(commandBuffer, 0, 1, &viewport)
	r.device.api.cmdSetScissor(commandBuffer, 0, 1, &scissor)
	r.device.api.cmdBindPipeline(commandBuffer, vkPipelineBindPointGraphics, r.pipeline)

	if len(batches) > 0 {
		buffer := r.vertexBuffers[r.currentFrame].buffer
		offset := vkDeviceSize(0)
		r.device.api.cmdBindVertexBuffers(commandBuffer, 0, 1, &buffer, &offset)
		var boundSet vkDescriptorSet
		for _, batch := range batches {
			if batch.texture == nil {
				continue
			}
			if batch.texture.descriptorSet != boundSet {
				set := batch.texture.descriptorSet
				r.device.api.cmdBindDescriptorSets(commandBuffer, vkPipelineBindPointGraphics, r.pipelineLayout, 0, 1, &set, 0, nil)
				boundSet = set
			}
			r.device.api.cmdDraw(commandBuffer, batch.vertexCount, 1, batch.firstVertex, 0)
		}
	}

	r.device.api.cmdEndRenderPass(commandBuffer)
	if result := r.device.api.endCommandBuffer(commandBuffer); result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrQueueSubmit, result)
	}
	return nil
}

func (r *Renderer2D) submitFrame(frameIndex int) error {
	waitStage := uint32(vkPipelineStageColorAttachmentOutputBit)
	commandBuffer := r.commandBuffers[frameIndex]
	submitInfo := vkSubmitInfo{
		SType:                vkStructureTypeSubmitInfo,
		WaitSemaphoreCount:   1,
		PWaitSemaphores:      &r.imageAvailable[frameIndex],
		PWaitDstStageMask:    &waitStage,
		CommandBufferCount:   1,
		PCommandBuffers:      &commandBuffer,
		SignalSemaphoreCount: 1,
		PSignalSemaphores:    &r.renderFinished[frameIndex],
	}
	result := r.device.api.queueSubmit(r.device.graphicsQueue, 1, &submitInfo, r.inFlightFences[frameIndex])
	if result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrQueueSubmit, result)
	}
	return nil
}

var errSwapchainNeedsRebuild = fmt.Errorf("swapchain needs rebuild")

func (r *Renderer2D) presentFrame(frameIndex int, imageIndex uint32) error {
	swapchainHandle := r.swapchain.handle
	presentInfo := vkPresentInfoKHR{
		SType:              vkStructureTypePresentInfoKHR,
		WaitSemaphoreCount: 1,
		PWaitSemaphores:    &r.renderFinished[frameIndex],
		SwapchainCount:     1,
		PSwapchains:        &swapchainHandle,
		PImageIndices:      &imageIndex,
	}
	result := r.device.api.queuePresentKHR(r.device.presentQueue, &presentInfo)
	switch result {
	case vkSuccess:
		return nil
	case vkSuboptimalKHR, vkErrorOutOfDateKHR:
		return errSwapchainNeedsRebuild
	default:
		return fmt.Errorf("%w: %s", ErrPresent, result)
	}
}

func (r *Renderer2D) computeViewport() (vkViewport, vkRect2D) {
	targetWidth := float64(r.swapchainExtent.Width)
	targetHeight := float64(r.swapchainExtent.Height)
	logicalWidth := float64(r.virtualWidth)
	logicalHeight := float64(r.virtualHeight)
	if logicalWidth <= 0 || logicalHeight <= 0 {
		logicalWidth = targetWidth
		logicalHeight = targetHeight
	}

	scaleX := targetWidth / logicalWidth
	scaleY := targetHeight / logicalHeight
	scale := math.Min(scaleX, scaleY)
	if r.pixelPerfect && scale >= 1 {
		scale = math.Floor(scale)
		if scale < 1 {
			scale = 1
		}
	}

	viewportWidth := logicalWidth * scale
	viewportHeight := logicalHeight * scale
	offsetX := (targetWidth - viewportWidth) * 0.5
	offsetY := (targetHeight - viewportHeight) * 0.5

	viewport := vkViewport{
		X:        float32(offsetX),
		Y:        float32(offsetY),
		Width:    float32(viewportWidth),
		Height:   float32(viewportHeight),
		MinDepth: 0,
		MaxDepth: 1,
	}
	scissor := vkRect2D{
		Offset: vkOffset2D{X: int32(math.Round(offsetX)), Y: int32(math.Round(offsetY))},
		Extent: vkExtent2D{Width: uint32(math.Round(viewportWidth)), Height: uint32(math.Round(viewportHeight))},
	}
	return viewport, scissor
}
