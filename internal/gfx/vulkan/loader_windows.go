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
