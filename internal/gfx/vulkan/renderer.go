package vulkan

import (
	_ "embed"
	"fmt"
	"image"
	"image/draw"
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
	buffer   vkBuffer
	memory   vkDeviceMemory
	size     vkDeviceSize
	capacity int
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
	if texture, ok := r.texturesByPath[cleanPath]; ok {
		return texture.id, texture.width, texture.height, nil
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
	_ = lines

	if r == nil {
		return nil
	}
	if err := r.ensureSwapchainResources(); err != nil {
		return err
	}

	vertices, batches := r.buildDrawData(cmds)
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
		r.vertexBuffers[index] = buffer
	}
	return nil
}

func (r *Renderer2D) destroyVertexBuffers() {
	for _, buffer := range r.vertexBuffers {
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
	*buffer = created
	return nil
}

func (r *Renderer2D) uploadVertices(frameIndex int, vertices []quadVertex) error {
	if len(vertices) == 0 {
		return nil
	}
	buffer := r.vertexBuffers[frameIndex]
	size := vertexBufferSize(len(vertices))
	var mapped unsafe.Pointer
	result := r.device.api.mapMemory(r.device.handle, buffer.memory, 0, size, 0, &mapped)
	if result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrMapMemory, result)
	}
	target := unsafe.Slice((*quadVertex)(mapped), len(vertices))
	copy(target, vertices)
	r.device.api.unmapMemory(r.device.handle, buffer.memory)
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

func (r *Renderer2D) buildDrawData(cmds []render.DrawCmd) ([]quadVertex, []drawBatch) {
	vertices := make([]quadVertex, 0, len(cmds)*6)
	batches := make([]drawBatch, 0, len(cmds))
	virtualWidth := float64(r.virtualWidth)
	virtualHeight := float64(r.virtualHeight)
	if virtualWidth <= 0 || virtualHeight <= 0 {
		virtualWidth = float64(r.swapchain.desc.Extent.Width)
		virtualHeight = float64(r.swapchain.desc.Extent.Height)
	}

	for _, cmd := range cmds {
		switch drawCmd := cmd.(type) {
		case render.FillRect:
			first := uint32(len(vertices))
			vertices = appendQuad(vertices, drawCmd.Rect, geom.Rect{W: 1, H: 1}, quadColor(drawCmd.Color), virtualWidth, virtualHeight, false, false, 1, 1)
			batches = append(batches, drawBatch{texture: r.whiteTexture, firstVertex: first, vertexCount: 6})
		case render.SpriteCmd:
			texture := r.texturesByID[drawCmd.Texture]
			if texture == nil {
				continue
			}
			first := uint32(len(vertices))
			vertices = appendQuad(vertices, drawCmd.Dst, drawCmd.Src, whiteColor(), virtualWidth, virtualHeight, drawCmd.FlipH, drawCmd.FlipV, texture.width, texture.height)
			batches = append(batches, drawBatch{texture: texture, firstVertex: first, vertexCount: 6})
		}
	}
	return vertices, batches
}

func appendQuad(vertices []quadVertex, dst geom.Rect, src geom.Rect, color [4]float32, virtualWidth, virtualHeight float64, flipH, flipV bool, textureWidth, textureHeight int) []quadVertex {
	if src.W == 0 {
		src.W = float64(textureWidth)
	}
	if src.H == 0 {
		src.H = float64(textureHeight)
	}
	u0 := float32(src.X / float64(textureWidth))
	v0 := float32(src.Y / float64(textureHeight))
	u1 := float32((src.X + src.W) / float64(textureWidth))
	v1 := float32((src.Y + src.H) / float64(textureHeight))
	if flipH {
		u0, u1 = u1, u0
	}
	if flipV {
		v0, v1 = v1, v0
	}

	x0 := clipX(dst.X, virtualWidth)
	y0 := clipY(dst.Y, virtualHeight)
	x1 := clipX(dst.X+dst.W, virtualWidth)
	y1 := clipY(dst.Y+dst.H, virtualHeight)

	topLeft := quadVertex{Position: [2]float32{x0, y0}, UV: [2]float32{u0, v0}, Color: color}
	topRight := quadVertex{Position: [2]float32{x1, y0}, UV: [2]float32{u1, v0}, Color: color}
	bottomLeft := quadVertex{Position: [2]float32{x0, y1}, UV: [2]float32{u0, v1}, Color: color}
	bottomRight := quadVertex{Position: [2]float32{x1, y1}, UV: [2]float32{u1, v1}, Color: color}

	return append(vertices,
		topLeft,
		bottomLeft,
		topRight,
		topRight,
		bottomLeft,
		bottomRight,
	)
}

func clipX(x float64, width float64) float32 {
	if width == 0 {
		return -1
	}
	return float32((x/width)*2 - 1)
}

