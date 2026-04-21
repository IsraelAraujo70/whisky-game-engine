package vulkan

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
)

var (
	ErrUnavailable            = errors.New("vulkan: loader unavailable")
	ErrMissingExtension       = errors.New("vulkan: required instance extension unavailable")
	ErrMissingDeviceExtension = errors.New("vulkan: required device extension unavailable")
	ErrMissingLayer           = errors.New("vulkan: required validation layer unavailable")
	ErrCreateInstance         = errors.New("vulkan: failed to create instance")
	ErrCreateSurface          = errors.New("vulkan: failed to create surface")
	ErrCreateDevice           = errors.New("vulkan: failed to create logical device")
	ErrCreateSwapchain        = errors.New("vulkan: failed to create swapchain")
	ErrCreateImageView        = errors.New("vulkan: failed to create image view")
	ErrCreateRenderPass       = errors.New("vulkan: failed to create render pass")
	ErrCreateFramebuffer      = errors.New("vulkan: failed to create framebuffer")
	ErrCreateCommandPool      = errors.New("vulkan: failed to create command pool")
	ErrAllocateCommandBuffer  = errors.New("vulkan: failed to allocate command buffer")
	ErrCreateSemaphore        = errors.New("vulkan: failed to create semaphore")
	ErrCreateFence            = errors.New("vulkan: failed to create fence")
	ErrCreateShaderModule     = errors.New("vulkan: failed to create shader module")
	ErrCreatePipelineLayout   = errors.New("vulkan: failed to create pipeline layout")
	ErrCreatePipeline         = errors.New("vulkan: failed to create pipeline")
	ErrCreateDescriptorLayout = errors.New("vulkan: failed to create descriptor set layout")
	ErrCreateDescriptorPool   = errors.New("vulkan: failed to create descriptor pool")
	ErrAllocateDescriptorSet  = errors.New("vulkan: failed to allocate descriptor set")
	ErrCreateSampler          = errors.New("vulkan: failed to create sampler")
	ErrCreateBuffer           = errors.New("vulkan: failed to create buffer")
	ErrAllocateMemory         = errors.New("vulkan: failed to allocate device memory")
	ErrCreateImage            = errors.New("vulkan: failed to create image")
	ErrMapMemory              = errors.New("vulkan: failed to map device memory")
	ErrAcquireImage           = errors.New("vulkan: failed to acquire swapchain image")
	ErrQueueSubmit            = errors.New("vulkan: failed to submit queue work")
	ErrPresent                = errors.New("vulkan: failed to present swapchain image")
	ErrNoPhysicalDevice       = errors.New("vulkan: no physical devices available")
	ErrNoSuitableDevice       = errors.New("vulkan: no suitable physical device found")
	ErrNoQueueFamily          = errors.New("vulkan: no compatible queue family found")
	ErrDeviceWaitIdle         = errors.New("vulkan: failed to wait for device idle")
	ErrNotImplemented         = errors.New("vulkan: feature not implemented yet")
	ErrSurfaceUnsupported     = errors.New("vulkan: surface creation is not implemented yet")
)

const (
	vkSuccess          vkResult = 0
	vkIncomplete       vkResult = 5
	vkErrorLayerAbsent vkResult = -6
	vkErrorExtAbsent   vkResult = -7

	vkStructureTypeApplicationInfo    = 0
	vkStructureTypeInstanceCreateInfo = 1

	vkAPIVersion10 uint32 = 1 << 22
)

type Options struct {
	EnableValidation bool
	SurfaceTarget    *rhi.SurfaceTarget
	ApplicationName  string
	EngineName       string
	APIVersion       uint32
}

type vkResult int32
type vkInstance uintptr

type vkApplicationInfo struct {
	SType              int32
	_                  [4]byte
	PNext              unsafe.Pointer
	PApplicationName   *byte
	ApplicationVersion uint32
	PEngineName        *byte
	EngineVersion      uint32
	APIVersion         uint32
}

type vkInstanceCreateInfo struct {
	SType                   int32
	_                       [4]byte
	PNext                   unsafe.Pointer
	Flags                   uint32
	_                       [4]byte
	PApplicationInfo        *vkApplicationInfo
	EnabledLayerCount       uint32
	PpEnabledLayerNames     **byte
	EnabledExtensionCount   uint32
	PpEnabledExtensionNames **byte
}

