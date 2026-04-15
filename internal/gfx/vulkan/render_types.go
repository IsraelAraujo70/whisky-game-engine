package vulkan

import "unsafe"

const (
	vkErrorOutOfDateKHR vkResult = -1000001004
	vkSuboptimalKHR     vkResult = 1000001003

	vkStructureTypeFenceCreateInfo                      = 8
	vkStructureTypeSemaphoreCreateInfo                  = 9
	vkStructureTypeBufferCreateInfo                     = 12
	vkStructureTypeImageCreateInfo                      = 14
	vkStructureTypeImageViewCreateInfo                  = 15
	vkStructureTypeShaderModuleCreateInfo               = 16
	vkStructureTypePipelineShaderStageCreateInfo        = 18
	vkStructureTypePipelineVertexInputStateCreateInfo   = 19
	vkStructureTypePipelineInputAssemblyStateCreateInfo = 20
	vkStructureTypePipelineViewportStateCreateInfo      = 22
	vkStructureTypePipelineRasterizationStateCreateInfo = 23
	vkStructureTypePipelineMultisampleStateCreateInfo   = 24
	vkStructureTypePipelineColorBlendStateCreateInfo    = 26
	vkStructureTypePipelineDynamicStateCreateInfo       = 27
	vkStructureTypeGraphicsPipelineCreateInfo           = 28
	vkStructureTypePipelineLayoutCreateInfo             = 30
	vkStructureTypeSamplerCreateInfo                    = 31
	vkStructureTypeDescriptorSetLayoutCreateInfo        = 32
	vkStructureTypeDescriptorPoolCreateInfo             = 33
	vkStructureTypeDescriptorSetAllocateInfo            = 34
	vkStructureTypeWriteDescriptorSet                   = 35
	vkStructureTypeFramebufferCreateInfo                = 37
	vkStructureTypeRenderPassCreateInfo                 = 38
	vkStructureTypeCommandPoolCreateInfo                = 39
	vkStructureTypeCommandBufferAllocateInfo            = 40
	vkStructureTypeCommandBufferBeginInfo               = 42
	vkStructureTypeRenderPassBeginInfo                  = 43
	vkStructureTypeImageMemoryBarrier                   = 45
	vkStructureTypeMemoryAllocateInfo                   = 5
	vkStructureTypeSubmitInfo                           = 4
	vkStructureTypePresentInfoKHR                       = 1000001001

	vkImageUsageTransferDstBit   = 0x00000002
	vkImageUsageSampledBit       = 0x00000004
	vkBufferUsageTransferSrcBit  = 0x00000001
	vkBufferUsageVertexBufferBit = 0x00000080

	vkMemoryPropertyDeviceLocalBit  = 0x00000001
	vkMemoryPropertyHostVisibleBit  = 0x00000002
	vkMemoryPropertyHostCoherentBit = 0x00000004

	vkImageType2D                       = 1
	vkImageTilingOptimal                = 0
	vkImageLayoutUndefined              = 0
	vkImageLayoutColorAttachmentOptimal = 2
	vkImageLayoutTransferDstOptimal     = 6
	vkImageLayoutShaderReadOnlyOptimal  = 7
	vkImageLayoutPresentSrcKHR          = 1000001002

	vkImageViewType2D = 1

	vkImageAspectColorBit = 0x00000001

	vkAttachmentLoadOpClear     = 1
	vkAttachmentStoreOpStore    = 0
	vkAttachmentLoadOpDontCare  = 2
	vkAttachmentStoreOpDontCare = 1

	vkPipelineBindPointGraphics = 0
	vkCommandBufferLevelPrimary = 0
	vkSubpassContentsInline     = 0

	vkCommandPoolCreateResetCommandBufferBit = 0x00000002
	vkFenceCreateSignaledBit                 = 0x00000001

	vkQueueFamilyIgnored = ^uint32(0)
	vkSubpassExternal    = ^uint32(0)

	vkPipelineStageTopOfPipeBit             = 0x00000001
	vkPipelineStageFragmentShaderBit        = 0x00000080
	vkPipelineStageColorAttachmentOutputBit = 0x00000400
	vkPipelineStageTransferBit              = 0x00001000

	vkAccessColorAttachmentReadBit  = 0x00000080
	vkAccessColorAttachmentWriteBit = 0x00000100
	vkAccessShaderReadBit           = 0x00000020
	vkAccessTransferWriteBit        = 0x00001000

	vkDependencyByRegionBit = 0x00000001

	vkPrimitiveTopologyTriangleList = 3
	vkPolygonModeFill               = 0
	vkCullModeNone                  = 0
	vkFrontFaceCounterClockwise     = 1
	vkSampleCount1Bit               = 0x00000001

	vkDynamicStateViewport = 0
	vkDynamicStateScissor  = 1

	vkBlendFactorZero             = 0
	vkBlendFactorOne              = 1
	vkBlendFactorSrcAlpha         = 6
	vkBlendFactorOneMinusSrcAlpha = 7
	vkBlendOpAdd                  = 0

	vkColorComponentRBit = 0x00000001
	vkColorComponentGBit = 0x00000002
	vkColorComponentBBit = 0x00000004
	vkColorComponentABit = 0x00000008

	vkDescriptorTypeCombinedImageSampler = 1

	vkShaderStageVertexBit   = 0x00000001
	vkShaderStageFragmentBit = 0x00000010

	vkFilterNearest                 = 0
	vkSamplerMipmapModeNearest      = 0
	vkSamplerAddressModeClampToEdge = 2
	vkBorderColorIntOpaqueBlack     = 3

	vkWholeSize vkDeviceSize = ^vkDeviceSize(0)
)