func clipY(y float64, height float64) float32 {
	if height == 0 {
		return -1
	}
	return float32((y/height)*2 - 1)
}

func whiteColor() [4]float32 {
	return [4]float32{1, 1, 1, 1}
}

func quadColor(color geom.Color) [4]float32 {
	return [4]float32{color.R, color.G, color.B, color.A}
}

func vertexBufferSize(vertexCount int) vkDeviceSize {
	return vkDeviceSize(vertexCount) * vkDeviceSize(unsafe.Sizeof(quadVertex{}))
}

func (r *Renderer2D) createTextureFromRGBA(pixels []byte, width, height int) (*gpuTexture, error) {
	staging, err := r.createBuffer(vkDeviceSize(len(pixels)), vkBufferUsageTransferSrcBit, vkMemoryPropertyHostVisibleBit|vkMemoryPropertyHostCoherentBit)
	if err != nil {
		return nil, err
	}
	defer staging.destroy(r.device)

	var mapped unsafe.Pointer
	if result := r.device.api.mapMemory(r.device.handle, staging.memory, 0, staging.size, 0, &mapped); result != vkSuccess {
		return nil, fmt.Errorf("%w: %s", ErrMapMemory, result)
	}
	copy(unsafe.Slice((*byte)(mapped), len(pixels)), pixels)
	r.device.api.unmapMemory(r.device.handle, staging.memory)

	image, memory, err := r.createImage(width, height, vkFormatR8G8B8A8Unorm, vkImageUsageTransferDstBit|vkImageUsageSampledBit)
	if err != nil {
		return nil, err
	}

	if err := r.runImmediateCommands(func(commandBuffer vkCommandBuffer) {
		r.transitionImageLayout(commandBuffer, image, vkImageLayoutUndefined, vkImageLayoutTransferDstOptimal, 0, vkAccessTransferWriteBit, vkPipelineStageTopOfPipeBit, vkPipelineStageTransferBit)
		r.copyBufferToImage(commandBuffer, staging.buffer, image, width, height)
		r.transitionImageLayout(commandBuffer, image, vkImageLayoutTransferDstOptimal, vkImageLayoutShaderReadOnlyOptimal, vkAccessTransferWriteBit, vkAccessShaderReadBit, vkPipelineStageTransferBit, vkPipelineStageFragmentShaderBit)
	}); err != nil {
		r.device.api.destroyImage(r.device.handle, image, nil)
		r.device.api.freeMemory(r.device.handle, memory, nil)
		return nil, err
	}

	view, err := r.createImageView(image, vkFormatR8G8B8A8Unorm)
	if err != nil {
		r.device.api.destroyImage(r.device.handle, image, nil)
		r.device.api.freeMemory(r.device.handle, memory, nil)
		return nil, err
	}

	setLayout := r.descriptorSetLayout
	allocInfo := vkDescriptorSetAllocateInfo{
		SType:              vkStructureTypeDescriptorSetAllocateInfo,
		DescriptorPool:     r.descriptorPool,
		DescriptorSetCount: 1,
		PSetLayouts:        &setLayout,
	}
	var descriptorSet vkDescriptorSet
	if result := r.device.api.allocateDescriptorSets(r.device.handle, &allocInfo, &descriptorSet); result != vkSuccess {
		r.device.api.destroyImageView(r.device.handle, view, nil)
		r.device.api.destroyImage(r.device.handle, image, nil)
		r.device.api.freeMemory(r.device.handle, memory, nil)
		return nil, fmt.Errorf("%w: %s", ErrAllocateDescriptorSet, result)
	}

	imageInfo := vkDescriptorImageInfo{
		Sampler:     r.sampler,
		ImageView:   view,
		ImageLayout: vkImageLayoutShaderReadOnlyOptimal,
	}
	write := vkWriteDescriptorSet{
		SType:           vkStructureTypeWriteDescriptorSet,
		DstSet:          descriptorSet,
		DstBinding:      0,
		DescriptorCount: 1,
		DescriptorType:  vkDescriptorTypeCombinedImageSampler,
		PImageInfo:      &imageInfo,
	}
	r.device.api.updateDescriptorSets(r.device.handle, 1, &write, 0, nil)

	return &gpuTexture{
		width:         width,
		height:        height,
		image:         image,
		memory:        memory,
		view:          view,
		descriptorSet: descriptorSet,
	}, nil
}