type vkExtensionProperties struct {
	ExtensionName [256]byte
	SpecVersion   uint32
}

type vkLayerProperties struct {
	LayerName             [256]byte
	SpecVersion           uint32
	ImplementationVersion uint32
	Description           [256]byte
}

type vulkanAPI struct {
	enumerateInstanceExtensionProperties    func(layerName *byte, propertyCount *uint32, properties *vkExtensionProperties) vkResult
	enumerateInstanceLayerProperties        func(propertyCount *uint32, properties *vkLayerProperties) vkResult
	createInstance                          func(createInfo *vkInstanceCreateInfo, allocator unsafe.Pointer, instance *vkInstance) vkResult
	destroyInstance                         func(instance vkInstance, allocator unsafe.Pointer)
	enumeratePhysicalDevices                func(instance vkInstance, physicalDeviceCount *uint32, physicalDevices *vkPhysicalDevice) vkResult
	getPhysicalDeviceProperties             func(physicalDevice vkPhysicalDevice, properties unsafe.Pointer)
	getPhysicalDeviceQueueFamilyProperties  func(physicalDevice vkPhysicalDevice, queueFamilyPropertyCount *uint32, queueFamilyProperties *vkQueueFamilyProperties)
	getPhysicalDeviceMemoryProperties       func(physicalDevice vkPhysicalDevice, memoryProperties *vkPhysicalDeviceMemoryProperties)
	enumerateDeviceExtensionProperties      func(physicalDevice vkPhysicalDevice, layerName *byte, propertyCount *uint32, properties *vkExtensionProperties) vkResult
	getPhysicalDeviceSurfaceSupportKHR      func(physicalDevice vkPhysicalDevice, queueFamilyIndex uint32, surface vkSurfaceKHR, supported *uint32) vkResult
	getPhysicalDeviceSurfaceCapabilitiesKHR func(physicalDevice vkPhysicalDevice, surface vkSurfaceKHR, surfaceCapabilities *vkSurfaceCapabilitiesKHR) vkResult
	getPhysicalDeviceSurfaceFormatsKHR      func(physicalDevice vkPhysicalDevice, surface vkSurfaceKHR, surfaceFormatCount *uint32, surfaceFormats *vkSurfaceFormatKHR) vkResult
	getPhysicalDeviceSurfacePresentModesKHR func(physicalDevice vkPhysicalDevice, surface vkSurfaceKHR, presentModeCount *uint32, presentModes *int32) vkResult
	createDevice                            func(physicalDevice vkPhysicalDevice, createInfo *vkDeviceCreateInfo, allocator unsafe.Pointer, device *vkDevice) vkResult
	destroyDevice                           func(device vkDevice, allocator unsafe.Pointer)
	getDeviceQueue                          func(device vkDevice, queueFamilyIndex uint32, queueIndex uint32, queue *vkQueue)
	deviceWaitIdle                          func(device vkDevice) vkResult
	createWin32SurfaceKHR                   func(instance vkInstance, createInfo *vkWin32SurfaceCreateInfoKHR, allocator unsafe.Pointer, surface *vkSurfaceKHR) vkResult
	createXlibSurfaceKHR                    func(instance vkInstance, createInfo *vkXlibSurfaceCreateInfoKHR, allocator unsafe.Pointer, surface *vkSurfaceKHR) vkResult
	createWaylandSurfaceKHR                 func(instance vkInstance, createInfo *vkWaylandSurfaceCreateInfoKHR, allocator unsafe.Pointer, surface *vkSurfaceKHR) vkResult
	destroySurfaceKHR                       func(instance vkInstance, surface vkSurfaceKHR, allocator unsafe.Pointer)
	createSwapchainKHR                      func(device vkDevice, createInfo *vkSwapchainCreateInfoKHR, allocator unsafe.Pointer, swapchain *vkSwapchainKHR) vkResult
	destroySwapchainKHR                     func(device vkDevice, swapchain vkSwapchainKHR, allocator unsafe.Pointer)
	getSwapchainImagesKHR                   func(device vkDevice, swapchain vkSwapchainKHR, count *uint32, images *vkImage) vkResult
	acquireNextImageKHR                     func(device vkDevice, swapchain vkSwapchainKHR, timeout uint64, semaphore vkSemaphore, fence vkFence, imageIndex *uint32) vkResult
	queuePresentKHR                         func(queue vkQueue, presentInfo *vkPresentInfoKHR) vkResult
	queueSubmit                             func(queue vkQueue, submitCount uint32, submits *vkSubmitInfo, fence vkFence) vkResult
	queueWaitIdle                           func(queue vkQueue) vkResult
	createImageView                         func(device vkDevice, createInfo *vkImageViewCreateInfo, allocator unsafe.Pointer, imageView *vkImageView) vkResult
	destroyImageView                        func(device vkDevice, imageView vkImageView, allocator unsafe.Pointer)
	createRenderPass                        func(device vkDevice, createInfo *vkRenderPassCreateInfo, allocator unsafe.Pointer, renderPass *vkRenderPass) vkResult
	destroyRenderPass                       func(device vkDevice, renderPass vkRenderPass, allocator unsafe.Pointer)
	createFramebuffer                       func(device vkDevice, createInfo *vkFramebufferCreateInfo, allocator unsafe.Pointer, framebuffer *vkFramebuffer) vkResult
	destroyFramebuffer                      func(device vkDevice, framebuffer vkFramebuffer, allocator unsafe.Pointer)
	createSemaphore                         func(device vkDevice, createInfo *vkSemaphoreCreateInfo, allocator unsafe.Pointer, semaphore *vkSemaphore) vkResult
	destroySemaphore                        func(device vkDevice, semaphore vkSemaphore, allocator unsafe.Pointer)
	createFence                             func(device vkDevice, createInfo *vkFenceCreateInfo, allocator unsafe.Pointer, fence *vkFence) vkResult
	destroyFence                            func(device vkDevice, fence vkFence, allocator unsafe.Pointer)
	waitForFences                           func(device vkDevice, fenceCount uint32, fences *vkFence, waitAll uint32, timeout uint64) vkResult
	resetFences                             func(device vkDevice, fenceCount uint32, fences *vkFence) vkResult
	createCommandPool                       func(device vkDevice, createInfo *vkCommandPoolCreateInfo, allocator unsafe.Pointer, commandPool *vkCommandPool) vkResult
	destroyCommandPool                      func(device vkDevice, commandPool vkCommandPool, allocator unsafe.Pointer)
	allocateCommandBuffers                  func(device vkDevice, createInfo *vkCommandBufferAllocateInfo, commandBuffers *vkCommandBuffer) vkResult
	freeCommandBuffers                      func(device vkDevice, commandPool vkCommandPool, commandBufferCount uint32, commandBuffers *vkCommandBuffer)
	beginCommandBuffer                      func(commandBuffer vkCommandBuffer, beginInfo *vkCommandBufferBeginInfo) vkResult
	endCommandBuffer                        func(commandBuffer vkCommandBuffer) vkResult
	resetCommandBuffer                      func(commandBuffer vkCommandBuffer, flags uint32) vkResult
	cmdBeginRenderPass                      func(commandBuffer vkCommandBuffer, beginInfo *vkRenderPassBeginInfo, contents uint32)
	cmdEndRenderPass                        func(commandBuffer vkCommandBuffer)
	cmdBindPipeline                         func(commandBuffer vkCommandBuffer, bindPoint uint32, pipeline vkPipeline)
	cmdSetViewport                          func(commandBuffer vkCommandBuffer, firstViewport uint32, viewportCount uint32, viewports *vkViewport)
	cmdSetScissor                           func(commandBuffer vkCommandBuffer, firstScissor uint32, scissorCount uint32, scissors *vkRect2D)
	cmdBindVertexBuffers                    func(commandBuffer vkCommandBuffer, firstBinding uint32, bindingCount uint32, buffers *vkBuffer, offsets *vkDeviceSize)
	cmdBindDescriptorSets                   func(commandBuffer vkCommandBuffer, bindPoint uint32, layout vkPipelineLayout, firstSet uint32, descriptorSetCount uint32, descriptorSets *vkDescriptorSet, dynamicOffsetCount uint32, dynamicOffsets *uint32)
	cmdDraw                                 func(commandBuffer vkCommandBuffer, vertexCount uint32, instanceCount uint32, firstVertex uint32, firstInstance uint32)
	createShaderModule                      func(device vkDevice, createInfo *vkShaderModuleCreateInfo, allocator unsafe.Pointer, shaderModule *vkShaderModule) vkResult
	destroyShaderModule                     func(device vkDevice, shaderModule vkShaderModule, allocator unsafe.Pointer)
	createPipelineLayout                    func(device vkDevice, createInfo *vkPipelineLayoutCreateInfo, allocator unsafe.Pointer, layout *vkPipelineLayout) vkResult
	destroyPipelineLayout                   func(device vkDevice, layout vkPipelineLayout, allocator unsafe.Pointer)
	createGraphicsPipelines                 func(device vkDevice, cache vkPipelineCache, createInfoCount uint32, createInfos *vkGraphicsPipelineCreateInfo, allocator unsafe.Pointer, pipelines *vkPipeline) vkResult
	destroyPipeline                         func(device vkDevice, pipeline vkPipeline, allocator unsafe.Pointer)
	createDescriptorSetLayout               func(device vkDevice, createInfo *vkDescriptorSetLayoutCreateInfo, allocator unsafe.Pointer, setLayout *vkDescriptorSetLayout) vkResult
	destroyDescriptorSetLayout              func(device vkDevice, setLayout vkDescriptorSetLayout, allocator unsafe.Pointer)
	createDescriptorPool                    func(device vkDevice, createInfo *vkDescriptorPoolCreateInfo, allocator unsafe.Pointer, descriptorPool *vkDescriptorPool) vkResult
	destroyDescriptorPool                   func(device vkDevice, descriptorPool vkDescriptorPool, allocator unsafe.Pointer)
	allocateDescriptorSets                  func(device vkDevice, createInfo *vkDescriptorSetAllocateInfo, descriptorSets *vkDescriptorSet) vkResult
	freeDescriptorSets                      func(device vkDevice, descriptorPool vkDescriptorPool, descriptorSetCount uint32, descriptorSets *vkDescriptorSet) vkResult
	updateDescriptorSets                    func(device vkDevice, descriptorWriteCount uint32, descriptorWrites *vkWriteDescriptorSet, descriptorCopyCount uint32, descriptorCopies unsafe.Pointer)
	createSampler                           func(device vkDevice, createInfo *vkSamplerCreateInfo, allocator unsafe.Pointer, sampler *vkSampler) vkResult
	destroySampler                          func(device vkDevice, sampler vkSampler, allocator unsafe.Pointer)
	createBuffer                            func(device vkDevice, createInfo *vkBufferCreateInfo, allocator unsafe.Pointer, buffer *vkBuffer) vkResult
	destroyBuffer                           func(device vkDevice, buffer vkBuffer, allocator unsafe.Pointer)
	getBufferMemoryRequirements             func(device vkDevice, buffer vkBuffer, memoryRequirements *vkMemoryRequirements)
	allocateMemory                          func(device vkDevice, createInfo *vkMemoryAllocateInfo, allocator unsafe.Pointer, memory *vkDeviceMemory) vkResult
	freeMemory                              func(device vkDevice, memory vkDeviceMemory, allocator unsafe.Pointer)
	bindBufferMemory                        func(device vkDevice, buffer vkBuffer, memory vkDeviceMemory, memoryOffset vkDeviceSize) vkResult
	mapMemory                               func(device vkDevice, memory vkDeviceMemory, offset vkDeviceSize, size vkDeviceSize, flags uint32, data *unsafe.Pointer) vkResult
	unmapMemory                             func(device vkDevice, memory vkDeviceMemory)
	createImage                             func(device vkDevice, createInfo *vkImageCreateInfo, allocator unsafe.Pointer, image *vkImage) vkResult
	destroyImage                            func(device vkDevice, image vkImage, allocator unsafe.Pointer)
	getImageMemoryRequirements              func(device vkDevice, image vkImage, memoryRequirements *vkMemoryRequirements)
	bindImageMemory                         func(device vkDevice, image vkImage, memory vkDeviceMemory, memoryOffset vkDeviceSize) vkResult
	cmdPipelineBarrier                      func(commandBuffer vkCommandBuffer, srcStageMask uint32, dstStageMask uint32, dependencyFlags uint32, memoryBarrierCount uint32, memoryBarriers unsafe.Pointer, bufferMemoryBarrierCount uint32, bufferMemoryBarriers unsafe.Pointer, imageMemoryBarrierCount uint32, imageMemoryBarriers *vkImageMemoryBarrier)
	cmdCopyBufferToImage                    func(commandBuffer vkCommandBuffer, srcBuffer vkBuffer, dstImage vkImage, dstImageLayout uint32, regionCount uint32, regions *vkBufferImageCopy)
}

