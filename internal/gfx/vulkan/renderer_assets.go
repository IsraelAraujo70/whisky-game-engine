package vulkan

import (
	"fmt"
	"image"
	"image/draw"
	"unsafe"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

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
	if r.debugFont != nil {
		r.debugFont.texture.destroy(r.device)
		r.debugFont = nil
	}
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

func (r *Renderer2D) createDebugFont() (*bitmapFont, error) {
	face := basicfont.Face7x13
	printables := make([]rune, 0, 95)
	for value := rune(32); value <= 126; value++ {
		printables = append(printables, value)
	}

	cols := 16
	rows := (len(printables) + cols - 1) / cols
	atlasWidth := cols * face.Advance
	atlasHeight := rows * face.Height
	atlas := image.NewRGBA(image.Rect(0, 0, atlasWidth, atlasHeight))
	drawer := &font.Drawer{
		Dst:  atlas,
		Src:  image.White,
		Face: face,
	}

	glyphs := make(map[rune]geom.Rect, len(printables))
	for index, value := range printables {
		col := index % cols
		row := index / cols
		x := col * face.Advance
		y := row * face.Height
		drawer.Dot = fixed.P(x, y+face.Ascent)
		drawer.DrawString(string(value))
		glyphs[value] = geom.Rect{
			X: float64(x),
			Y: float64(y),
			W: float64(face.Width),
			H: float64(face.Height),
		}
	}

	texture, err := r.createTextureFromRGBA(atlas.Pix, atlas.Bounds().Dx(), atlas.Bounds().Dy())
	if err != nil {
		return nil, err
	}
	return &bitmapFont{
		texture:     texture,
		glyphWidth:  face.Advance,
		glyphHeight: face.Height,
		lineHeight:  face.Height + 2,
		glyphs:      glyphs,
	}, nil
}
