package backend

import (
	"math"
	"os"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/input"
	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
	vkapi "github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/vulkan"
	nativewindowapi "github.com/IsraelAraujo70/whisky-game-engine/internal/nativewindow"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
)

type vulkanDesktopBackend struct {
	window        platformapi.NativeWindow
	instance      rhi.Instance
	surface       rhi.Surface
	device        rhi.Device
	swapchain     rhi.Swapchain
	renderer      *vkapi.Renderer2D
	virtualWidth  int
	virtualHeight int
	pixelPerfect  bool
}

func newVulkanDesktopBackend(title string, width, height int, keyMap map[string]string) (platformapi.Backend, error) {
	window, err := nativewindowapi.NewDesktop(title, width, height, keyMap)
	if err != nil {
		return nil, err
	}

	windowWidth, windowHeight := window.Size()
	target := rhi.SurfaceTarget{
		Window: window.NativeHandle(),
		Extent: rhi.Extent2D{Width: windowWidth, Height: windowHeight},
	}

	instance, err := vkapi.NewInstance(vkapi.Options{
		EnableValidation: vulkanValidationEnabled(),
		SurfaceTarget:    &target,
		ApplicationName:  title,
	})
	if err != nil {
		_ = window.Destroy()
		return nil, err
	}

	surface, err := instance.CreateSurface(target)
	if err != nil {
		_ = instance.Destroy()
		_ = window.Destroy()
		return nil, err
	}

	device, err := instance.CreateDevice(surface, rhi.DeviceOptions{
		PreferDiscreteGPU: true,
		EnableValidation:  vulkanValidationEnabled(),
	})
	if err != nil {
		_ = surface.Destroy()
		_ = instance.Destroy()
		_ = window.Destroy()
		return nil, err
	}

	swapchain, err := device.CreateSwapchain(surface, rhi.SwapchainDescriptor{
		Extent:      target.Extent,
		PresentMode: rhi.PresentModeFIFO,
	})
	if err != nil {
		_ = device.Destroy()
		_ = surface.Destroy()
		_ = instance.Destroy()
		_ = window.Destroy()
		return nil, err
	}

	renderer, err := vkapi.NewRenderer2D(device, swapchain)
	if err != nil {
		_ = swapchain.Destroy()
		_ = device.Destroy()
		_ = surface.Destroy()
		_ = instance.Destroy()
		_ = window.Destroy()
		return nil, err
	}

	return &vulkanDesktopBackend{
		window:    window,
		instance:  instance,
		surface:   surface,
		device:    device,
		swapchain: swapchain,
		renderer:  renderer,
	}, nil
}

func (b *vulkanDesktopBackend) UpdateInput(state *input.State) {
	if b == nil || b.window == nil {
		return
	}
	b.window.UpdateInput(state)

	// Scale mouse from window (pixel) coordinates to virtual coordinates.
	// This mirrors the viewport calculation in renderer.computeViewport so
	// that mouse positions match the coordinate space used by DrawRect/DrawText.
	if b.virtualWidth > 0 && b.virtualHeight > 0 {
		ww, wh := b.window.Size()
		if ww > 0 && wh > 0 {
			logicalW := float64(b.virtualWidth)
			logicalH := float64(b.virtualHeight)
			targetW := float64(ww)
			targetH := float64(wh)

			scaleX := targetW / logicalW
			scaleY := targetH / logicalH
			scale := math.Min(scaleX, scaleY)
			if b.pixelPerfect && scale >= 1 {
				scale = math.Floor(scale)
				if scale < 1 {
					scale = 1
				}
			}

			viewportW := logicalW * scale
			viewportH := logicalH * scale
			offsetX := (targetW - viewportW) * 0.5
			offsetY := (targetH - viewportH) * 0.5

			mouse := state.Mouse()
			mx, my := mouse.Position()
			mouse.SetPosition(
				(mx-offsetX)/scale,
				(my-offsetY)/scale,
			)
		}
	}
}

func (b *vulkanDesktopBackend) PumpEvents() bool {
	if b == nil || b.window == nil {
		return true
	}
	return b.window.PumpEvents()
}

func (b *vulkanDesktopBackend) LoadTexture(path string) (render.TextureID, int, int, error) {
	if b.renderer == nil {
		return 0, 0, 0, nil
	}
	return b.renderer.LoadTexture(path)
}

func (b *vulkanDesktopBackend) SetLogicalSize(w, h int, pixelPerfect bool) error {
	b.virtualWidth = w
	b.virtualHeight = h
	b.pixelPerfect = pixelPerfect
	if b.renderer != nil {
		b.renderer.SetLogicalSize(w, h, pixelPerfect)
	}
	return nil
}

func (b *vulkanDesktopBackend) DrawFrame(clearColor geom.Color, cmds []render.DrawCmd, lines []string) error {
	if b == nil || b.window == nil || b.swapchain == nil {
		return nil
	}

	width, height := b.window.Size()
	desc := b.swapchain.Descriptor()
	if desc.Extent.Width != width || desc.Extent.Height != height {
		if err := b.swapchain.Resize(width, height); err != nil {
			return err
		}
	}
	if b.renderer == nil {
		return nil
	}
	return b.renderer.DrawFrame(clearColor, cmds, lines)
}

func (b *vulkanDesktopBackend) Destroy() error {
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
	if b.surface != nil {
		if destroyErr := b.surface.Destroy(); err == nil {
			err = destroyErr
		}
		b.surface = nil
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

// SetWindowSize resizes the OS window if the platform supports it.
func (b *vulkanDesktopBackend) SetWindowSize(width, height int) error {
	if b == nil || b.window == nil {
		return platformapi.ErrNotSupported
	}
	dc, ok := b.window.(platformapi.DisplayController)
	if !ok {
		return platformapi.ErrNotSupported
	}
	return dc.SetWindowSize(width, height)
}

// SetWindowMode sets the window mode (windowed, borderless, fullscreen).
func (b *vulkanDesktopBackend) SetWindowMode(mode platformapi.WindowMode) error {
	if b == nil || b.window == nil {
		return platformapi.ErrNotSupported
	}
	dc, ok := b.window.(platformapi.DisplayController)
	if !ok {
		return platformapi.ErrNotSupported
	}
	return dc.SetWindowMode(mode)
}

// Monitors returns available monitors and their display modes.
func (b *vulkanDesktopBackend) Monitors() ([]platformapi.MonitorInfo, error) {
	if b == nil || b.window == nil {
		return nil, platformapi.ErrNotSupported
	}
	dc, ok := b.window.(platformapi.DisplayController)
	if !ok {
		return nil, platformapi.ErrNotSupported
	}
	return dc.Monitors()
}

// MoveToMonitor moves the window to the specified monitor.
func (b *vulkanDesktopBackend) MoveToMonitor(index int) error {
	if b == nil || b.window == nil {
		return platformapi.ErrNotSupported
	}
	dc, ok := b.window.(platformapi.DisplayController)
	if !ok {
		return platformapi.ErrNotSupported
	}
	return dc.MoveToMonitor(index)
}

func vulkanValidationEnabled() bool {
	return os.Getenv("WHISKY_VULKAN_VALIDATION") == "1"
}
