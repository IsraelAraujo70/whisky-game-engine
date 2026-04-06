package vulkan

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

const (
	vkStructureTypeSwapchainCreateInfoKHR = 1000001000

	vkImageUsageColorAttachmentBit = 0x00000010

	vkSharingModeExclusive  = 0
	vkSharingModeConcurrent = 1

	vkCompositeAlphaOpaqueBitKHR = 0x00000001

	vkFormatR8G8B8A8Unorm = 37
	vkFormatB8G8R8A8Unorm = 44
	vkFormatR8G8B8A8SRGB  = 43
	vkFormatB8G8R8A8SRGB  = 50

	vkColorSpaceSRGBNonlinearKHR = 0

	vkPresentModeImmediateKHR = 0
	vkPresentModeMailboxKHR   = 1
	vkPresentModeFIFOKHR      = 2

	vkSurfaceTransformIdentityBitKHR = 0x00000001

	vkUndefinedExtent = ^uint32(0)
)

type vkSwapchainKHR uintptr

type vkExtent2D struct {
	Width  uint32
	Height uint32
}

type vkSurfaceCapabilitiesKHR struct {
	MinImageCount           uint32
	MaxImageCount           uint32
	CurrentExtent           vkExtent2D
	MinImageExtent          vkExtent2D
	MaxImageExtent          vkExtent2D
	MaxImageArrayLayers     uint32
	SupportedTransforms     uint32
	CurrentTransform        uint32
	SupportedCompositeAlpha uint32
	SupportedUsageFlags     uint32
}

type vkSurfaceFormatKHR struct {
	Format     int32
	ColorSpace int32
}

type vkSwapchainCreateInfoKHR struct {
	SType                 int32
	_                     [4]byte
	PNext                 unsafe.Pointer
	Flags                 uint32
	_                     [4]byte
	Surface               vkSurfaceKHR
	MinImageCount         uint32
	ImageFormat           int32
	ImageColorSpace       int32
	ImageExtent           vkExtent2D
	ImageArrayLayers      uint32
	ImageUsage            uint32
	ImageSharingMode      uint32
	QueueFamilyIndexCount uint32
	PQueueFamilyIndices   *uint32
	PreTransform          uint32
	CompositeAlpha        uint32
	PresentMode           int32
	Clipped               uint32
	_                     [4]byte
	OldSwapchain          vkSwapchainKHR
}

type swapchain struct {
	api    *vulkanAPI
	device *device
	handle vkSwapchainKHR
	desc   rhi.SwapchainDescriptor
}

func (d *device) CreateSwapchain(surface rhi.Surface, desc rhi.SwapchainDescriptor) (rhi.Swapchain, error) {
	vkSurface, err := requireSurface(surface)
	if err != nil {
		return nil, err
	}
	if d.surface != nil && d.surface.handle != vkSurface.handle {
		return nil, fmt.Errorf("%w: swapchain surface must belong to the same Vulkan device", ErrCreateSwapchain)
	}
	if d.api.createSwapchainKHR == nil || d.api.destroySwapchainKHR == nil {
		return nil, ErrNotImplemented
	}

	normalized, createInfo, err := d.buildSwapchainCreateInfo(vkSurface, desc, 0)
	if err != nil {
		return nil, err
	}

	var handle vkSwapchainKHR
	result := d.api.createSwapchainKHR(d.handle, &createInfo, nil, &handle)
	runtime.KeepAlive(createInfo)
	if result != vkSuccess {
		return nil, fmt.Errorf("%w: %s", ErrCreateSwapchain, result)
	}

	sc := &swapchain{
		api:    d.api,
		device: d,
		handle: handle,
		desc:   normalized,
	}
	runtime.SetFinalizer(sc, func(s *swapchain) {
		_ = s.Destroy()
	})
	return sc, nil
}

func (s *swapchain) Backend() rhi.BackendKind {
	return rhi.BackendKindVulkan
}

func (s *swapchain) Descriptor() rhi.SwapchainDescriptor {
	return s.desc
}

