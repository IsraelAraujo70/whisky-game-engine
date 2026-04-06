package vulkan

import (
	"fmt"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

const (
	extSurface        = "VK_KHR_surface"
	extWin32Surface   = "VK_KHR_win32_surface"
	extXlibSurface    = "VK_KHR_xlib_surface"
	extWaylandSurface = "VK_KHR_wayland_surface"
	extDebugUtils     = "VK_EXT_debug_utils"

	layerKhronosValidation = "VK_LAYER_KHRONOS_validation"
)

func RequiredInstanceExtensions(target rhi.SurfaceTarget, opts Options) ([]string, error) {
	normalized, err := rhi.NormalizeSurfaceTarget(target)
	if err != nil {
		return nil, err
	}

	extensions := []string{extSurface}
	switch normalized.Window.Kind {
	case platformapi.NativeWindowKindWin32:
		extensions = append(extensions, extWin32Surface)
	case platformapi.NativeWindowKindX11:
		extensions = append(extensions, extXlibSurface)
	case platformapi.NativeWindowKindWayland:
		extensions = append(extensions, extWaylandSurface)
	default:
		return nil, fmt.Errorf("vulkan: unsupported native window kind %q", normalized.Window.Kind)
	}

	if opts.EnableValidation {
		extensions = append(extensions, extDebugUtils)
	}

	return extensions, nil
}

func ValidationLayers(opts Options) []string {
	if !opts.EnableValidation {
		return nil
	}
	return []string{layerKhronosValidation}
}