func (r *Renderer2D) createImageView(image vkImage, format int32) (vkImageView, error) {
	createInfo := vkImageViewCreateInfo{
		SType:    vkStructureTypeImageViewCreateInfo,
		Image:    image,
		ViewType: vkImageViewType2D,
		Format:   format,
		SubresourceRange: vkImageSubresourceRange{
			AspectMask:     vkImageAspectColorBit,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}
	var view vkImageView
	result := r.device.api.createImageView(r.device.handle, &createInfo, nil, &view)
	if result != vkSuccess {
		return 0, fmt.Errorf("%w: %s", ErrCreateImageView, result)
	}
	return view, nil
}

func (r *Renderer2D) createBuffer(size vkDeviceSize, usage uint32, properties uint32) (gpuBuffer, error) {
	createInfo := vkBufferCreateInfo{
		SType:       vkStructureTypeBufferCreateInfo,
		Size:        size,
		Usage:       usage,
		SharingMode: vkSharingModeExclusive,
	}
	var buffer vkBuffer
	if result := r.device.api.createBuffer(r.device.handle, &createInfo, nil, &buffer); result != vkSuccess {
		return gpuBuffer{}, fmt.Errorf("%w: %s", ErrCreateBuffer, result)
	}

	var requirements vkMemoryRequirements
	r.device.api.getBufferMemoryRequirements(r.device.handle, buffer, &requirements)
	memoryType, err := findMemoryType(r.device.memoryProperties, requirements.MemoryTypeBits, properties)
	if err != nil {
		r.device.api.destroyBuffer(r.device.handle, buffer, nil)
		return gpuBuffer{}, err
	}
	allocInfo := vkMemoryAllocateInfo{
		SType:           vkStructureTypeMemoryAllocateInfo,
		AllocationSize:  requirements.Size,
		MemoryTypeIndex: memoryType,
	}
	var memory vkDeviceMemory
	if result := r.device.api.allocateMemory(r.device.handle, &allocInfo, nil, &memory); result != vkSuccess {
		r.device.api.destroyBuffer(r.device.handle, buffer, nil)
		return gpuBuffer{}, fmt.Errorf("%w: %s", ErrAllocateMemory, result)
	}
	if result := r.device.api.bindBufferMemory(r.device.handle, buffer, memory, 0); result != vkSuccess {
		r.device.api.freeMemory(r.device.handle, memory, nil)
		r.device.api.destroyBuffer(r.device.handle, buffer, nil)
		return gpuBuffer{}, fmt.Errorf("%w: %s", ErrCreateBuffer, result)
	}
	return gpuBuffer{buffer: buffer, memory: memory, size: size}, nil
}

func (r *Renderer2D) createImage(width, height int, format int32, usage uint32) (vkImage, vkDeviceMemory, error) {
	createInfo := vkImageCreateInfo{
		SType:         vkStructureTypeImageCreateInfo,
		ImageType:     vkImageType2D,
		Format:        format,
		Extent:        vkExtent3DImage{Width: uint32(width), Height: uint32(height), Depth: 1},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       vkSampleCount1Bit,
		Tiling:        vkImageTilingOptimal,
		Usage:         usage,
		SharingMode:   vkSharingModeExclusive,
		InitialLayout: vkImageLayoutUndefined,
	}
	var image vkImage
	if result := r.device.api.createImage(r.device.handle, &createInfo, nil, &image); result != vkSuccess {
		return 0, 0, fmt.Errorf("%w: %s", ErrCreateImage, result)
	}

	var requirements vkMemoryRequirements
	r.device.api.getImageMemoryRequirements(r.device.handle, image, &requirements)
	memoryType, err := findMemoryType(r.device.memoryProperties, requirements.MemoryTypeBits, vkMemoryPropertyDeviceLocalBit)
	if err != nil {
		r.device.api.destroyImage(r.device.handle, image, nil)
		return 0, 0, err
	}
	allocInfo := vkMemoryAllocateInfo{
		SType:           vkStructureTypeMemoryAllocateInfo,
		AllocationSize:  requirements.Size,
		MemoryTypeIndex: memoryType,
	}
	var memory vkDeviceMemory
	if result := r.device.api.allocateMemory(r.device.handle, &allocInfo, nil, &memory); result != vkSuccess {
		r.device.api.destroyImage(r.device.handle, image, nil)
		return 0, 0, fmt.Errorf("%w: %s", ErrAllocateMemory, result)
	}
	if result := r.device.api.bindImageMemory(r.device.handle, image, memory, 0); result != vkSuccess {
		r.device.api.freeMemory(r.device.handle, memory, nil)
		r.device.api.destroyImage(r.device.handle, image, nil)
		return 0, 0, fmt.Errorf("%w: %s", ErrCreateImage, result)
	}
	return image, memory, nil
}

func findMemoryType(properties vkPhysicalDeviceMemoryProperties, typeBits uint32, required uint32) (uint32, error) {
	for index := uint32(0); index < properties.MemoryTypeCount; index++ {
		mask := uint32(1) << index
		if typeBits&mask == 0 {
			continue
		}
		if properties.MemoryTypes[index].PropertyFlags&required == required {
			return index, nil
		}
	}
	return 0, fmt.Errorf("%w: no memory type for property mask %#x", ErrAllocateMemory, required)
}

func (r *Renderer2D) runImmediateCommands(record func(commandBuffer vkCommandBuffer)) error {
	createInfo := vkCommandBufferAllocateInfo{
		SType:              vkStructureTypeCommandBufferAllocateInfo,
		CommandPool:        r.commandPool,
		Level:              vkCommandBufferLevelPrimary,
		CommandBufferCount: 1,
	}
	var commandBuffer vkCommandBuffer
	if result := r.device.api.allocateCommandBuffers(r.device.handle, &createInfo, &commandBuffer); result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrAllocateCommandBuffer, result)
	}
	defer r.device.api.freeCommandBuffers(r.device.handle, r.commandPool, 1, &commandBuffer)

	beginInfo := vkCommandBufferBeginInfo{SType: vkStructureTypeCommandBufferBeginInfo}
	if result := r.device.api.beginCommandBuffer(commandBuffer, &beginInfo); result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrQueueSubmit, result)
	}
	record(commandBuffer)
	if result := r.device.api.endCommandBuffer(commandBuffer); result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrQueueSubmit, result)
	}

	submitInfo := vkSubmitInfo{
		SType:              vkStructureTypeSubmitInfo,
		CommandBufferCount: 1,
		PCommandBuffers:    &commandBuffer,
	}
	if result := r.device.api.queueSubmit(r.device.graphicsQueue, 1, &submitInfo, 0); result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrQueueSubmit, result)
	}
	if result := r.device.api.queueWaitIdle(r.device.graphicsQueue); result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrQueueSubmit, result)
	}
	return nil
}

