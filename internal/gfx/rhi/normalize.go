package rhi

import (
	"errors"
	"fmt"

	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

var (
	ErrInvalidSurfaceTarget       = errors.New("rhi: invalid surface target")
	ErrInvalidSwapchainDescriptor = errors.New("rhi: invalid swapchain descriptor")
)

func NormalizeSurfaceTarget(target SurfaceTarget) (SurfaceTarget, error) {
	if err := validateNativeWindowHandle(target.Window); err != nil {
		return SurfaceTarget{}, err
	}
	if target.Extent.Width <= 0 || target.Extent.Height <= 0 {
		return SurfaceTarget{}, fmt.Errorf(
			"%w: extent must be positive, got %dx%d",
			ErrInvalidSurfaceTarget,
			target.Extent.Width,
			target.Extent.Height,
		)
	}
	return target, nil
}

func NormalizeSwapchainDescriptor(desc SwapchainDescriptor, fallback Extent2D) (SwapchainDescriptor, error) {
	if fallback.Width <= 0 || fallback.Height <= 0 {
		return SwapchainDescriptor{}, fmt.Errorf(
			"%w: fallback extent must be positive, got %dx%d",
			ErrInvalidSwapchainDescriptor,
			fallback.Width,
			fallback.Height,
		)
	}

	normalized := desc
	if normalized.Extent.Width == 0 {
		normalized.Extent.Width = fallback.Width
	}
	if normalized.Extent.Height == 0 {
		normalized.Extent.Height = fallback.Height
	}
	if normalized.Extent.Width <= 0 || normalized.Extent.Height <= 0 {
		return SwapchainDescriptor{}, fmt.Errorf(
			"%w: extent must be positive, got %dx%d",
			ErrInvalidSwapchainDescriptor,
			normalized.Extent.Width,
			normalized.Extent.Height,
		)
	}
	if normalized.Format == PixelFormatUnknown {
		normalized.Format = PixelFormatBGRA8Unorm
	}
	if normalized.PresentMode == "" {
		normalized.PresentMode = PresentModeFIFO
	}
	if normalized.BufferCount == 0 {
		normalized.BufferCount = 2
	}
	if normalized.BufferCount < 2 {
		return SwapchainDescriptor{}, fmt.Errorf(
			"%w: buffer count must be at least 2, got %d",
			ErrInvalidSwapchainDescriptor,
			normalized.BufferCount,
		)
	}

	return normalized, nil
}

func validateNativeWindowHandle(handle platformapi.NativeWindowHandle) error {
	switch handle.Kind {
	case platformapi.NativeWindowKindWin32:
		if handle.Window == 0 || handle.Instance == 0 {
			return fmt.Errorf("%w: win32 handle requires window and instance", ErrInvalidSurfaceTarget)
		}
	case platformapi.NativeWindowKindX11:
		if handle.Display == 0 || handle.Window == 0 {
			return fmt.Errorf("%w: x11 handle requires display and window", ErrInvalidSurfaceTarget)
		}
	case platformapi.NativeWindowKindWayland:
		if handle.Display == 0 || handle.Window == 0 {
			return fmt.Errorf("%w: wayland handle requires display and surface", ErrInvalidSurfaceTarget)
		}
	default:
		return fmt.Errorf("%w: unsupported native window kind %q", ErrInvalidSurfaceTarget, handle.Kind)
	}

	return nil
}