func (s *swapchain) Resize(width, height int) error {
	if s == nil || s.handle == 0 {
		return nil
	}
	if width <= 0 || height <= 0 {
		return fmt.Errorf("%w: resize extent must be positive, got %dx%d", ErrCreateSwapchain, width, height)
	}

	newDesc := s.desc
	newDesc.Extent = rhi.Extent2D{Width: width, Height: height}
	oldSwapchain := s.handle
	if s.requiresDestructiveResize() {
		if err := s.device.WaitIdle(); err != nil {
			return err
		}
		s.api.destroySwapchainKHR(s.device.handle, oldSwapchain, nil)
		s.handle = 0
		oldSwapchain = 0
	}

	createInfoDesc, createInfo, err := s.device.buildSwapchainCreateInfo(s.device.surface, newDesc, oldSwapchain)
	if err != nil {
		if s.handle == 0 {
			s.handle = oldSwapchain
		}
		return err
	}

	var next vkSwapchainKHR
	result := s.api.createSwapchainKHR(s.device.handle, &createInfo, nil, &next)
	runtime.KeepAlive(createInfo)
	if result != vkSuccess {
		if s.handle == 0 {
			s.handle = oldSwapchain
		}
		return fmt.Errorf("%w: %s", ErrCreateSwapchain, result)
	}

	if s.handle != 0 {
		s.api.destroySwapchainKHR(s.device.handle, s.handle, nil)
	}
	s.handle = next
	s.desc = createInfoDesc
	return nil
}

func (s *swapchain) requiresDestructiveResize() bool {
	return s != nil &&
		s.device != nil &&
		s.device.surface != nil &&
		s.device.surface.target.Window.Kind == platformapi.NativeWindowKindWayland
}

func (s *swapchain) Destroy() error {
	if s == nil || s.handle == 0 {
		return nil
	}
	s.api.destroySwapchainKHR(s.device.handle, s.handle, nil)
	s.handle = 0
	runtime.SetFinalizer(s, nil)
	return nil
}

func requireSwapchain(value rhi.Swapchain) (*swapchain, error) {
	if value == nil {
		return nil, fmt.Errorf("%w: nil swapchain", ErrCreateSwapchain)
	}
	swapchain, ok := value.(*swapchain)
	if !ok {
		return nil, fmt.Errorf("%w: expected Vulkan swapchain, got %T", ErrCreateSwapchain, value)
	}
	if swapchain.handle == 0 {
		return nil, fmt.Errorf("%w: swapchain handle is invalid", ErrCreateSwapchain)
	}
	return swapchain, nil
}

func (d *device) buildSwapchainCreateInfo(surface *surface, desc rhi.SwapchainDescriptor, oldSwapchain vkSwapchainKHR) (rhi.SwapchainDescriptor, vkSwapchainCreateInfoKHR, error) {
	normalized, err := rhi.NormalizeSwapchainDescriptor(desc, surface.target.Extent)
	if err != nil {
		return rhi.SwapchainDescriptor{}, vkSwapchainCreateInfoKHR{}, err
	}

	caps, err := getSurfaceCapabilities(d.api, d.physicalDevice, surface.handle)
	if err != nil {
		return rhi.SwapchainDescriptor{}, vkSwapchainCreateInfoKHR{}, err
	}
	formats, err := getSurfaceFormats(d.api, d.physicalDevice, surface.handle)
	if err != nil {
		return rhi.SwapchainDescriptor{}, vkSwapchainCreateInfoKHR{}, err
	}
	presentModes, err := getPresentModes(d.api, d.physicalDevice, surface.handle)
	if err != nil {
		return rhi.SwapchainDescriptor{}, vkSwapchainCreateInfoKHR{}, err
	}

	surfaceFormat, format, err := chooseSurfaceFormat(formats, normalized.Format)
	if err != nil {
		return rhi.SwapchainDescriptor{}, vkSwapchainCreateInfoKHR{}, err
	}
	presentMode, err := choosePresentMode(presentModes, normalized.PresentMode)
	if err != nil {
		return rhi.SwapchainDescriptor{}, vkSwapchainCreateInfoKHR{}, err
	}
	imageExtent := chooseSwapchainExtent(caps, normalized.Extent)
	imageCount := chooseImageCount(caps, normalized.BufferCount)
	sharingMode, queueFamilyIndices := d.swapchainSharingMode()

	normalized.Format = format
	normalized.PresentMode = normalized.PresentMode
	normalized.BufferCount = int(imageCount)
	normalized.Extent = rhi.Extent2D{Width: int(imageExtent.Width), Height: int(imageExtent.Height)}

	createInfo := vkSwapchainCreateInfoKHR{
		SType:                 vkStructureTypeSwapchainCreateInfoKHR,
		Surface:               surface.handle,
		MinImageCount:         imageCount,
		ImageFormat:           surfaceFormat.Format,
		ImageColorSpace:       surfaceFormat.ColorSpace,
		ImageExtent:           imageExtent,
		ImageArrayLayers:      1,
		ImageUsage:            vkImageUsageColorAttachmentBit,
		ImageSharingMode:      sharingMode,
		QueueFamilyIndexCount: uint32(len(queueFamilyIndices)),
		PreTransform:          choosePreTransform(caps),
		CompositeAlpha:        chooseCompositeAlpha(caps),
		PresentMode:           presentMode,
		Clipped:               1,
		OldSwapchain:          oldSwapchain,
	}
	if len(queueFamilyIndices) > 0 {
		createInfo.PQueueFamilyIndices = &queueFamilyIndices[0]
	}
	return normalized, createInfo, nil
}

