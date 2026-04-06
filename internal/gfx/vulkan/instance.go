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
	default:
		return fmt.Sprintf("VkResult(%d)", int32(r))
	}
}
