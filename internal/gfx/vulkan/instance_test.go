package vulkan

import (
	"errors"
	"reflect"
	"testing"
	"unsafe"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

func TestNewInstanceWithAPIEnablesExtensionsAndLayers(t *testing.T) {
	var captured struct {
		extensions []string
		layers     []string
	}

	api := &vulkanAPI{
		enumerateInstanceExtensionProperties: fakeEnumerateExtensions(
			extSurface, extXlibSurface, extDebugUtils,
		),
		enumerateInstanceLayerProperties: fakeEnumerateLayers(layerKhronosValidation),
		createInstance: func(createInfo *vkInstanceCreateInfo, allocator unsafe.Pointer, instance *vkInstance) vkResult {
			captured.extensions = decodeCStringArray(createInfo.PpEnabledExtensionNames, createInfo.EnabledExtensionCount)
			captured.layers = decodeCStringArray(createInfo.PpEnabledLayerNames, createInfo.EnabledLayerCount)
			*instance = vkInstance(0xCAFE)
			return vkSuccess
		},
		destroyInstance: func(instance vkInstance, allocator unsafe.Pointer) {},
	}

	inst, err := newInstanceWithAPI(api, Options{
		EnableValidation: true,
		SurfaceTarget: &rhi.SurfaceTarget{
			Window: platformapi.NativeWindowHandle{
				Kind:    platformapi.NativeWindowKindX11,
				Display: 0x1000,
				Window:  0x2000,
			},
			Extent: rhi.Extent2D{Width: 1280, Height: 720},
		},
	})
	if err != nil {
		t.Fatalf("expected instance creation to succeed, got %v", err)
	}
	defer inst.Destroy()

	if inst.Backend() != rhi.BackendKindVulkan {
		t.Fatalf("expected backend %q, got %q", rhi.BackendKindVulkan, inst.Backend())
	}

	wantExtensions := []string{extSurface, extXlibSurface, extDebugUtils}
	if !reflect.DeepEqual(captured.extensions, wantExtensions) {
		t.Fatalf("expected extensions %v, got %v", wantExtensions, captured.extensions)
	}

	wantLayers := []string{layerKhronosValidation}
	if !reflect.DeepEqual(captured.layers, wantLayers) {
		t.Fatalf("expected layers %v, got %v", wantLayers, captured.layers)
	}
}

func TestNewInstanceWithAPIMissingExtension(t *testing.T) {
	api := &vulkanAPI{
		enumerateInstanceExtensionProperties: fakeEnumerateExtensions(extSurface),
		enumerateInstanceLayerProperties:     fakeEnumerateLayers(),
		createInstance: func(createInfo *vkInstanceCreateInfo, allocator unsafe.Pointer, instance *vkInstance) vkResult {
			t.Fatal("createInstance should not be called when required extension is missing")
			return vkSuccess
		},
		destroyInstance: func(instance vkInstance, allocator unsafe.Pointer) {},
	}

	_, err := newInstanceWithAPI(api, Options{
		SurfaceTarget: &rhi.SurfaceTarget{
			Window: platformapi.NativeWindowHandle{
				Kind:     platformapi.NativeWindowKindWin32,
				Window:   0x1234,
				Instance: 0x5678,
			},
			Extent: rhi.Extent2D{Width: 800, Height: 600},
		},
	})
	if !errors.Is(err, ErrMissingExtension) {
		t.Fatalf("expected ErrMissingExtension, got %v", err)
	}
}

func TestNewInstanceWithAPIMissingValidationLayer(t *testing.T) {
	api := &vulkanAPI{
		enumerateInstanceExtensionProperties: fakeEnumerateExtensions(extDebugUtils),
		enumerateInstanceLayerProperties:     fakeEnumerateLayers(),
		createInstance: func(createInfo *vkInstanceCreateInfo, allocator unsafe.Pointer, instance *vkInstance) vkResult {
			t.Fatal("createInstance should not be called when validation layer is missing")
			return vkSuccess
		},
		destroyInstance: func(instance vkInstance, allocator unsafe.Pointer) {},
	}

	_, err := newInstanceWithAPI(api, Options{EnableValidation: true})
	if !errors.Is(err, ErrMissingLayer) {
		t.Fatalf("expected ErrMissingLayer, got %v", err)
	}
}

func TestInstanceDestroyCallsAPI(t *testing.T) {
	destroyed := false
	inst := &instance{
		api: &vulkanAPI{
			destroyInstance: func(instance vkInstance, allocator unsafe.Pointer) {
				destroyed = true
			},
		},
		handle: 0xDEAD,
	}

	if err := inst.Destroy(); err != nil {
		t.Fatalf("expected destroy to succeed, got %v", err)
	}
	if !destroyed {
		t.Fatal("expected destroyInstance to be called")
	}
	if inst.handle != 0 {
		t.Fatalf("expected instance handle to be reset, got %#x", inst.handle)
	}
}

func fakeEnumerateExtensions(names ...string) func(*byte, *uint32, *vkExtensionProperties) vkResult {
	return func(layerName *byte, propertyCount *uint32, properties *vkExtensionProperties) vkResult {
		if properties == nil {
			*propertyCount = uint32(len(names))
			return vkSuccess
		}
		slice := unsafe.Slice(properties, *propertyCount)
		for i, name := range names {
			copy(slice[i].ExtensionName[:], []byte(name))
		}
		return vkSuccess
	}
}

func fakeEnumerateLayers(names ...string) func(*uint32, *vkLayerProperties) vkResult {
	return func(propertyCount *uint32, properties *vkLayerProperties) vkResult {
		if properties == nil {
			*propertyCount = uint32(len(names))
			return vkSuccess
		}
		slice := unsafe.Slice(properties, *propertyCount)
		for i, name := range names {
			copy(slice[i].LayerName[:], []byte(name))
		}
		return vkSuccess
	}
}

func decodeCStringArray(values **byte, count uint32) []string {
	if values == nil || count == 0 {
		return nil
	}
	ptrs := unsafe.Slice(values, count)
	result := make([]string, 0, count)
	for _, ptr := range ptrs {
		result = append(result, decodeCString(ptr))
	}
	return result
}

func decodeCString(ptr *byte) string {
	if ptr == nil {
		return ""
	}
	var raw []byte
	for p := uintptr(unsafe.Pointer(ptr)); ; p++ {
		b := *(*byte)(unsafe.Pointer(p))
		if b == 0 {
			return string(raw)
		}
		raw = append(raw, b)
	}
}
