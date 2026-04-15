//go:build darwin

package backend

import (
	"fmt"

	"github.com/ebitengine/purego/objc"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/input"
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
	window   metalLayerWindow
	renderer *metalapi.Renderer2D
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
	renderer, err := metalapi.NewRenderer2D(window)
	if err != nil {
		_ = window.Destroy()
		return nil, err
	}
	return &metalDesktopBackend{window: window, renderer: renderer}, nil
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
	if b == nil || b.renderer == nil {
		return nil
	}
	return b.renderer.SetLogicalSize(w, h, pixelPerfect)
}

func (b *metalDesktopBackend) DrawFrame(clearColor geom.Color, cmds []render.DrawCmd, lines []string) error {
	if b == nil || b.renderer == nil {
		return nil
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
	if b.window != nil {
		if destroyErr := b.window.Destroy(); err == nil {
			err = destroyErr
		}
		b.window = nil
	}
	return err
}
