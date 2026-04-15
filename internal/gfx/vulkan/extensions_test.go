package vulkan

import (
	"reflect"
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

func TestRequiredInstanceExtensionsForX11(t *testing.T) {
	extensions, err := RequiredInstanceExtensions(rhi.SurfaceTarget{
		Window: platformapi.NativeWindowHandle{
			Kind:    platformapi.NativeWindowKindX11,
			Display: 0x1000,
			Window:  0x2000,
		},
		Extent: rhi.Extent2D{Width: 1280, Height: 720},
	}, Options{})
	if err != nil {
		t.Fatalf("expected valid extension list, got error: %v", err)
	}

	want := []string{extSurface, extXlibSurface}
	if !reflect.DeepEqual(extensions, want) {
		t.Fatalf("expected %v, got %v", want, extensions)
	}
}

func TestRequiredInstanceExtensionsWithValidation(t *testing.T) {
	extensions, err := RequiredInstanceExtensions(rhi.SurfaceTarget{
		Window: platformapi.NativeWindowHandle{
			Kind:     platformapi.NativeWindowKindWin32,
			Window:   0x1234,
			Instance: 0x5678,
		},
		Extent: rhi.Extent2D{Width: 800, Height: 600},
	}, Options{EnableValidation: true})
	if err != nil {
		t.Fatalf("expected valid extension list, got error: %v", err)
	}

	want := []string{extSurface, extWin32Surface, extDebugUtils}
	if !reflect.DeepEqual(extensions, want) {
		t.Fatalf("expected %v, got %v", want, extensions)
	}
}

func TestValidationLayers(t *testing.T) {
	layers := ValidationLayers(Options{EnableValidation: true})
	want := []string{layerKhronosValidation}
	if !reflect.DeepEqual(layers, want) {
		t.Fatalf("expected %v, got %v", want, layers)
	}
}