type instance struct {
	api               *vulkanAPI
	handle            vkInstance
	enabledExtensions []string
	enabledLayers     []string
}

func NewInstance(opts Options) (rhi.Instance, error) {
	api, err := loadDefaultAPI()
	if err != nil {
		return nil, err
	}
	return newInstanceWithAPI(api, opts)
}

func newInstanceWithAPI(api *vulkanAPI, opts Options) (rhi.Instance, error) {
	normalized := normalizeOptions(opts)

	enabledExtensions, enabledLayers, err := resolveInstanceRequirements(api, normalized)
	if err != nil {
		return nil, err
	}

	instanceHandle, err := createVulkanInstance(api, normalized, enabledExtensions, enabledLayers)
	if err != nil {
		return nil, err
	}

	inst := &instance{
		api:               api,
		handle:            instanceHandle,
		enabledExtensions: enabledExtensions,
		enabledLayers:     enabledLayers,
	}
	runtime.SetFinalizer(inst, func(inst *instance) {
		_ = inst.Destroy()
	})
	return inst, nil
}

func (i *instance) Backend() rhi.BackendKind {
	return rhi.BackendKindVulkan
}

func (i *instance) Destroy() error {
	if i == nil || i.handle == 0 {
		return nil
	}
	i.api.destroyInstance(i.handle, nil)
	i.handle = 0
	runtime.SetFinalizer(i, nil)
	return nil
}

