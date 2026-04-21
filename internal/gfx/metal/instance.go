//go:build darwin

package metal

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/gfx/rhi"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

// Options configure Metal instance creation.
type Options struct {
	ApplicationName  string
	EnableValidation bool
}

var (
	instanceOnce sync.Once
	instanceErr  error
	mtlCreateSystemDefaultDevice func() objc.ID
)

func loadMetalRuntime() error {
	instanceOnce.Do(func() {
		if _, err := purego.Dlopen("/System/Library/Frameworks/QuartzCore.framework/QuartzCore", purego.RTLD_GLOBAL|purego.RTLD_LAZY); err != nil {
			instanceErr = fmt.Errorf("metal: load QuartzCore: %w", err)
			return
		}
		metalHandle, err := purego.Dlopen("/System/Library/Frameworks/Metal.framework/Metal", purego.RTLD_GLOBAL|purego.RTLD_LAZY)
		if err != nil {
			instanceErr = fmt.Errorf("metal: load Metal.framework: %w", err)
			return
		}
		purego.RegisterLibFunc(&mtlCreateSystemDefaultDevice, metalHandle, "MTLCreateSystemDefaultDevice")
		if mtlCreateSystemDefaultDevice == nil {
			instanceErr = fmt.Errorf("metal: MTLCreateSystemDefaultDevice not found")
		}
	})
	return instanceErr
}

// NewInstance creates a Metal RHI instance backed by the system default MTLDevice.
func NewInstance(opts Options) (rhi.Instance, error) {
	if err := loadMetalRuntime(); err != nil {
		return nil, err
	}
	device := mtlCreateSystemDefaultDevice()
	if device == 0 {
		return nil, fmt.Errorf("metal: MTLCreateSystemDefaultDevice returned nil")
	}
	return &metalInstance{device: device}, nil
}

type metalInstance struct {
	device objc.ID
}

func (i *metalInstance) Backend() rhi.BackendKind { return rhi.BackendKindMetal }

func (i *metalInstance) CreateSurface(target rhi.SurfaceTarget) (rhi.Surface, error) {
	if target.Window.Kind != platformapi.NativeWindowKindCocoa {
		return nil, fmt.Errorf("metal: unsupported window kind %q", target.Window.Kind)
	}

	layerClass := objc.GetClass("CAMetalLayer")
	if layerClass == 0 {
		return nil, fmt.Errorf("metal: CAMetalLayer class not available")
	}

	selLayer := objc.RegisterName("layer")
	selRetain := objc.RegisterName("retain")
	selSetDevice := objc.RegisterName("setDevice:")
	selSetPixelFormat := objc.RegisterName("setPixelFormat:")
	selSetFramebufferOnly := objc.RegisterName("setFramebufferOnly:")
	selSetOpaque := objc.RegisterName("setOpaque:")
	selSetWantsLayer := objc.RegisterName("setWantsLayer:")
	selSetLayer := objc.RegisterName("setLayer:")
	selSetNeedsDisplay := objc.RegisterName("setNeedsDisplay:")

	layer := objc.ID(layerClass).Send(selLayer)
	if layer == 0 {
		return nil, fmt.Errorf("metal: failed to create CAMetalLayer")
	}
	layer = layer.Send(selRetain)
	layer.Send(selSetDevice, i.device)
	layer.Send(selSetPixelFormat, uintptr(mtlPixelFormatBGRA8Unorm))
	layer.Send(selSetFramebufferOnly, true)
	layer.Send(selSetOpaque, true)

	view := objc.ID(target.Window.View)
	if view != 0 {
		view.Send(selSetWantsLayer, true)
		view.Send(selSetLayer, layer)
		view.Send(selSetNeedsDisplay, true)
	}

	return &metalSurface{
		target: target,
		layer:  layer,
	}, nil
}

func (i *metalInstance) CreateDevice(surface rhi.Surface, opts rhi.DeviceOptions) (rhi.Device, error) {
	// Metal has one device per system; ignore surface and opts.
	_ = surface
	_ = opts

	selNewCommandQueue := objc.RegisterName("newCommandQueue")
	commandQueue := objc.Send[objc.ID](i.device, selNewCommandQueue)
	if commandQueue == 0 {
		return nil, fmt.Errorf("metal: failed to create command queue")
	}

	return &metalDevice{
		device:       i.device,
		commandQueue: commandQueue,
	}, nil
}

func (i *metalInstance) Destroy() error {
	return nil
}

type metalSurface struct {
	target rhi.SurfaceTarget
	layer  objc.ID
}

func (s *metalSurface) Backend() rhi.BackendKind { return rhi.BackendKindMetal }
func (s *metalSurface) Target() rhi.SurfaceTarget { return s.target }
func (s *metalSurface) Destroy() error {
	if s.layer != 0 {
		selRelease := objc.RegisterName("release")
		s.layer.Send(selRelease)
		s.layer = 0
	}
	return nil
}