type vkDeviceSize uint64
type vkImage uintptr
type vkImageView uintptr
type vkRenderPass uintptr
type vkFramebuffer uintptr
type vkCommandPool uintptr
type vkCommandBuffer uintptr
type vkSemaphore uintptr
type vkFence uintptr
type vkShaderModule uintptr
type vkPipelineLayout uintptr
type vkPipeline uintptr
type vkPipelineCache uintptr
type vkDescriptorSetLayout uintptr
type vkDescriptorPool uintptr
type vkDescriptorSet uintptr
type vkSampler uintptr
type vkBuffer uintptr
type vkDeviceMemory uintptr

type vkMemoryType struct {
	PropertyFlags uint32
	HeapIndex     uint32
}

type vkMemoryHeap struct {
	Size  vkDeviceSize
	Flags uint32
}

type vkPhysicalDeviceMemoryProperties struct {
	MemoryTypeCount uint32
	MemoryTypes     [32]vkMemoryType
	MemoryHeapCount uint32
	MemoryHeaps     [16]vkMemoryHeap
}

type vkMemoryRequirements struct {
	Size           vkDeviceSize
	Alignment      vkDeviceSize
	MemoryTypeBits uint32
}

type vkMemoryAllocateInfo struct {
	SType           int32
	PNext           unsafe.Pointer
	AllocationSize  vkDeviceSize
	MemoryTypeIndex uint32
}

type vkBufferCreateInfo struct {
	SType                 int32
	PNext                 unsafe.Pointer
	Flags                 uint32
	Size                  vkDeviceSize
	Usage                 uint32
	SharingMode           uint32
	QueueFamilyIndexCount uint32
	PQueueFamilyIndices   *uint32
}

type vkExtent3DImage struct {
	Width  uint32
	Height uint32
	Depth  uint32
}

type vkImageCreateInfo struct {
	SType                 int32
	PNext                 unsafe.Pointer
	Flags                 uint32
	ImageType             uint32
	Format                int32
	Extent                vkExtent3DImage
	MipLevels             uint32
	ArrayLayers           uint32
	Samples               uint32
	Tiling                uint32
	Usage                 uint32
	SharingMode           uint32
	QueueFamilyIndexCount uint32
	PQueueFamilyIndices   *uint32
	InitialLayout         uint32
}

type vkComponentMapping struct {
	R int32
	G int32
	B int32
	A int32
}