func (d *device) swapchainSharingMode() (uint32, []uint32) {
	if d.graphicsQueueIndex == d.presentQueueIndex {
		return vkSharingModeExclusive, nil
	}
	indices := []uint32{d.graphicsQueueIndex, d.presentQueueIndex}
	return vkSharingModeConcurrent, indices
}

func getSurfaceCapabilities(api *vulkanAPI, physicalDevice vkPhysicalDevice, surface vkSurfaceKHR) (vkSurfaceCapabilitiesKHR, error) {
	var capabilities vkSurfaceCapabilitiesKHR
	result := api.getPhysicalDeviceSurfaceCapabilitiesKHR(physicalDevice, surface, &capabilities)
	if result != vkSuccess {
		return vkSurfaceCapabilitiesKHR{}, fmt.Errorf("%w: surface capabilities: %s", ErrCreateSwapchain, result)
	}
	return capabilities, nil
}

func getSurfaceFormats(api *vulkanAPI, physicalDevice vkPhysicalDevice, surface vkSurfaceKHR) ([]vkSurfaceFormatKHR, error) {
	var count uint32
	result := api.getPhysicalDeviceSurfaceFormatsKHR(physicalDevice, surface, &count, nil)
	if result != vkSuccess {
		return nil, fmt.Errorf("%w: surface formats count: %s", ErrCreateSwapchain, result)
	}
	if count == 0 {
		return nil, nil
	}

	formats := make([]vkSurfaceFormatKHR, count)
	result = api.getPhysicalDeviceSurfaceFormatsKHR(physicalDevice, surface, &count, &formats[0])
	if result != vkSuccess && result != vkIncomplete {
		return nil, fmt.Errorf("%w: surface formats: %s", ErrCreateSwapchain, result)
	}
	return formats[:count], nil
}

func getPresentModes(api *vulkanAPI, physicalDevice vkPhysicalDevice, surface vkSurfaceKHR) ([]int32, error) {
	var count uint32
	result := api.getPhysicalDeviceSurfacePresentModesKHR(physicalDevice, surface, &count, nil)
	if result != vkSuccess {
		return nil, fmt.Errorf("%w: present modes count: %s", ErrCreateSwapchain, result)
	}
	if count == 0 {
		return nil, nil
	}

	modes := make([]int32, count)
	result = api.getPhysicalDeviceSurfacePresentModesKHR(physicalDevice, surface, &count, &modes[0])
	if result != vkSuccess && result != vkIncomplete {
		return nil, fmt.Errorf("%w: present modes: %s", ErrCreateSwapchain, result)
	}
	return modes[:count], nil
}

func chooseSurfaceFormat(formats []vkSurfaceFormatKHR, requested rhi.PixelFormat) (vkSurfaceFormatKHR, rhi.PixelFormat, error) {
	wantFormat, err := mapPixelFormatToVulkan(requested)
	if err != nil {
		return vkSurfaceFormatKHR{}, rhi.PixelFormatUnknown, err
	}
	for _, format := range formats {
		if format.Format == wantFormat && format.ColorSpace == vkColorSpaceSRGBNonlinearKHR {
			return format, requested, nil
		}
	}
	if len(formats) == 0 {
		return vkSurfaceFormatKHR{}, rhi.PixelFormatUnknown, fmt.Errorf("%w: no surface formats reported", ErrCreateSwapchain)
	}
	if fallback, ok := mapVulkanFormatToPixelFormat(formats[0].Format); ok {
		return formats[0], fallback, nil
	}
	return vkSurfaceFormatKHR{}, rhi.PixelFormatUnknown, fmt.Errorf("%w: unsupported surface format %#x", ErrCreateSwapchain, formats[0].Format)
}

