//go:build linux

package vulkan

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
)

var (
	defaultAPIOnce sync.Once
	defaultAPI     *vulkanAPI
	defaultAPIErr  error
)

func loadDefaultAPI() (*vulkanAPI, error) {
	defaultAPIOnce.Do(func() {
		handle, err := loadUnixVulkanLibrary()
		if err != nil {
			defaultAPIErr = err
			return
		}

		api := &vulkanAPI{}
		purego.RegisterLibFunc(&api.enumerateInstanceExtensionProperties, handle, "vkEnumerateInstanceExtensionProperties")
		purego.RegisterLibFunc(&api.enumerateInstanceLayerProperties, handle, "vkEnumerateInstanceLayerProperties")
		purego.RegisterLibFunc(&api.createInstance, handle, "vkCreateInstance")
		purego.RegisterLibFunc(&api.destroyInstance, handle, "vkDestroyInstance")
		tryRegisterLibFunc(handle, "vkCreateXlibSurfaceKHR", &api.createXlibSurfaceKHR)
		tryRegisterLibFunc(handle, "vkCreateWaylandSurfaceKHR", &api.createWaylandSurfaceKHR)
		tryRegisterLibFunc(handle, "vkDestroySurfaceKHR", &api.destroySurfaceKHR)
		defaultAPI = api
	})
	return defaultAPI, defaultAPIErr
}

func tryRegisterLibFunc(handle uintptr, name string, target any) {
	addr, err := purego.Dlsym(handle, name)
	if err != nil {
		return
	}
	purego.RegisterFunc(target, addr)
}

func loadUnixVulkanLibrary() (uintptr, error) {
	candidates := []string{"libvulkan.so.1", "libvulkan.so"}
	for _, name := range candidates {
		handle, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err == nil {
			return handle, nil
		}
	}
	return 0, fmt.Errorf("%w: unable to load libvulkan.so", ErrUnavailable)
}
