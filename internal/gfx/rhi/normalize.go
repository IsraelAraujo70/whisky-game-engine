package rhi

import (
	"errors"
	"fmt"

	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

var (
	ErrInvalidSurfaceTarget       = errors.New("rhi: invalid surface target")
	ErrInvalidSwapchainDescriptor = errors.New("rhi: invalid swapchain descriptor")
	ErrInvalidBufferDescriptor    = errors.New("rhi: invalid buffer descriptor")
	ErrInvalidTextureDescriptor   = errors.New("rhi: invalid texture descriptor")
	ErrInvalidPipelineDescriptor  = errors.New("rhi: invalid pipeline descriptor")
	ErrInvalidVertexLayout        = errors.New("rhi: invalid vertex layout")
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

// NormalizeBufferDescriptor validates a BufferDescriptor and fills defaults.
func NormalizeBufferDescriptor(desc BufferDescriptor) (BufferDescriptor, error) {
	if desc.Size <= 0 {
		return BufferDescriptor{}, fmt.Errorf(
			"%w: size must be positive, got %d",
			ErrInvalidBufferDescriptor,
			desc.Size,
		)
	}
	if desc.Usage == 0 {
		return BufferDescriptor{}, fmt.Errorf(
			"%w: at least one usage flag must be set",
			ErrInvalidBufferDescriptor,
		)
	}
	return desc, nil
}

// NormalizeTextureDescriptor validates a TextureDescriptor and fills defaults.
func NormalizeTextureDescriptor(desc TextureDescriptor) (TextureDescriptor, error) {
	if desc.Width <= 0 || desc.Height <= 0 {
		return TextureDescriptor{}, fmt.Errorf(
			"%w: dimensions must be positive, got %dx%d",
			ErrInvalidTextureDescriptor,
			desc.Width,
			desc.Height,
		)
	}
	normalized := desc
	if normalized.Format == PixelFormatUnknown {
		normalized.Format = PixelFormatRGBA8Unorm
	}
	if normalized.Usage == 0 {
		normalized.Usage = TextureUsageSampled
	}
	return normalized, nil
}

// NormalizePipelineDescriptor validates a PipelineDescriptor and fills defaults.
func NormalizePipelineDescriptor(desc PipelineDescriptor) (PipelineDescriptor, error) {
	if desc.VertexShader == nil {
		return PipelineDescriptor{}, fmt.Errorf(
			"%w: vertex shader is required",
			ErrInvalidPipelineDescriptor,
		)
	}
	if desc.FragmentShader == nil {
		return PipelineDescriptor{}, fmt.Errorf(
			"%w: fragment shader is required",
			ErrInvalidPipelineDescriptor,
		)
	}
	if err := ValidateVertexLayout(desc.VertexLayout); err != nil {
		return PipelineDescriptor{}, err
	}
	return desc, nil
}

// ValidateVertexLayout checks that a VertexLayout is well-formed.
func ValidateVertexLayout(layout VertexLayout) error {
	if layout.Stride == 0 {
		return fmt.Errorf("%w: stride must be positive", ErrInvalidVertexLayout)
	}
	if len(layout.Attributes) == 0 {
		return fmt.Errorf("%w: at least one attribute is required", ErrInvalidVertexLayout)
	}
	for i, attr := range layout.Attributes {
		if attr.Offset >= layout.Stride {
			return fmt.Errorf(
				"%w: attribute %d offset %d exceeds stride %d",
				ErrInvalidVertexLayout,
				i, attr.Offset, layout.Stride,
			)
		}
	}
	return nil
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
	case platformapi.NativeWindowKindCocoa:
		if handle.Window == 0 || handle.View == 0 {
			return fmt.Errorf("%w: cocoa handle requires window and view", ErrInvalidSurfaceTarget)
		}
	default:
		return fmt.Errorf("%w: unsupported native window kind %q", ErrInvalidSurfaceTarget, handle.Kind)
	}

	return nil
}