func choosePresentMode(modes []int32, requested rhi.PresentMode) (int32, error) {
	want := mapPresentModeToVulkan(requested)
	for _, mode := range modes {
		if mode == want {
			return mode, nil
		}
	}
	if requested == rhi.PresentModeFIFO {
		return vkPresentModeFIFOKHR, nil
	}
	return 0, fmt.Errorf("%w: present mode %q unavailable", ErrCreateSwapchain, requested)
}

func chooseSwapchainExtent(caps vkSurfaceCapabilitiesKHR, requested rhi.Extent2D) vkExtent2D {
	if caps.CurrentExtent.Width != vkUndefinedExtent {
		return caps.CurrentExtent
	}
	return vkExtent2D{
		Width:  clampUint32(uint32(requested.Width), caps.MinImageExtent.Width, caps.MaxImageExtent.Width),
		Height: clampUint32(uint32(requested.Height), caps.MinImageExtent.Height, caps.MaxImageExtent.Height),
	}
}

func chooseImageCount(caps vkSurfaceCapabilitiesKHR, requested int) uint32 {
	count := uint32(requested)
	if count < caps.MinImageCount {
		count = caps.MinImageCount
	}
	if caps.MaxImageCount != 0 && count > caps.MaxImageCount {
		count = caps.MaxImageCount
	}
	return count
}

func choosePreTransform(caps vkSurfaceCapabilitiesKHR) uint32 {
	if caps.CurrentTransform != 0 {
		return caps.CurrentTransform
	}
	if caps.SupportedTransforms&vkSurfaceTransformIdentityBitKHR != 0 {
		return vkSurfaceTransformIdentityBitKHR
	}
	return caps.SupportedTransforms
}

func chooseCompositeAlpha(caps vkSurfaceCapabilitiesKHR) uint32 {
	if caps.SupportedCompositeAlpha&vkCompositeAlphaOpaqueBitKHR != 0 {
		return vkCompositeAlphaOpaqueBitKHR
	}
	return caps.SupportedCompositeAlpha
}

func mapPixelFormatToVulkan(format rhi.PixelFormat) (int32, error) {
	switch format {
	case rhi.PixelFormatBGRA8Unorm:
		return vkFormatB8G8R8A8Unorm, nil
	case rhi.PixelFormatRGBA8Unorm:
		return vkFormatR8G8B8A8Unorm, nil
	case rhi.PixelFormatBGRA8SRGB:
		return vkFormatB8G8R8A8SRGB, nil
	case rhi.PixelFormatRGBA8SRGB:
		return vkFormatR8G8B8A8SRGB, nil
	default:
		return 0, fmt.Errorf("%w: unsupported pixel format %q", ErrCreateSwapchain, format)
	}
}

func mapVulkanFormatToPixelFormat(format int32) (rhi.PixelFormat, bool) {
	switch format {
	case vkFormatB8G8R8A8Unorm:
		return rhi.PixelFormatBGRA8Unorm, true
	case vkFormatR8G8B8A8Unorm:
		return rhi.PixelFormatRGBA8Unorm, true
	case vkFormatB8G8R8A8SRGB:
		return rhi.PixelFormatBGRA8SRGB, true
	case vkFormatR8G8B8A8SRGB:
		return rhi.PixelFormatRGBA8SRGB, true
	default:
		return rhi.PixelFormatUnknown, false
	}
}

func mapPresentModeToVulkan(mode rhi.PresentMode) int32 {
	switch mode {
	case rhi.PresentModeImmediate:
		return vkPresentModeImmediateKHR
	case rhi.PresentModeMailbox:
		return vkPresentModeMailboxKHR
	default:
		return vkPresentModeFIFOKHR
	}
}

func clampUint32(value, min, max uint32) uint32 {
	if value < min {
		return min
	}
	if max != 0 && value > max {
		return max
	}
	return value
}
