package backend

import (
	"image/png"
	"os"
	"path/filepath"
	"sync"

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
	virtualWidth  int
	virtualHeight int
	pixelPerfect  bool
	textures      textureCatalog
}

type textureCatalog struct {
	mu     sync.Mutex
	nextID render.TextureID
	byPath map[string]textureMeta
	byID   map[render.TextureID]textureMeta
}

type textureMeta struct {
	id     render.TextureID
	width  int
	height int
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

	return &vulkanDesktopBackend{
		window:    window,
		instance:  instance,
		surface:   surface,
		device:    device,
		swapchain: swapchain,
		textures: textureCatalog{
			byPath: make(map[string]textureMeta),
			byID:   make(map[render.TextureID]textureMeta),
		},
	}, nil
}

func (b *vulkanDesktopBackend) UpdateInput(state *input.State) {
	if b == nil || b.window == nil {
		return
	}
	b.window.UpdateInput(state)
}

func (b *vulkanDesktopBackend) PumpEvents() bool {
	if b == nil || b.window == nil {
		return true
	}
	return b.window.PumpEvents()
}

func (b *vulkanDesktopBackend) LoadTexture(path string) (render.TextureID, int, int, error) {
	return b.textures.load(path)
}

func (b *vulkanDesktopBackend) SetLogicalSize(w, h int, pixelPerfect bool) error {
	b.virtualWidth = w
	b.virtualHeight = h
	b.pixelPerfect = pixelPerfect
	return nil
}

func (b *vulkanDesktopBackend) DrawFrame(clearColor geom.Color, cmds []render.DrawCmd, lines []string) error {
	_ = clearColor
	_ = cmds
	_ = lines

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
	return nil
}

func (b *vulkanDesktopBackend) Destroy() error {
	var err error
	if b == nil {
		return nil
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

func (c *textureCatalog) load(path string) (render.TextureID, int, int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return 0, 0, 0, err
	}
	if meta, ok := c.byPath[cleanPath]; ok {
		return meta.id, meta.width, meta.height, nil
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return 0, 0, 0, err
	}
	defer file.Close()

	cfg, err := png.DecodeConfig(file)
	if err != nil {
		return 0, 0, 0, err
	}

	c.nextID++
	meta := textureMeta{
		id:     c.nextID,
		width:  cfg.Width,
		height: cfg.Height,
	}
	c.byPath[cleanPath] = meta
	c.byID[meta.id] = meta
	return meta.id, meta.width, meta.height, nil
}

func vulkanValidationEnabled() bool {
	return os.Getenv("WHISKY_VULKAN_VALIDATION") == "1"
}
