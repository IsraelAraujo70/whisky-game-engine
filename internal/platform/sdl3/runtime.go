package sdl3

import (
	"github.com/Zyko0/go-sdl3/sdl"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/input"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
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

// SetLogicalSize configures virtual resolution scaling. When pixelPerfect is
// true integer scaling is used; otherwise letterboxing is applied.
func (rt *Runtime) SetLogicalSize(w, h int, pixelPerfect bool) error {
	mode := sdl.LOGICAL_PRESENTATION_LETTERBOX
	if pixelPerfect {
		mode = sdl.LOGICAL_PRESENTATION_INTEGER_SCALE
	}
	return rt.renderer.SetLogicalPresentation(int32(w), int32(h), mode)
}

// keyMap maps SDL scancodes to engine control names.
//
// TODO: This table is currently hardcoded. It should be made configurable so
// that games can define their own scancode-to-control mappings (e.g. via a
// KeyMap field on Config or a RegisterKey API) instead of relying on a fixed
// set of controls built into the platform layer.
var keyMap = []struct {
	scancode sdl.Scancode
	name     string
}{
	{sdl.SCANCODE_W, "w"},
	{sdl.SCANCODE_A, "a"},
	{sdl.SCANCODE_S, "s"},
	{sdl.SCANCODE_D, "d"},
	{sdl.SCANCODE_UP, "up"},
	{sdl.SCANCODE_DOWN, "down"},
	{sdl.SCANCODE_LEFT, "left"},
	{sdl.SCANCODE_RIGHT, "right"},
	{sdl.SCANCODE_SPACE, "space"},
	{sdl.SCANCODE_LSHIFT, "lshift"},
	{sdl.SCANCODE_RETURN, "enter"},
}

// UpdateInput reads the current keyboard state and feeds it into the input
// system so that action bindings (Pressed / JustPressed / Axis) work.
func (rt *Runtime) UpdateInput(state *input.State) {
	keys := sdl.GetKeyboardState()
	for _, km := range keyMap {
		state.SetPressed(km.name, keys[km.scancode])
	}
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

func (rt *Runtime) DrawFrame(clearColor geom.Color, rects []render.FillRect, lines []string) error {
	if err := rt.renderer.SetDrawColorFloat(
		clearColor.R, clearColor.G, clearColor.B, clearColor.A,
	); err != nil {
		return err
	}

	if err := rt.renderer.Clear(); err != nil {
		return err
	}

	// Draw filled rectangles (tiles, player, etc.).
	for _, r := range rects {
		if err := rt.renderer.SetDrawColorFloat(
			r.Color.R, r.Color.G, r.Color.B, r.Color.A,
		); err != nil {
			return err
		}
		fr := sdl.FRect{
			X: float32(r.Rect.X),
			Y: float32(r.Rect.Y),
			W: float32(r.Rect.W),
			H: float32(r.Rect.H),
		}
		if err := rt.renderer.RenderFillRect(&fr); err != nil {
			return err
		}
	}

	// Debug text overlay on top of everything.
	if err := rt.renderer.SetDrawColor(240, 226, 188, 255); err != nil {
		return err
	}
	for i, line := range lines {
		if err := rt.renderer.DebugText(4, float32(4+i*10), line); err != nil {
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