type vkImageSubresourceRange struct {
	AspectMask     uint32
	BaseMipLevel   uint32
	LevelCount     uint32
	BaseArrayLayer uint32
	LayerCount     uint32
}

type vkImageViewCreateInfo struct {
	SType            int32
	PNext            unsafe.Pointer
	Flags            uint32
	Image            vkImage
	ViewType         uint32
	Format           int32
	Components       vkComponentMapping
	SubresourceRange vkImageSubresourceRange
}

type vkAttachmentDescription struct {
	Flags          uint32
	Format         int32
	Samples        uint32
	LoadOp         int32
	StoreOp        int32
	StencilLoadOp  int32
	StencilStoreOp int32
	InitialLayout  uint32
	FinalLayout    uint32
}

type vkAttachmentReference struct {
	Attachment uint32
	Layout     uint32
}

type vkSubpassDescription struct {
	Flags                   uint32
	PipelineBindPoint       uint32
	InputAttachmentCount    uint32
	PInputAttachments       unsafe.Pointer
	ColorAttachmentCount    uint32
	PColorAttachments       *vkAttachmentReference
	PResolveAttachments     unsafe.Pointer
	PDepthStencilAttachment unsafe.Pointer
	PreserveAttachmentCount uint32
	PreserveAttachments     unsafe.Pointer
}

type vkSubpassDependency struct {
	SrcSubpass      uint32
	DstSubpass      uint32
	SrcStageMask    uint32
	DstStageMask    uint32
	SrcAccessMask   uint32
	DstAccessMask   uint32
	DependencyFlags uint32
}

type vkRenderPassCreateInfo struct {
	SType           int32
	PNext           unsafe.Pointer
	Flags           uint32
	AttachmentCount uint32
	PAttachments    *vkAttachmentDescription
	SubpassCount    uint32
	PSubpasses      *vkSubpassDescription
	DependencyCount uint32
	PDependencies   *vkSubpassDependency
}

type vkFramebufferCreateInfo struct {
	SType           int32
	PNext           unsafe.Pointer
	Flags           uint32
	RenderPass      vkRenderPass
	AttachmentCount uint32
	PAttachments    *vkImageView
	Width           uint32
	Height          uint32
	Layers          uint32
}

type vkCommandPoolCreateInfo struct {
	SType            int32
	PNext            unsafe.Pointer
	Flags            uint32
	QueueFamilyIndex uint32
}

type vkCommandBufferAllocateInfo struct {
	SType              int32
	PNext              unsafe.Pointer
	CommandPool        vkCommandPool
	Level              uint32
	CommandBufferCount uint32
}

type vkCommandBufferBeginInfo struct {
	SType            int32
	PNext            unsafe.Pointer
	Flags            uint32
	PInheritanceInfo unsafe.Pointer
}

type vkOffset2D struct {
	X int32
	Y int32
}

type vkRect2D struct {
	Offset vkOffset2D
	Extent vkExtent2D
}

type vkClearColorValue struct {
	Float32 [4]float32
}

type vkClearValue struct {
	Color vkClearColorValue
}

type vkRenderPassBeginInfo struct {
	SType           int32
	PNext           unsafe.Pointer
	RenderPass      vkRenderPass
	Framebuffer     vkFramebuffer
	RenderArea      vkRect2D
	ClearValueCount uint32
	PClearValues    *vkClearValue
}

type vkSemaphoreCreateInfo struct {
	SType int32
	PNext unsafe.Pointer
	Flags uint32
}

type vkFenceCreateInfo struct {
	SType int32
	PNext unsafe.Pointer
	Flags uint32
}

type vkSubmitInfo struct {
	SType                int32
	PNext                unsafe.Pointer
	WaitSemaphoreCount   uint32
	PWaitSemaphores      *vkSemaphore
	PWaitDstStageMask    *uint32
	CommandBufferCount   uint32
	PCommandBuffers      *vkCommandBuffer
	SignalSemaphoreCount uint32
	PSignalSemaphores    *vkSemaphore
}