func normalizeOptions(opts Options) Options {
	if opts.ApplicationName == "" {
		opts.ApplicationName = "whisky game"
	}
	if opts.EngineName == "" {
		opts.EngineName = "whisky engine"
	}
	if opts.APIVersion == 0 {
		opts.APIVersion = vkAPIVersion10
	}
	return opts
}

func resolveInstanceRequirements(api *vulkanAPI, opts Options) ([]string, []string, error) {
	requiredExtensions := []string{}
	if opts.SurfaceTarget != nil {
		extensions, err := RequiredInstanceExtensions(*opts.SurfaceTarget, opts)
		if err != nil {
			return nil, nil, err
		}
		requiredExtensions = append(requiredExtensions, extensions...)
	} else if opts.EnableValidation {
		requiredExtensions = append(requiredExtensions, extDebugUtils)
	}

	requiredLayers := ValidationLayers(opts)

	availableExtensions, err := enumerateInstanceExtensions(api)
	if err != nil {
		return nil, nil, err
	}
	for _, name := range requiredExtensions {
		if !contains(availableExtensions, name) {
			return nil, nil, fmt.Errorf("%w: %s", ErrMissingExtension, name)
		}
	}

	availableLayers, err := enumerateInstanceLayers(api)
	if err != nil {
		return nil, nil, err
	}
	for _, name := range requiredLayers {
		if !contains(availableLayers, name) {
			return nil, nil, fmt.Errorf("%w: %s", ErrMissingLayer, name)
		}
	}

	return requiredExtensions, requiredLayers, nil
}

