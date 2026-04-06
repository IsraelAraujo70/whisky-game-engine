//go:build windows

package vulkan

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/windows"
)

var (
	defaultAPIOnce sync.Once
	defaultAPI     *vulkanAPI
	defaultAPIErr  error
)

func loadDefaultAPI() (*vulkanAPI, error) {
	defaultAPIOnce.Do(func() {
		dll := windows.NewLazyDLL("vulkan-1.dll")
		api := &vulkanAPI{}

		if err := registerProc(dll, "vkEnumerateInstanceExtensionProperties", &api.enumerateInstanceExtensionProperties); err != nil {
			defaultAPIErr = err
			return
		}
		if err := registerProc(dll, "vkEnumerateInstanceLayerProperties", &api.enumerateInstanceLayerProperties); err != nil {
			defaultAPIErr = err
			return
		}
		if err := registerProc(dll, "vkCreateInstance", &api.createInstance); err != nil {
			defaultAPIErr = err
			return
		}
		if err := registerProc(dll, "vkDestroyInstance", &api.destroyInstance); err != nil {
			defaultAPIErr = err
			return
		}
		tryRegisterProc(dll, "vkEnumeratePhysicalDevices", &api.enumeratePhysicalDevices)
		tryRegisterProc(dll, "vkGetPhysicalDeviceProperties", &api.getPhysicalDeviceProperties)
		tryRegisterProc(dll, "vkGetPhysicalDeviceQueueFamilyProperties", &api.getPhysicalDeviceQueueFamilyProperties)
		tryRegisterProc(dll, "vkEnumerateDeviceExtensionProperties", &api.enumerateDeviceExtensionProperties)
		tryRegisterProc(dll, "vkGetPhysicalDeviceSurfaceSupportKHR", &api.getPhysicalDeviceSurfaceSupportKHR)
		tryRegisterProc(dll, "vkGetPhysicalDeviceSurfaceCapabilitiesKHR", &api.getPhysicalDeviceSurfaceCapabilitiesKHR)
		tryRegisterProc(dll, "vkGetPhysicalDeviceSurfaceFormatsKHR", &api.getPhysicalDeviceSurfaceFormatsKHR)
		tryRegisterProc(dll, "vkGetPhysicalDeviceSurfacePresentModesKHR", &api.getPhysicalDeviceSurfacePresentModesKHR)
		tryRegisterProc(dll, "vkCreateDevice", &api.createDevice)
		tryRegisterProc(dll, "vkDestroyDevice", &api.destroyDevice)
		tryRegisterProc(dll, "vkGetDeviceQueue", &api.getDeviceQueue)
		tryRegisterProc(dll, "vkDeviceWaitIdle", &api.deviceWaitIdle)
		tryRegisterProc(dll, "vkCreateWin32SurfaceKHR", &api.createWin32SurfaceKHR)
		tryRegisterProc(dll, "vkDestroySurfaceKHR", &api.destroySurfaceKHR)
		tryRegisterProc(dll, "vkCreateSwapchainKHR", &api.createSwapchainKHR)
		tryRegisterProc(dll, "vkDestroySwapchainKHR", &api.destroySwapchainKHR)
		defaultAPI = api
	})
	return defaultAPI, defaultAPIErr
}

func registerProc(dll *windows.LazyDLL, name string, target any) error {
	proc := dll.NewProc(name)
	if err := proc.Find(); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrUnavailable, name, err)
	}
	purego.RegisterFunc(target, proc.Addr())
	return nil
}

func tryRegisterProc(dll *windows.LazyDLL, name string, target any) {
	proc := dll.NewProc(name)
	if err := proc.Find(); err != nil {
		return
	}
	purego.RegisterFunc(target, proc.Addr())
}
