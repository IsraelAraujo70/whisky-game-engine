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

// ---------------------------------------------------------------------------
// BufferDescriptor
// ---------------------------------------------------------------------------

func TestNormalizeBufferDescriptorValid(t *testing.T) {
	desc, err := NormalizeBufferDescriptor(BufferDescriptor{
		Size:  1024,
		Usage: BufferUsageVertex,
	})
	if err != nil {
		t.Fatalf("expected valid buffer descriptor, got error: %v", err)
	}
	if desc.Size != 1024 {
		t.Fatalf("expected size 1024, got %d", desc.Size)
	}
}

func TestNormalizeBufferDescriptorRejectsZeroSize(t *testing.T) {
	_, err := NormalizeBufferDescriptor(BufferDescriptor{
		Size:  0,
		Usage: BufferUsageVertex,
	})
	if !errors.Is(err, ErrInvalidBufferDescriptor) {
		t.Fatalf("expected ErrInvalidBufferDescriptor, got %v", err)
	}
}

func TestNormalizeBufferDescriptorRejectsNoUsage(t *testing.T) {
	_, err := NormalizeBufferDescriptor(BufferDescriptor{
		Size:  256,
		Usage: 0,
	})
	if !errors.Is(err, ErrInvalidBufferDescriptor) {
		t.Fatalf("expected ErrInvalidBufferDescriptor, got %v", err)
	}
}

func TestNormalizeBufferDescriptorCombinedUsage(t *testing.T) {
	desc, err := NormalizeBufferDescriptor(BufferDescriptor{
		Size:  512,
		Usage: BufferUsageVertex | BufferUsageStaging,
	})
	if err != nil {
		t.Fatalf("expected valid descriptor, got error: %v", err)
	}
	if desc.Usage&BufferUsageVertex == 0 || desc.Usage&BufferUsageStaging == 0 {
		t.Fatalf("expected combined usage flags, got %d", desc.Usage)
	}
}

// ---------------------------------------------------------------------------
// TextureDescriptor
// ---------------------------------------------------------------------------

func TestNormalizeTextureDescriptorDefaults(t *testing.T) {
	desc, err := NormalizeTextureDescriptor(TextureDescriptor{
		Width:  128,
		Height: 64,
	})
	if err != nil {
		t.Fatalf("expected valid texture descriptor, got error: %v", err)
	}
	if desc.Format != PixelFormatRGBA8Unorm {
		t.Fatalf("expected default format %q, got %q", PixelFormatRGBA8Unorm, desc.Format)
	}
	if desc.Usage != TextureUsageSampled {
		t.Fatalf("expected default usage TextureUsageSampled, got %d", desc.Usage)
	}
}

func TestNormalizeTextureDescriptorRejectsZeroDimension(t *testing.T) {
	_, err := NormalizeTextureDescriptor(TextureDescriptor{
		Width:  0,
		Height: 256,
	})
	if !errors.Is(err, ErrInvalidTextureDescriptor) {
		t.Fatalf("expected ErrInvalidTextureDescriptor, got %v", err)
	}
}

func TestNormalizeTextureDescriptorPreservesExplicitFormat(t *testing.T) {
	desc, err := NormalizeTextureDescriptor(TextureDescriptor{
		Width:  32,
		Height: 32,
		Format: PixelFormatBGRA8SRGB,
		Usage:  TextureUsageRenderTarget,
	})
	if err != nil {
		t.Fatalf("expected valid texture descriptor, got error: %v", err)
	}
	if desc.Format != PixelFormatBGRA8SRGB {
		t.Fatalf("expected preserved format %q, got %q", PixelFormatBGRA8SRGB, desc.Format)
	}
}

// ---------------------------------------------------------------------------
// PipelineDescriptor
// ---------------------------------------------------------------------------

type mockShaderModule struct {
	stage ShaderStage
}