func createVulkanInstance(api *vulkanAPI, opts Options, enabledExtensions, enabledLayers []string) (vkInstance, error) {
	appNameBytes, appName := cString(opts.ApplicationName)
	engineNameBytes, engineName := cString(opts.EngineName)
	layerStorage, layerPointers := cStringList(enabledLayers)
	extStorage, extPointers := cStringList(enabledExtensions)

	appInfo := vkApplicationInfo{
		SType:            vkStructureTypeApplicationInfo,
		PApplicationName: appName,
		PEngineName:      engineName,
		APIVersion:       opts.APIVersion,
	}
	createInfo := vkInstanceCreateInfo{
		SType:                   vkStructureTypeInstanceCreateInfo,
		PApplicationInfo:        &appInfo,
		EnabledLayerCount:       uint32(len(layerPointers)),
		PpEnabledLayerNames:     bytePtrPtr(layerPointers),
		EnabledExtensionCount:   uint32(len(extPointers)),
		PpEnabledExtensionNames: bytePtrPtr(extPointers),
	}

	var handle vkInstance
	result := api.createInstance(&createInfo, nil, &handle)

	runtime.KeepAlive(appNameBytes)
	runtime.KeepAlive(engineNameBytes)
	runtime.KeepAlive(layerStorage)
	runtime.KeepAlive(layerPointers)
	runtime.KeepAlive(extStorage)
	runtime.KeepAlive(extPointers)

	if result != vkSuccess {
		if result == vkErrorLayerAbsent {
			return 0, fmt.Errorf("%w: validation layer rejected by loader", ErrMissingLayer)
		}
		if result == vkErrorExtAbsent {
			return 0, fmt.Errorf("%w: extension rejected by loader", ErrMissingExtension)
		}
		return 0, fmt.Errorf("%w: %s", ErrCreateInstance, result)
	}

	return handle, nil
}