func (r *Renderer2D) transitionImageLayout(commandBuffer vkCommandBuffer, image vkImage, oldLayout, newLayout uint32, srcAccessMask, dstAccessMask uint32, srcStage, dstStage uint32) {
	barrier := vkImageMemoryBarrier{
		SType:               vkStructureTypeImageMemoryBarrier,
		OldLayout:           oldLayout,
		NewLayout:           newLayout,
		SrcAccessMask:       srcAccessMask,
		DstAccessMask:       dstAccessMask,
		SrcQueueFamilyIndex: vkQueueFamilyIgnored,
		DstQueueFamilyIndex: vkQueueFamilyIgnored,
		Image:               image,
		SubresourceRange: vkImageSubresourceRange{
			AspectMask:     vkImageAspectColorBit,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}
	r.device.api.cmdPipelineBarrier(commandBuffer, srcStage, dstStage, 0, 0, nil, 0, nil, 1, &barrier)
}

func (r *Renderer2D) copyBufferToImage(commandBuffer vkCommandBuffer, buffer vkBuffer, image vkImage, width, height int) {
	region := vkBufferImageCopy{
		ImageSubresource: vkImageSubresourceLayers{
			AspectMask:     vkImageAspectColorBit,
			MipLevel:       0,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
		ImageExtent: vkExtent3DImage{
			Width:  uint32(width),
			Height: uint32(height),
			Depth:  1,
		},
	}
	r.device.api.cmdCopyBufferToImage(commandBuffer, buffer, image, vkImageLayoutTransferDstOptimal, 1, &region)
}

func (r *Renderer2D) destroyTextures() {
	for _, texture := range r.texturesByID {
		texture.destroy(r.device)
	}
	r.texturesByID = make(map[render.TextureID]*gpuTexture)
	r.texturesByPath = make(map[string]*gpuTexture)
	if r.whiteTexture != nil {
		r.whiteTexture.destroy(r.device)
		r.whiteTexture = nil
	}
}

func (t *gpuTexture) destroy(device *device) {
	if t == nil || device == nil {
		return
	}
	if t.view != 0 {
		device.api.destroyImageView(device.handle, t.view, nil)
		t.view = 0
	}
	if t.image != 0 {
		device.api.destroyImage(device.handle, t.image, nil)
		t.image = 0
	}
	if t.memory != 0 {
		device.api.freeMemory(device.handle, t.memory, nil)
		t.memory = 0
	}
}

func (b *gpuBuffer) destroy(device *device) {
	if device == nil {
		return
	}
	if b.buffer != 0 {
		device.api.destroyBuffer(device.handle, b.buffer, nil)
		b.buffer = 0
	}
	if b.memory != 0 {
		device.api.freeMemory(device.handle, b.memory, nil)
		b.memory = 0
	}
	b.capacity = 0
	b.size = 0
}

func imageToRGBA(src image.Image) *image.RGBA {
	if rgba, ok := src.(*image.RGBA); ok {
		return rgba
	}
	bounds := src.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, src, bounds.Min, draw.Src)
	return rgba
}

func onePixelWhite() []byte {
	return []byte{255, 255, 255, 255}
}
