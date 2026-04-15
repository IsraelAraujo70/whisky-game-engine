package vulkan

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

const (
	vkStructureTypeXlibSurfaceCreateInfoKHR    = 1000004000
	vkStructureTypeWaylandSurfaceCreateInfoKHR = 1000006000
	vkStructureTypeWin32SurfaceCreateInfoKHR   = 1000009000
)

type vkSurfaceKHR uintptr

type vkXlibSurfaceCreateInfoKHR struct {
	SType  int32
	_      [4]byte
	PNext  unsafe.Pointer
	Flags  uint32
	_      [4]byte
	Dpy    unsafe.Pointer
	Window uintptr
}

type vkWaylandSurfaceCreateInfoKHR struct {
	SType   int32
	_       [4]byte
	PNext   unsafe.Pointer
	Flags   uint32
	_       [4]byte
	Display unsafe.Pointer
	Surface unsafe.Pointer
}

type vkWin32SurfaceCreateInfoKHR struct {
	SType     int32
	_         [4]byte
	PNext     unsafe.Pointer
	Flags     uint32
	_         [4]byte
	Hinstance unsafe.Pointer
	Hwnd      unsafe.Pointer
}

type surface struct {
	api      *vulkanAPI
	instance vkInstance
	handle   vkSurfaceKHR
	target   rhi.SurfaceTarget
}

func (i *instance) CreateSurface(target rhi.SurfaceTarget) (rhi.Surface, error) {
	normalized, err := rhi.NormalizeSurfaceTarget(target)
	if err != nil {
		return nil, err
	}
	if err := i.requireSurfaceExtensions(normalized.Window.Kind); err != nil {
		return nil, err
	}

	handle, err := i.createPlatformSurface(normalized)
	if err != nil {
		return nil, err
	}

	surf := &surface{
		api:      i.api,
		instance: i.handle,
		handle:   handle,
		target:   normalized,
	}
	runtime.SetFinalizer(surf, func(s *surface) {
		_ = s.Destroy()
	})
	return surf, nil
}

func (s *surface) Backend() rhi.BackendKind {
	return rhi.BackendKindVulkan
}

func (s *surface) Target() rhi.SurfaceTarget {
	return s.target
}

func (s *surface) Destroy() error {
	if s == nil || s.handle == 0 {
		return nil
	}
	if s.api.destroySurfaceKHR != nil {
		s.api.destroySurfaceKHR(s.instance, s.handle, nil)
	}
	s.handle = 0
	runtime.SetFinalizer(s, nil)
	return nil
}

func (i *instance) requireSurfaceExtensions(kind platformapi.NativeWindowKind) error {
	if !contains(i.enabledExtensions, extSurface) {
		return fmt.Errorf("%w: %s", ErrMissingExtension, extSurface)
	}

	switch kind {
	case platformapi.NativeWindowKindWin32:
		if !contains(i.enabledExtensions, extWin32Surface) {
			return fmt.Errorf("%w: %s", ErrMissingExtension, extWin32Surface)
		}
	case platformapi.NativeWindowKindX11:
		if !contains(i.enabledExtensions, extXlibSurface) {
			return fmt.Errorf("%w: %s", ErrMissingExtension, extXlibSurface)
		}
	case platformapi.NativeWindowKindWayland:
		if !contains(i.enabledExtensions, extWaylandSurface) {
			return fmt.Errorf("%w: %s", ErrMissingExtension, extWaylandSurface)
		}
	default:
		return fmt.Errorf("%w: unsupported native window kind %q", ErrSurfaceUnsupported, kind)
	}

	return nil
}

func (i *instance) createPlatformSurface(target rhi.SurfaceTarget) (vkSurfaceKHR, error) {
	switch target.Window.Kind {
	case platformapi.NativeWindowKindWin32:
		return i.createWin32Surface(target)
	case platformapi.NativeWindowKindX11:
		return i.createXlibSurface(target)
	case platformapi.NativeWindowKindWayland:
		return i.createWaylandSurface(target)
	default:
		return 0, fmt.Errorf("%w: unsupported native window kind %q", ErrSurfaceUnsupported, target.Window.Kind)
	}
}

func (i *instance) createWin32Surface(target rhi.SurfaceTarget) (vkSurfaceKHR, error) {
	if i.api.createWin32SurfaceKHR == nil {
		return 0, ErrSurfaceUnsupported
	}

	createInfo := vkWin32SurfaceCreateInfoKHR{
		SType:     vkStructureTypeWin32SurfaceCreateInfoKHR,
		Hinstance: unsafe.Pointer(target.Window.Instance),
		Hwnd:      unsafe.Pointer(target.Window.Window),
	}
	return i.createSurfaceHandle(func(surface *vkSurfaceKHR) vkResult {
		return i.api.createWin32SurfaceKHR(i.handle, &createInfo, nil, surface)
	})
}

func (i *instance) createXlibSurface(target rhi.SurfaceTarget) (vkSurfaceKHR, error) {
	if i.api.createXlibSurfaceKHR == nil {
		return 0, ErrSurfaceUnsupported
	}

	createInfo := vkXlibSurfaceCreateInfoKHR{
		SType:  vkStructureTypeXlibSurfaceCreateInfoKHR,
		Dpy:    unsafe.Pointer(target.Window.Display),
		Window: target.Window.Window,
	}
	return i.createSurfaceHandle(func(surface *vkSurfaceKHR) vkResult {
		return i.api.createXlibSurfaceKHR(i.handle, &createInfo, nil, surface)
	})
}

func (i *instance) createWaylandSurface(target rhi.SurfaceTarget) (vkSurfaceKHR, error) {
	if i.api.createWaylandSurfaceKHR == nil {
		return 0, ErrSurfaceUnsupported
	}

	createInfo := vkWaylandSurfaceCreateInfoKHR{
		SType:   vkStructureTypeWaylandSurfaceCreateInfoKHR,
		Display: unsafe.Pointer(target.Window.Display),
		Surface: unsafe.Pointer(target.Window.Window),
	}
	return i.createSurfaceHandle(func(surface *vkSurfaceKHR) vkResult {
		return i.api.createWaylandSurfaceKHR(i.handle, &createInfo, nil, surface)
	})
}

func (i *instance) createSurfaceHandle(create func(surface *vkSurfaceKHR) vkResult) (vkSurfaceKHR, error) {
	var handle vkSurfaceKHR
	if result := create(&handle); result != vkSuccess {
		return 0, fmt.Errorf("%w: %s", ErrCreateSurface, result)
	}
	return handle, nil
}
