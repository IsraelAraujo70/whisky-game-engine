package rhi

import (
	"errors"
	"testing"

	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

func TestNormalizeSurfaceTargetWin32(t *testing.T) {
	target, err := NormalizeSurfaceTarget(SurfaceTarget{
		Window: platformapi.NativeWindowHandle{
			Kind:     platformapi.NativeWindowKindWin32,
			Window:   0x1234,
			Instance: 0x5678,
		},
		Extent: Extent2D{Width: 1280, Height: 720},
	})
	if err != nil {
		t.Fatalf("expected valid target, got error: %v", err)
	}
	if target.Extent.Width != 1280 || target.Extent.Height != 720 {
		t.Fatalf("unexpected extent: %+v", target.Extent)
	}
}

func TestNormalizeSurfaceTargetRejectsMissingHandleFields(t *testing.T) {
	_, err := NormalizeSurfaceTarget(SurfaceTarget{
		Window: platformapi.NativeWindowHandle{
			Kind:   platformapi.NativeWindowKindX11,
			Window: 0x1234,
		},
		Extent: Extent2D{Width: 800, Height: 600},
	})
	if !errors.Is(err, ErrInvalidSurfaceTarget) {
		t.Fatalf("expected ErrInvalidSurfaceTarget, got %v", err)
	}
}

func TestNormalizeSurfaceTargetCocoa(t *testing.T) {
	target, err := NormalizeSurfaceTarget(SurfaceTarget{
		Window: platformapi.NativeWindowHandle{
			Kind:   platformapi.NativeWindowKindCocoa,
			Window: 0x1000,
			View:   0x2000,
			Layer:  0x3000,
		},
		Extent: Extent2D{Width: 1440, Height: 900},
	})
	if err != nil {
		t.Fatalf("expected valid cocoa target, got error: %v", err)
	}
	if target.Window.Kind != platformapi.NativeWindowKindCocoa {
		t.Fatalf("expected cocoa kind, got %q", target.Window.Kind)
	}
	if target.Window.View != 0x2000 {
		t.Fatalf("expected cocoa view handle 0x2000, got %#x", target.Window.View)
	}
}

func TestNormalizeSwapchainDescriptorDefaults(t *testing.T) {
	desc, err := NormalizeSwapchainDescriptor(SwapchainDescriptor{}, Extent2D{Width: 1920, Height: 1080})
	if err != nil {
		t.Fatalf("expected valid descriptor, got error: %v", err)
	}
	if desc.Extent.Width != 1920 || desc.Extent.Height != 1080 {
		t.Fatalf("expected fallback extent, got %+v", desc.Extent)
	}
	if desc.Format != PixelFormatBGRA8Unorm {
		t.Fatalf("expected default format %q, got %q", PixelFormatBGRA8Unorm, desc.Format)
	}
	if desc.PresentMode != PresentModeFIFO {
		t.Fatalf("expected default present mode %q, got %q", PresentModeFIFO, desc.PresentMode)
	}
	if desc.BufferCount != 2 {
		t.Fatalf("expected default buffer count 2, got %d", desc.BufferCount)
	}
}

func TestNormalizeSwapchainDescriptorRejectsSingleBuffer(t *testing.T) {
	_, err := NormalizeSwapchainDescriptor(SwapchainDescriptor{
		Extent:      Extent2D{Width: 640, Height: 480},
		BufferCount: 1,
	}, Extent2D{Width: 640, Height: 480})
	if !errors.Is(err, ErrInvalidSwapchainDescriptor) {
		t.Fatalf("expected ErrInvalidSwapchainDescriptor, got %v", err)
	}
}

func TestNormalizeSwapchainDescriptorRejectsInvalidFallback(t *testing.T) {
	_, err := NormalizeSwapchainDescriptor(SwapchainDescriptor{}, Extent2D{})
	if !errors.Is(err, ErrInvalidSwapchainDescriptor) {
		t.Fatalf("expected ErrInvalidSwapchainDescriptor, got %v", err)
	}
}
