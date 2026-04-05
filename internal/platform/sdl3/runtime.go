package sdl3

import (
	"math"

	"github.com/Zyko0/go-sdl3/sdl"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
)

type Runtime struct {
	window         *sdl.Window
	renderer       *sdl.Renderer
	libraryLoaded  bool
	sdlInitialized bool
}

func New(title string, width, height int) (*Runtime, error) {
	if err := sdl.LoadLibrary(sdl.Path()); err != nil {
		return nil, err
	}

	rt := &Runtime{libraryLoaded: true}

	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		_ = rt.Destroy()
		return nil, err
	}
	rt.sdlInitialized = true

	window, renderer, err := sdl.CreateWindowAndRenderer(title, width, height, 0)
	if err != nil {
		_ = rt.Destroy()
		return nil, err
	}

	rt.window = window
	rt.renderer = renderer
	return rt, nil
}

func (rt *Runtime) PumpEvents() bool {
	var event sdl.Event

	for sdl.PollEvent(&event) {
		switch event.Type {
		case sdl.EVENT_QUIT:
			return true
		case sdl.EVENT_KEY_DOWN:
			key := event.KeyboardEvent()
			if key != nil && key.Scancode == sdl.SCANCODE_ESCAPE {
				return true
			}
		}
	}

	return false
}

func (rt *Runtime) DrawFrame(clearColor geom.Color, lines []string) error {
	if err := rt.renderer.SetDrawColor(
		colorByte(clearColor.R),
		colorByte(clearColor.G),
		colorByte(clearColor.B),
		colorByte(clearColor.A),
	); err != nil {
		return err
	}

	if err := rt.renderer.Clear(); err != nil {
		return err
	}

	if err := rt.renderer.SetDrawColor(240, 226, 188, 255); err != nil {
		return err
	}

	for i, line := range lines {
		if err := rt.renderer.DebugText(16, float32(16+i*18), line); err != nil {
			return err
		}
	}

	return rt.renderer.Present()
}

func (rt *Runtime) Destroy() error {
	if rt.renderer != nil {
		rt.renderer.Destroy()
		rt.renderer = nil
	}

	if rt.window != nil {
		rt.window.Destroy()
		rt.window = nil
	}

	if rt.sdlInitialized {
		sdl.Quit()
		rt.sdlInitialized = false
	}

	if rt.libraryLoaded {
		rt.libraryLoaded = false
		return sdl.CloseLibrary()
	}

	return nil
}

func colorByte(value float32) uint8 {
	clamped := math.Max(0, math.Min(1, float64(value)))
	return uint8(clamped * 255)
}
