package vulkan

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
)

const (
	extSwapchain = "VK_KHR_swapchain"

	vkQueueGraphicsBit = 0x00000001

	vkPhysicalDeviceTypeIntegratedGPU = 1
	vkPhysicalDeviceTypeDiscreteGPU   = 2

	vkStructureTypeDeviceQueueCreateInfo = 2
	vkStructureTypeDeviceCreateInfo      = 3
)

type vkPhysicalDevice uintptr
type vkDevice uintptr
type vkQueue uintptr

type vkDeviceQueueCreateInfo struct {
	SType            int32
	_                [4]byte
	PNext            unsafe.Pointer
	Flags            uint32
	QueueFamilyIndex uint32
	QueueCount       uint32
	_                [4]byte
	PQueuePriorities *float32
}

type vkDeviceCreateInfo struct {
	SType                   int32
	_                       [4]byte
	PNext                   unsafe.Pointer
	Flags                   uint32
	QueueCreateInfoCount    uint32
	PQueueCreateInfos       *vkDeviceQueueCreateInfo
	EnabledLayerCount       uint32
	PpEnabledLayerNames     **byte
	EnabledExtensionCount   uint32
	PpEnabledExtensionNames **byte
	PEnabledFeatures        unsafe.Pointer
}

type vkQueueFamilyProperties struct {
	QueueFlags                  uint32
	QueueCount                  uint32
	TimestampValidBits          uint32
	MinImageTransferGranularity vkExtent3D
}

type vkExtent3D struct {
	Width  uint32
	Height uint32
	Depth  uint32
}

type vkPhysicalDevicePropertiesHeader struct {
	APIVersion    uint32
	DriverVersion uint32
	VendorID      uint32
	DeviceID      uint32
	DeviceType    uint32
	DeviceName    [256]byte
}

type device struct {
	api                *vulkanAPI
	physicalDevice     vkPhysicalDevice
	memoryProperties   vkPhysicalDeviceMemoryProperties
	handle             vkDevice
	surface            *surface
	graphicsQueue      vkQueue
	presentQueue       vkQueue
	graphicsQueueIndex uint32
	presentQueueIndex  uint32
}

type deviceCandidate struct {
	physicalDevice    vkPhysicalDevice
	name              string
	deviceType        uint32
	graphicsQueue     uint32
	presentQueue      uint32
	hasGraphicsQueue  bool
	hasPresentQueue   bool
	hasSwapchainExt   bool
	preferDiscreteGPU bool
}

func (i *instance) CreateDevice(surface rhi.Surface, opts rhi.DeviceOptions) (rhi.Device, error) {
	vkSurface, err := requireSurface(surface)
	if err != nil {
		return nil, err
	}

	candidates, err := enumerateDeviceCandidates(i.api, i.handle, vkSurface)
	if err != nil {
		return nil, err
	}
	candidate, err := selectDeviceCandidate(candidates, opts)
	if err != nil {
		return nil, err
	}

	deviceHandle, graphicsQueue, presentQueue, err := createLogicalDevice(i.api, candidate)
	if err != nil {
		return nil, err
	}

	dev := &device{
		api:                i.api,
		physicalDevice:     candidate.physicalDevice,
		memoryProperties:   readPhysicalDeviceMemoryProperties(i.api, candidate.physicalDevice),
		handle:             deviceHandle,
		surface:            vkSurface,
		graphicsQueue:      graphicsQueue,
		presentQueue:       presentQueue,
		graphicsQueueIndex: candidate.graphicsQueue,
		presentQueueIndex:  candidate.presentQueue,
	}
	runtime.SetFinalizer(dev, func(dev *device) {
		_ = dev.Destroy()
	})
	return dev, nil
}

func (d *device) Backend() rhi.BackendKind {
	return rhi.BackendKindVulkan
}

func (d *device) WaitIdle() error {
	if d == nil || d.handle == 0 {
		return nil
	}
	if d.api.deviceWaitIdle == nil {
		return ErrNotImplemented
	}
	if result := d.api.deviceWaitIdle(d.handle); result != vkSuccess {
		return fmt.Errorf("%w: %s", ErrDeviceWaitIdle, result)
	}
	return nil
}

func (d *device) Destroy() error {
	if d == nil || d.handle == 0 {
		return nil
	}
	_ = d.WaitIdle()
	d.api.destroyDevice(d.handle, nil)
	d.handle = 0
	d.graphicsQueue = 0
	d.presentQueue = 0
	runtime.SetFinalizer(d, nil)
	return nil
}

func requireSurface(value rhi.Surface) (*surface, error) {
	if value == nil {
		return nil, fmt.Errorf("%w: nil surface", ErrSurfaceUnsupported)
	}
	surface, ok := value.(*surface)
	if !ok {
		return nil, fmt.Errorf("%w: expected Vulkan surface, got %T", ErrSurfaceUnsupported, value)
	}
	if surface.handle == 0 {
		return nil, fmt.Errorf("%w: surface handle is invalid", ErrSurfaceUnsupported)
	}
	return surface, nil
}

