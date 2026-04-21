//go:build darwin

package backend

import (
	"fmt"

	"github.com/ebitengine/purego/objc"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/input"
	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
	metalapi "github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/metal"
	nativewindowapi "github.com/IsraelAraujo70/whisky-game-engine/internal/nativewindow"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
)

type desktopBackend = platformapi.Backend

type metalLayerWindow interface {
	platformapi.NativeWindow
	AttachLayer(layer objc.ID)
}

type metalDesktopBackend struct {
	window     platformapi.NativeWindow
	instance   rhi.Instance
	device     rhi.Device
	swapchain  rhi.Swapchain
	renderer   *metalapi.Renderer2D
	virtualWidth  int
	virtualHeight int
	pixelPerfect  bool
}

var metalDesktopBackendFactory = func(title string, width, height int, keyMap map[string]string) (desktopBackend, error) {
	return newMetalDesktopBackend(title, width, height, keyMap)
}

func newMetalDesktopBackend(title string, width, height int, keyMap map[string]string) (platformapi.Backend, error) {
	windowValue, err := nativewindowapi.NewDesktop(title, width, height, keyMap)
	if err != nil {
		return nil, err
	}
	window, ok := windowValue.(metalLayerWindow)
	if !ok {
		_ = windowValue.Destroy()
		return nil, fmt.Errorf("backend: native window does not expose AttachLayer")
	}

	instance, err := metalapi.NewInstance(metalapi.Options{ApplicationName: title})
	if err != nil {
		_ = window.Destroy()
		return nil, err
	}

	surface, err := instance.CreateSurface(rhi.SurfaceTarget{
		Window: window.NativeHandle(),
		Extent: rhi.Extent2D{Width: width, Height: height},
	})
	if err != nil {
		_ = instance.Destroy()
		_ = window.Destroy()
		return nil, err
	}

	device, err := instance.CreateDevice(surface, rhi.DeviceOptions{})
	if err != nil {
		_ = surface.Destroy()
		_ = instance.Destroy()
		_ = window.Destroy()
		return nil, err
	}

	swapchain, err := device.CreateSwapchain(surface, rhi.SwapchainDescriptor{
		Extent:      rhi.Extent2D{Width: width, Height: height},
		Format:      rhi.PixelFormatBGRA8Unorm,
		PresentMode: rhi.PresentModeFIFO,
		BufferCount: 3,
	})
	if err != nil {
		_ = device.Destroy()
		_ = surface.Destroy()
		_ = instance.Destroy()
		_ = window.Destroy()
		return nil, err
	}

	renderer, err := metalapi.NewRenderer2D(device, swapchain)
	if err != nil {
		_ = swapchain.Destroy()
		_ = device.Destroy()
		_ = surface.Destroy()
		_ = instance.Destroy()
		_ = window.Destroy()
		return nil, err
	}

	return &metalDesktopBackend{
		window:    window,
		instance:  instance,
		device:    device,
		swapchain: swapchain,
		renderer:  renderer,
	}, nil
}

func (b *metalDesktopBackend) UpdateInput(state *input.State) {
	if b == nil || b.window == nil {
		return
	}
	b.window.UpdateInput(state)
}

func (b *metalDesktopBackend) PumpEvents() bool {
	if b == nil || b.window == nil {
		return true
	}
	return b.window.PumpEvents()
}

func (b *metalDesktopBackend) LoadTexture(path string) (render.TextureID, int, int, error) {
	if b == nil || b.renderer == nil {
		return 0, 0, 0, nil
	}
	return b.renderer.LoadTexture(path)
}

func (b *metalDesktopBackend) SetLogicalSize(w, h int, pixelPerfect bool) error {
	b.virtualWidth = w
	b.virtualHeight = h
	b.pixelPerfect = pixelPerfect
	if b == nil || b.renderer == nil {
		return nil
	}
	return b.renderer.SetLogicalSize(w, h, pixelPerfect)
}

func (b *metalDesktopBackend) DrawFrame(clearColor geom.Color, cmds []render.DrawCmd, lines []string) error {
	if b == nil || b.renderer == nil {
		return nil
	}
	width, height := b.window.Size()
	desc := b.swapchain.Descriptor()
	if desc.Extent.Width != width || desc.Extent.Height != height {
		if err := b.swapchain.Resize(width, height); err != nil {
			return err
		}
	}
	return b.renderer.DrawFrame(clearColor, cmds, lines)
}

func (b *metalDesktopBackend) Destroy() error {
	var err error
	if b == nil {
		return nil
	}
	if b.renderer != nil {
		if destroyErr := b.renderer.Destroy(); err == nil {
			err = destroyErr
		}
		b.renderer = nil
	}
	if b.swapchain != nil {
		if destroyErr := b.swapchain.Destroy(); err == nil {
			err = destroyErr
		}
		b.swapchain = nil
	}
	if b.device != nil {
		if destroyErr := b.device.Destroy(); err == nil {
			err = destroyErr
		}
		b.device = nil
	}
	if b.instance != nil {
		if destroyErr := b.instance.Destroy(); err == nil {
			err = destroyErr
		}
		b.instance = nil
	}
	if b.window != nil {
		if destroyErr := b.window.Destroy(); err == nil {
			err = destroyErr
		}
		b.window = nil
	}
	return err
}