type vkPresentInfoKHR struct {
	SType              int32
	PNext              unsafe.Pointer
	WaitSemaphoreCount uint32
	PWaitSemaphores    *vkSemaphore
	SwapchainCount     uint32
	PSwapchains        *vkSwapchainKHR
	PImageIndices      *uint32
	PResults           *vkResult
}

type vkShaderModuleCreateInfo struct {
	SType    int32
	PNext    unsafe.Pointer
	Flags    uint32
	CodeSize uintptr
	PCode    *uint32
}

type vkPipelineShaderStageCreateInfo struct {
	SType               int32
	PNext               unsafe.Pointer
	Flags               uint32
	Stage               uint32
	Module              vkShaderModule
	PName               *byte
	PSpecializationInfo unsafe.Pointer
}

type vkVertexInputBindingDescription struct {
	Binding   uint32
	Stride    uint32
	InputRate uint32
}

type vkVertexInputAttributeDescription struct {
	Location uint32
	Binding  uint32
	Format   int32
	Offset   uint32
}

type vkPipelineVertexInputStateCreateInfo struct {
	SType                           int32
	PNext                           unsafe.Pointer
	Flags                           uint32
	VertexBindingDescriptionCount   uint32
	PVertexBindingDescriptions      *vkVertexInputBindingDescription
	VertexAttributeDescriptionCount uint32
	PVertexAttributeDescriptions    *vkVertexInputAttributeDescription
}

type vkPipelineInputAssemblyStateCreateInfo struct {
	SType                  int32
	PNext                  unsafe.Pointer
	Flags                  uint32
	Topology               uint32
	PrimitiveRestartEnable uint32
}

type vkViewport struct {
	X        float32
	Y        float32
	Width    float32
	Height   float32
	MinDepth float32
	MaxDepth float32
}

type vkPipelineViewportStateCreateInfo struct {
	SType         int32
	PNext         unsafe.Pointer
	Flags         uint32
	ViewportCount uint32
	PViewports    *vkViewport
	ScissorCount  uint32
	PScissors     *vkRect2D
}

type vkPipelineRasterizationStateCreateInfo struct {
	SType                   int32
	PNext                   unsafe.Pointer
	Flags                   uint32
	DepthClampEnable        uint32
	RasterizerDiscardEnable uint32
	PolygonMode             uint32
	CullMode                uint32
	FrontFace               uint32
	DepthBiasEnable         uint32
	DepthBiasConstantFactor float32
	DepthBiasClamp          float32
	DepthBiasSlopeFactor    float32
	LineWidth               float32
}

type vkPipelineMultisampleStateCreateInfo struct {
	SType                 int32
	PNext                 unsafe.Pointer
	Flags                 uint32
	RasterizationSamples  uint32
	SampleShadingEnable   uint32
	MinSampleShading      float32
	PSampleMask           *uint32
	AlphaToCoverageEnable uint32
	AlphaToOneEnable      uint32
}

type vkPipelineColorBlendAttachmentState struct {
	BlendEnable         uint32
	SrcColorBlendFactor uint32
	DstColorBlendFactor uint32
	ColorBlendOp        uint32
	SrcAlphaBlendFactor uint32
	DstAlphaBlendFactor uint32
	AlphaBlendOp        uint32
	ColorWriteMask      uint32
}

type vkPipelineColorBlendStateCreateInfo struct {
	SType           int32
	PNext           unsafe.Pointer
	Flags           uint32
	LogicOpEnable   uint32
	LogicOp         uint32
	AttachmentCount uint32
	PAttachments    *vkPipelineColorBlendAttachmentState
	BlendConstants  [4]float32
}

type vkPipelineDynamicStateCreateInfo struct {
	SType             int32
	PNext             unsafe.Pointer
	Flags             uint32
	DynamicStateCount uint32
	PDynamicStates    *uint32
}

type vkPipelineLayoutCreateInfo struct {
	SType                  int32
	PNext                  unsafe.Pointer
	Flags                  uint32
	SetLayoutCount         uint32
	PSetLayouts            *vkDescriptorSetLayout
	PushConstantRangeCount uint32
	PPushConstantRanges    unsafe.Pointer
}