func requireDevice(value rhi.Device) (*device, error) {
	if value == nil {
		return nil, fmt.Errorf("%w: nil device", ErrCreateDevice)
	}
	device, ok := value.(*device)
	if !ok {
		return nil, fmt.Errorf("%w: expected Vulkan device, got %T", ErrCreateDevice, value)
	}
	if device.handle == 0 {
		return nil, fmt.Errorf("%w: device handle is invalid", ErrCreateDevice)
	}
	return device, nil
}

func enumerateDeviceCandidates(api *vulkanAPI, instance vkInstance, surface *surface) ([]deviceCandidate, error) {
	physicalDevices, err := enumeratePhysicalDevices(api, instance)
	if err != nil {
		return nil, err
	}
	if len(physicalDevices) == 0 {
		return nil, ErrNoPhysicalDevice
	}

	candidates := make([]deviceCandidate, 0, len(physicalDevices))
	for _, physicalDevice := range physicalDevices {
		candidate, err := inspectDeviceCandidate(api, physicalDevice, surface)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

func inspectDeviceCandidate(api *vulkanAPI, physicalDevice vkPhysicalDevice, surface *surface) (deviceCandidate, error) {
	props := readPhysicalDeviceProperties(api, physicalDevice)
	extensions, err := enumerateDeviceExtensions(api, physicalDevice)
	if err != nil {
		return deviceCandidate{}, err
	}
	queueFamilies, err := enumerateQueueFamilies(api, physicalDevice)
	if err != nil {
		return deviceCandidate{}, err
	}

	candidate := deviceCandidate{
		physicalDevice:    physicalDevice,
		name:              props.name,
		deviceType:        props.deviceType,
		hasSwapchainExt:   contains(extensions, extSwapchain),
		preferDiscreteGPU: props.deviceType == vkPhysicalDeviceTypeDiscreteGPU,
	}

	for index, family := range queueFamilies {
		if !candidate.hasGraphicsQueue && family.QueueCount > 0 && family.QueueFlags&vkQueueGraphicsBit != 0 {
			candidate.graphicsQueue = uint32(index)
			candidate.hasGraphicsQueue = true
		}

		supported, err := getSurfaceSupport(api, physicalDevice, uint32(index), surface.handle)
		if err != nil {
			return deviceCandidate{}, err
		}
		if !candidate.hasPresentQueue && supported {
			candidate.presentQueue = uint32(index)
			candidate.hasPresentQueue = true
		}
	}

	return candidate, nil
}

func readPhysicalDeviceMemoryProperties(api *vulkanAPI, physicalDevice vkPhysicalDevice) vkPhysicalDeviceMemoryProperties {
	var properties vkPhysicalDeviceMemoryProperties
	if api.getPhysicalDeviceMemoryProperties != nil {
		api.getPhysicalDeviceMemoryProperties(physicalDevice, &properties)
	}
	return properties
}

func selectDeviceCandidate(candidates []deviceCandidate, opts rhi.DeviceOptions) (deviceCandidate, error) {
	var selected *deviceCandidate
	for index := range candidates {
		candidate := &candidates[index]
		if !candidate.hasGraphicsQueue || !candidate.hasPresentQueue || !candidate.hasSwapchainExt {
			continue
		}
		if selected == nil {
			selected = candidate
			continue
		}
		if opts.PreferDiscreteGPU && candidate.preferDiscreteGPU && !selected.preferDiscreteGPU {
			selected = candidate
		}
	}

	if selected == nil {
		return deviceCandidate{}, ErrNoSuitableDevice
	}
	return *selected, nil
}

func createLogicalDevice(api *vulkanAPI, candidate deviceCandidate) (vkDevice, vkQueue, vkQueue, error) {
	if api.createDevice == nil || api.destroyDevice == nil || api.getDeviceQueue == nil {
		return 0, 0, 0, ErrNotImplemented
	}

	queueIndices := uniqueQueueFamilyIndices(candidate.graphicsQueue, candidate.presentQueue)
	priority := float32(1.0)
	queueInfos := make([]vkDeviceQueueCreateInfo, 0, len(queueIndices))
	for _, index := range queueIndices {
		queueInfos = append(queueInfos, vkDeviceQueueCreateInfo{
			SType:            vkStructureTypeDeviceQueueCreateInfo,
			QueueFamilyIndex: index,
			QueueCount:       1,
			PQueuePriorities: &priority,
		})
	}

	extStorage, extPointers := cStringList([]string{extSwapchain})
	createInfo := vkDeviceCreateInfo{
		SType:                   vkStructureTypeDeviceCreateInfo,
		QueueCreateInfoCount:    uint32(len(queueInfos)),
		PQueueCreateInfos:       unsafe.SliceData(queueInfos),
		EnabledExtensionCount:   uint32(len(extPointers)),
		PpEnabledExtensionNames: bytePtrPtr(extPointers),
	}

	var handle vkDevice
	result := api.createDevice(candidate.physicalDevice, &createInfo, nil, &handle)
	runtime.KeepAlive(extStorage)
	runtime.KeepAlive(extPointers)
	runtime.KeepAlive(queueInfos)
	runtime.KeepAlive(priority)
	if result != vkSuccess {
		return 0, 0, 0, fmt.Errorf("%w: %s", ErrCreateDevice, result)
	}

	var graphicsQueue vkQueue
	var presentQueue vkQueue
	api.getDeviceQueue(handle, candidate.graphicsQueue, 0, &graphicsQueue)
	api.getDeviceQueue(handle, candidate.presentQueue, 0, &presentQueue)

	return handle, graphicsQueue, presentQueue, nil
}

func enumeratePhysicalDevices(api *vulkanAPI, instance vkInstance) ([]vkPhysicalDevice, error) {
	var count uint32
	result := api.enumeratePhysicalDevices(instance, &count, nil)
	if result != vkSuccess {
		return nil, fmt.Errorf("%w: enumerate physical devices count: %s", ErrNoPhysicalDevice, result)
	}
	if count == 0 {
		return nil, nil
	}

	devices := make([]vkPhysicalDevice, count)
	result = api.enumeratePhysicalDevices(instance, &count, &devices[0])
	if result != vkSuccess && result != vkIncomplete {
		return nil, fmt.Errorf("%w: enumerate physical devices: %s", ErrNoPhysicalDevice, result)
	}
	return devices[:count], nil
}

type physicalDeviceProperties struct {
	deviceType uint32
	name       string
}

func readPhysicalDeviceProperties(api *vulkanAPI, physicalDevice vkPhysicalDevice) physicalDeviceProperties {
	if api.getPhysicalDeviceProperties == nil {
		return physicalDeviceProperties{}
	}

	raw := make([]byte, 4096)
	api.getPhysicalDeviceProperties(physicalDevice, unsafe.Pointer(unsafe.SliceData(raw)))
	header := (*vkPhysicalDevicePropertiesHeader)(unsafe.Pointer(unsafe.SliceData(raw)))
	return physicalDeviceProperties{
		deviceType: header.DeviceType,
		name:       bytesToString(header.DeviceName[:]),
	}
}

func enumerateQueueFamilies(api *vulkanAPI, physicalDevice vkPhysicalDevice) ([]vkQueueFamilyProperties, error) {
	var count uint32
	api.getPhysicalDeviceQueueFamilyProperties(physicalDevice, &count, nil)
	if count == 0 {
		return nil, nil
	}

	families := make([]vkQueueFamilyProperties, count)
	api.getPhysicalDeviceQueueFamilyProperties(physicalDevice, &count, &families[0])
	return families[:count], nil
}

func enumerateDeviceExtensions(api *vulkanAPI, physicalDevice vkPhysicalDevice) ([]string, error) {
	var count uint32
	result := api.enumerateDeviceExtensionProperties(physicalDevice, nil, &count, nil)
	if result != vkSuccess {
		return nil, fmt.Errorf("%w: enumerate device extensions count: %s", ErrMissingDeviceExtension, result)
	}
	if count == 0 {
		return nil, nil
	}

	props := make([]vkExtensionProperties, count)
	result = api.enumerateDeviceExtensionProperties(physicalDevice, nil, &count, &props[0])
	if result != vkSuccess && result != vkIncomplete {
		return nil, fmt.Errorf("%w: enumerate device extensions: %s", ErrMissingDeviceExtension, result)
	}

	names := make([]string, 0, count)
	for _, prop := range props[:count] {
		names = append(names, bytesToString(prop.ExtensionName[:]))
	}
	return names, nil
}

func getSurfaceSupport(api *vulkanAPI, physicalDevice vkPhysicalDevice, queueFamilyIndex uint32, surface vkSurfaceKHR) (bool, error) {
	if api.getPhysicalDeviceSurfaceSupportKHR == nil {
		return false, ErrSurfaceUnsupported
	}
	var supported uint32
	result := api.getPhysicalDeviceSurfaceSupportKHR(physicalDevice, queueFamilyIndex, surface, &supported)
	if result != vkSuccess {
		return false, fmt.Errorf("%w: queue family %d present support: %s", ErrNoQueueFamily, queueFamilyIndex, result)
	}
	return supported != 0, nil
}

func uniqueQueueFamilyIndices(values ...uint32) []uint32 {
	if len(values) == 0 {
		return nil
	}

	result := make([]uint32, 0, len(values))
	seen := make(map[uint32]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