func enumerateInstanceExtensions(api *vulkanAPI) ([]string, error) {
	var count uint32
	result := api.enumerateInstanceExtensionProperties(nil, &count, nil)
	if result != vkSuccess {
		return nil, fmt.Errorf("%w: enumerate extensions count: %s", ErrUnavailable, result)
	}
	if count == 0 {
		return nil, nil
	}

	props := make([]vkExtensionProperties, count)
	result = api.enumerateInstanceExtensionProperties(nil, &count, &props[0])
	if result != vkSuccess && result != vkIncomplete {
		return nil, fmt.Errorf("%w: enumerate extensions: %s", ErrUnavailable, result)
	}

	names := make([]string, 0, count)
	for _, prop := range props[:count] {
		names = append(names, bytesToString(prop.ExtensionName[:]))
	}
	return names, nil
}

func enumerateInstanceLayers(api *vulkanAPI) ([]string, error) {
	var count uint32
	result := api.enumerateInstanceLayerProperties(&count, nil)
	if result != vkSuccess {
		return nil, fmt.Errorf("%w: enumerate layers count: %s", ErrUnavailable, result)
	}
	if count == 0 {
		return nil, nil
	}

	props := make([]vkLayerProperties, count)
	result = api.enumerateInstanceLayerProperties(&count, &props[0])
	if result != vkSuccess && result != vkIncomplete {
		return nil, fmt.Errorf("%w: enumerate layers: %s", ErrUnavailable, result)
	}

	names := make([]string, 0, count)
	for _, prop := range props[:count] {
		names = append(names, bytesToString(prop.LayerName[:]))
	}
	return names, nil
}

func cString(s string) ([]byte, *byte) {
	buf := append([]byte(s), 0)
	return buf, &buf[0]
}

func cStringList(values []string) ([][]byte, []*byte) {
	if len(values) == 0 {
		return nil, nil
	}
	storage := make([][]byte, 0, len(values))
	ptrs := make([]*byte, 0, len(values))
	for _, value := range values {
		buf, ptr := cString(value)
		storage = append(storage, buf)
		ptrs = append(ptrs, ptr)
	}
	return storage, ptrs
}

func bytePtrPtr(values []*byte) **byte {
	if len(values) == 0 {
		return nil
	}
	return (**byte)(unsafe.Pointer(unsafe.SliceData(values)))
}

func bytesToString(raw []byte) string {
	end := 0
	for end < len(raw) && raw[end] != 0 {
		end++
	}
	return string(raw[:end])
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func (r vkResult) String() string {
	switch r {
	case vkSuccess:
		return "VK_SUCCESS"
	case vkIncomplete:
		return "VK_INCOMPLETE"
	case vkErrorLayerAbsent:
		return "VK_ERROR_LAYER_NOT_PRESENT"
	case vkErrorExtAbsent:
		return "VK_ERROR_EXTENSION_NOT_PRESENT"
	case vkSuboptimalKHR:
		return "VK_SUBOPTIMAL_KHR"
	case vkErrorOutOfDateKHR:
		return "VK_ERROR_OUT_OF_DATE_KHR"
	default:
		return fmt.Sprintf("VkResult(%d)", int32(r))
	}
}
