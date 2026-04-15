package vulkan

import (
	"errors"
	"testing"
	"unsafe"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

func TestInstanceCreateDevicePrefersDiscreteGPU(t *testing.T) {
	const (
		integrated vkPhysicalDevice = 0x1001
		discrete   vkPhysicalDevice = 0x1002
	)

	createdDevice := vkDevice(0x9001)
	inst := &instance{
		api: &vulkanAPI{
			enumeratePhysicalDevices: func(instance vkInstance, count *uint32, devices *vkPhysicalDevice) vkResult {
				if devices == nil {
					*count = 2
					return vkSuccess
				}
				values := unsafe.Slice(devices, *count)
				values[0] = integrated
				values[1] = discrete
				return vkSuccess
			},
			getPhysicalDeviceProperties: func(physicalDevice vkPhysicalDevice, properties unsafe.Pointer) {
				header := (*vkPhysicalDevicePropertiesHeader)(properties)
				switch physicalDevice {
				case integrated:
					header.DeviceType = vkPhysicalDeviceTypeIntegratedGPU
					copy(header.DeviceName[:], []byte("integrated"))
				case discrete:
					header.DeviceType = vkPhysicalDeviceTypeDiscreteGPU
					copy(header.DeviceName[:], []byte("discrete"))
				}
			},
			getPhysicalDeviceQueueFamilyProperties: func(physicalDevice vkPhysicalDevice, count *uint32, props *vkQueueFamilyProperties) {
				if props == nil {
					*count = 1
					return
				}
				values := unsafe.Slice(props, *count)
				values[0] = vkQueueFamilyProperties{QueueFlags: vkQueueGraphicsBit, QueueCount: 1}
			},
			enumerateDeviceExtensionProperties: func(physicalDevice vkPhysicalDevice, layerName *byte, count *uint32, props *vkExtensionProperties) vkResult {
				if props == nil {
					*count = 1
					return vkSuccess
				}
				values := unsafe.Slice(props, *count)
				copy(values[0].ExtensionName[:], []byte(extSwapchain))
				return vkSuccess
			},
			getPhysicalDeviceSurfaceSupportKHR: func(physicalDevice vkPhysicalDevice, queueFamilyIndex uint32, surface vkSurfaceKHR, supported *uint32) vkResult {
				*supported = 1
				return vkSuccess
			},
			createDevice: func(physicalDevice vkPhysicalDevice, createInfo *vkDeviceCreateInfo, allocator unsafe.Pointer, device *vkDevice) vkResult {
				if physicalDevice != discrete {
					t.Fatalf("expected discrete GPU to be selected, got %#x", physicalDevice)
				}
				*device = createdDevice
				return vkSuccess
			},
			getDeviceQueue: func(device vkDevice, queueFamilyIndex uint32, queueIndex uint32, queue *vkQueue) {
				*queue = vkQueue(0x7000 + uintptr(queueFamilyIndex))
			},
			destroyDevice: func(device vkDevice, allocator unsafe.Pointer) {},
			deviceWaitIdle: func(device vkDevice) vkResult {
				return vkSuccess
			},
		},
		handle: 0xCAFE,
	}

	surface := &surface{
		api:    inst.api,
		handle: 0xDEAD,
		target: rhi.SurfaceTarget{
			Window: platformapi.NativeWindowHandle{
				Kind:    platformapi.NativeWindowKindX11,
				Display: 0x1,
				Window:  0x2,
			},
			Extent: rhi.Extent2D{Width: 1280, Height: 720},
		},
	}

	deviceValue, err := inst.CreateDevice(surface, rhi.DeviceOptions{PreferDiscreteGPU: true})
	if err != nil {
		t.Fatalf("expected device creation to succeed, got %v", err)
	}
	defer deviceValue.Destroy()

	device := deviceValue.(*device)
	if device.handle != createdDevice {
		t.Fatalf("expected device handle %#x, got %#x", createdDevice, device.handle)
	}
	if device.graphicsQueueIndex != 0 || device.presentQueueIndex != 0 {
		t.Fatalf("expected queue family 0 for graphics/present, got %d/%d", device.graphicsQueueIndex, device.presentQueueIndex)
	}
}

func TestInstanceCreateDeviceRejectsMissingSwapchainExtension(t *testing.T) {
	inst := &instance{
		api: &vulkanAPI{
			enumeratePhysicalDevices: func(instance vkInstance, count *uint32, devices *vkPhysicalDevice) vkResult {
				if devices == nil {
					*count = 1
					return vkSuccess
				}
				unsafe.Slice(devices, *count)[0] = vkPhysicalDevice(0x1001)
				return vkSuccess
			},
			getPhysicalDeviceQueueFamilyProperties: func(physicalDevice vkPhysicalDevice, count *uint32, props *vkQueueFamilyProperties) {
				if props == nil {
					*count = 1
					return
				}
				unsafe.Slice(props, *count)[0] = vkQueueFamilyProperties{QueueFlags: vkQueueGraphicsBit, QueueCount: 1}
			},
			enumerateDeviceExtensionProperties: func(physicalDevice vkPhysicalDevice, layerName *byte, count *uint32, props *vkExtensionProperties) vkResult {
				*count = 0
				return vkSuccess
			},
			getPhysicalDeviceSurfaceSupportKHR: func(physicalDevice vkPhysicalDevice, queueFamilyIndex uint32, surface vkSurfaceKHR, supported *uint32) vkResult {
				*supported = 1
				return vkSuccess
			},
		},
		handle: 0xCAFE,
	}

	_, err := inst.CreateDevice(&surface{
		api:    inst.api,
		handle: 0xDEAD,
		target: rhi.SurfaceTarget{
			Window: platformapi.NativeWindowHandle{
				Kind:    platformapi.NativeWindowKindWayland,
				Display: 0x1,
				Window:  0x2,
			},
			Extent: rhi.Extent2D{Width: 640, Height: 480},
		},
	}, rhi.DeviceOptions{})
	if !errors.Is(err, ErrNoSuitableDevice) {
		t.Fatalf("expected ErrNoSuitableDevice, got %v", err)
	}
}

func TestDeviceCreateSwapchainUsesSurfaceCapabilities(t *testing.T) {
	var captured vkSwapchainCreateInfoKHR

	dev := &device{
		api: &vulkanAPI{
			getPhysicalDeviceSurfaceCapabilitiesKHR: func(physicalDevice vkPhysicalDevice, surface vkSurfaceKHR, capabilities *vkSurfaceCapabilitiesKHR) vkResult {
				*capabilities = vkSurfaceCapabilitiesKHR{
					MinImageCount:           2,
					MaxImageCount:           4,
					CurrentExtent:           vkExtent2D{Width: vkUndefinedExtent, Height: vkUndefinedExtent},
					MinImageExtent:          vkExtent2D{Width: 320, Height: 180},
					MaxImageExtent:          vkExtent2D{Width: 1920, Height: 1080},
					CurrentTransform:        vkSurfaceTransformIdentityBitKHR,
					SupportedTransforms:     vkSurfaceTransformIdentityBitKHR,
					SupportedCompositeAlpha: vkCompositeAlphaOpaqueBitKHR,
				}
				return vkSuccess
			},
			getPhysicalDeviceSurfaceFormatsKHR: func(physicalDevice vkPhysicalDevice, surface vkSurfaceKHR, count *uint32, formats *vkSurfaceFormatKHR) vkResult {
				if formats == nil {
					*count = 1
					return vkSuccess
				}
				unsafe.Slice(formats, *count)[0] = vkSurfaceFormatKHR{
					Format:     vkFormatB8G8R8A8Unorm,
					ColorSpace: vkColorSpaceSRGBNonlinearKHR,
				}
				return vkSuccess
			},
			getPhysicalDeviceSurfacePresentModesKHR: func(physicalDevice vkPhysicalDevice, surface vkSurfaceKHR, count *uint32, modes *int32) vkResult {
				if modes == nil {
					*count = 2
					return vkSuccess
				}
				values := unsafe.Slice(modes, *count)
				values[0] = vkPresentModeFIFOKHR
				values[1] = vkPresentModeMailboxKHR
				return vkSuccess
			},
			createSwapchainKHR: func(device vkDevice, createInfo *vkSwapchainCreateInfoKHR, allocator unsafe.Pointer, swapchain *vkSwapchainKHR) vkResult {
				captured = *createInfo
				*swapchain = vkSwapchainKHR(0xABCD)
				return vkSuccess
			},
			destroySwapchainKHR: func(device vkDevice, swapchain vkSwapchainKHR, allocator unsafe.Pointer) {},
		},
		physicalDevice:     0x1234,
		handle:             0x5678,
		graphicsQueueIndex: 2,
		presentQueueIndex:  3,
	}
	dev.surface = &surface{
		handle: 0x9999,
		target: rhi.SurfaceTarget{
			Window: platformapi.NativeWindowHandle{
				Kind:    platformapi.NativeWindowKindX11,
				Display: 0x1,
				Window:  0x2,
			},
			Extent: rhi.Extent2D{Width: 1280, Height: 720},
		},
	}

	swapchainValue, err := dev.CreateSwapchain(dev.surface, rhi.SwapchainDescriptor{
		PresentMode: rhi.PresentModeMailbox,
		BufferCount: 3,
	})
	if err != nil {
		t.Fatalf("expected swapchain creation to succeed, got %v", err)
	}
	defer swapchainValue.Destroy()

	if captured.MinImageCount != 3 {
		t.Fatalf("expected min image count 3, got %d", captured.MinImageCount)
	}
	if captured.ImageFormat != vkFormatB8G8R8A8Unorm {
		t.Fatalf("expected BGRA8 swapchain format, got %#x", captured.ImageFormat)
	}
	if captured.PresentMode != vkPresentModeMailboxKHR {
		t.Fatalf("expected mailbox present mode, got %#x", captured.PresentMode)
	}
	if captured.ImageSharingMode != vkSharingModeConcurrent {
		t.Fatalf("expected concurrent sharing mode, got %d", captured.ImageSharingMode)
	}
	if captured.QueueFamilyIndexCount != 2 {
		t.Fatalf("expected two queue family indices, got %d", captured.QueueFamilyIndexCount)
	}

	got := swapchainValue.Descriptor()
	if got.Extent.Width != 1280 || got.Extent.Height != 720 {
		t.Fatalf("expected fallback extent 1280x720, got %dx%d", got.Extent.Width, got.Extent.Height)
	}
	if got.BufferCount != 3 {
		t.Fatalf("expected buffer count 3, got %d", got.BufferCount)
	}
}