type vkDescriptorSetLayoutBinding struct {
	Binding            uint32
	DescriptorType     uint32
	DescriptorCount    uint32
	StageFlags         uint32
	PImmutableSamplers *vkSampler
}

type vkDescriptorSetLayoutCreateInfo struct {
	SType        int32
	PNext        unsafe.Pointer
	Flags        uint32
	BindingCount uint32
	PBindings    *vkDescriptorSetLayoutBinding
}

type vkDescriptorPoolSize struct {
	Type            uint32
	DescriptorCount uint32
}

type vkDescriptorPoolCreateInfo struct {
	SType         int32
	PNext         unsafe.Pointer
	Flags         uint32
	MaxSets       uint32
	PoolSizeCount uint32
	PPoolSizes    *vkDescriptorPoolSize
}

type vkDescriptorSetAllocateInfo struct {
	SType              int32
	PNext              unsafe.Pointer
	DescriptorPool     vkDescriptorPool
	DescriptorSetCount uint32
	PSetLayouts        *vkDescriptorSetLayout
}

type vkDescriptorImageInfo struct {
	Sampler     vkSampler
	ImageView   vkImageView
	ImageLayout uint32
}

type vkWriteDescriptorSet struct {
	SType            int32
	PNext            unsafe.Pointer
	DstSet           vkDescriptorSet
	DstBinding       uint32
	DstArrayElement  uint32
	DescriptorCount  uint32
	DescriptorType   uint32
	PImageInfo       *vkDescriptorImageInfo
	PBufferInfo      unsafe.Pointer
	PTexelBufferView unsafe.Pointer
}

type vkGraphicsPipelineCreateInfo struct {
	SType               int32
	PNext               unsafe.Pointer
	Flags               uint32
	StageCount          uint32
	PStages             *vkPipelineShaderStageCreateInfo
	PVertexInputState   *vkPipelineVertexInputStateCreateInfo
	PInputAssemblyState *vkPipelineInputAssemblyStateCreateInfo
	PTessellationState  unsafe.Pointer
	PViewportState      *vkPipelineViewportStateCreateInfo
	PRasterizationState *vkPipelineRasterizationStateCreateInfo
	PMultisampleState   *vkPipelineMultisampleStateCreateInfo
	PDepthStencilState  unsafe.Pointer
	PColorBlendState    *vkPipelineColorBlendStateCreateInfo
	PDynamicState       *vkPipelineDynamicStateCreateInfo
	Layout              vkPipelineLayout
	RenderPass          vkRenderPass
	Subpass             uint32
	BasePipelineHandle  vkPipeline
	BasePipelineIndex   int32
}

type vkSamplerCreateInfo struct {
	SType                   int32
	PNext                   unsafe.Pointer
	Flags                   uint32
	MagFilter               int32
	MinFilter               int32
	MipmapMode              int32
	AddressModeU            int32
	AddressModeV            int32
	AddressModeW            int32
	MipLodBias              float32
	AnisotropyEnable        uint32
	MaxAnisotropy           float32
	CompareEnable           uint32
	CompareOp               int32
	MinLod                  float32
	MaxLod                  float32
	BorderColor             int32
	UnnormalizedCoordinates uint32
}

type vkImageSubresourceLayers struct {
	AspectMask     uint32
	MipLevel       uint32
	BaseArrayLayer uint32
	LayerCount     uint32
}

type vkOffset3D struct {
	X int32
	Y int32
	Z int32
}

type vkBufferImageCopy struct {
	BufferOffset      vkDeviceSize
	BufferRowLength   uint32
	BufferImageHeight uint32
	ImageSubresource  vkImageSubresourceLayers
	ImageOffset       vkOffset3D
	ImageExtent       vkExtent3DImage
}

type vkImageMemoryBarrier struct {
	SType               int32
	PNext               unsafe.Pointer
	SrcAccessMask       uint32
	DstAccessMask       uint32
	OldLayout           uint32
	NewLayout           uint32
	SrcQueueFamilyIndex uint32
	DstQueueFamilyIndex uint32
	Image               vkImage
	SubresourceRange    vkImageSubresourceRange
}