func (m *mockShaderModule) Backend() BackendKind { return BackendKindVulkan }
func (m *mockShaderModule) Stage() ShaderStage   { return m.stage }
func (m *mockShaderModule) Destroy() error       { return nil }

func TestNormalizePipelineDescriptorValid(t *testing.T) {
	desc := PipelineDescriptor{
		VertexShader:   &mockShaderModule{stage: ShaderStageVertex},
		FragmentShader: &mockShaderModule{stage: ShaderStageFragment},
		VertexLayout: VertexLayout{
			Stride: 32,
			Attributes: []VertexAttribute{
				{Location: 0, Format: VertexFormatFloat32x2, Offset: 0},
				{Location: 1, Format: VertexFormatFloat32x2, Offset: 8},
			},
		},
	}
	_, err := NormalizePipelineDescriptor(desc)
	if err != nil {
		t.Fatalf("expected valid pipeline descriptor, got error: %v", err)
	}
}

func TestNormalizePipelineDescriptorRejectsNilVertexShader(t *testing.T) {
	desc := PipelineDescriptor{
		FragmentShader: &mockShaderModule{stage: ShaderStageFragment},
		VertexLayout: VertexLayout{
			Stride: 32,
			Attributes: []VertexAttribute{
				{Location: 0, Format: VertexFormatFloat32x2, Offset: 0},
			},
		},
	}
	_, err := NormalizePipelineDescriptor(desc)
	if !errors.Is(err, ErrInvalidPipelineDescriptor) {
		t.Fatalf("expected ErrInvalidPipelineDescriptor, got %v", err)
	}
}

func TestNormalizePipelineDescriptorRejectsNilFragmentShader(t *testing.T) {
	desc := PipelineDescriptor{
		VertexShader: &mockShaderModule{stage: ShaderStageVertex},
		VertexLayout: VertexLayout{
			Stride: 32,
			Attributes: []VertexAttribute{
				{Location: 0, Format: VertexFormatFloat32x2, Offset: 0},
			},
		},
	}
	_, err := NormalizePipelineDescriptor(desc)
	if !errors.Is(err, ErrInvalidPipelineDescriptor) {
		t.Fatalf("expected ErrInvalidPipelineDescriptor, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// VertexLayout
// ---------------------------------------------------------------------------

func TestValidateVertexLayoutValid(t *testing.T) {
	err := ValidateVertexLayout(VertexLayout{
		Stride: 32,
		Attributes: []VertexAttribute{
			{Location: 0, Format: VertexFormatFloat32x2, Offset: 0},
			{Location: 1, Format: VertexFormatFloat32x4, Offset: 8},
		},
	})
	if err != nil {
		t.Fatalf("expected valid vertex layout, got error: %v", err)
	}
}

func TestValidateVertexLayoutRejectsZeroStride(t *testing.T) {
	err := ValidateVertexLayout(VertexLayout{
		Stride: 0,
		Attributes: []VertexAttribute{
			{Location: 0, Format: VertexFormatFloat32x2, Offset: 0},
		},
	})
	if !errors.Is(err, ErrInvalidVertexLayout) {
		t.Fatalf("expected ErrInvalidVertexLayout, got %v", err)
	}
}

func TestValidateVertexLayoutRejectsNoAttributes(t *testing.T) {
	err := ValidateVertexLayout(VertexLayout{
		Stride:     32,
		Attributes: nil,
	})
	if !errors.Is(err, ErrInvalidVertexLayout) {
		t.Fatalf("expected ErrInvalidVertexLayout, got %v", err)
	}
}

func TestValidateVertexLayoutRejectsOffsetBeyondStride(t *testing.T) {
	err := ValidateVertexLayout(VertexLayout{
		Stride: 16,
		Attributes: []VertexAttribute{
			{Location: 0, Format: VertexFormatFloat32x2, Offset: 20},
		},
	})
	if !errors.Is(err, ErrInvalidVertexLayout) {
		t.Fatalf("expected ErrInvalidVertexLayout, got %v", err)
	}
}
