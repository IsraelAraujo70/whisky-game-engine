package sdl3

import (
	"github.com/Zyko0/go-sdl3/sdl"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/input"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
)

// keyBinding associates an SDL scancode with a control name fed into the
// input system.
type keyBinding struct {
	scancode sdl.Scancode
	control  string
}

// nameToScancode maps human-readable key names to SDL scancodes. Games use
// these names in whisky.Config.KeyMap to define their own control mappings.
var nameToScancode = map[string]sdl.Scancode{
	// Letters
	"a": sdl.SCANCODE_A, "b": sdl.SCANCODE_B, "c": sdl.SCANCODE_C,
	"d": sdl.SCANCODE_D, "e": sdl.SCANCODE_E, "f": sdl.SCANCODE_F,
	"g": sdl.SCANCODE_G, "h": sdl.SCANCODE_H, "i": sdl.SCANCODE_I,
	"j": sdl.SCANCODE_J, "k": sdl.SCANCODE_K, "l": sdl.SCANCODE_L,
	"m": sdl.SCANCODE_M, "n": sdl.SCANCODE_N, "o": sdl.SCANCODE_O,
	"p": sdl.SCANCODE_P, "q": sdl.SCANCODE_Q, "r": sdl.SCANCODE_R,
	"s": sdl.SCANCODE_S, "t": sdl.SCANCODE_T, "u": sdl.SCANCODE_U,
	"v": sdl.SCANCODE_V, "w": sdl.SCANCODE_W, "x": sdl.SCANCODE_X,
	"y": sdl.SCANCODE_Y, "z": sdl.SCANCODE_Z,
	// Digits
	"0": sdl.SCANCODE_0, "1": sdl.SCANCODE_1, "2": sdl.SCANCODE_2,
	"3": sdl.SCANCODE_3, "4": sdl.SCANCODE_4, "5": sdl.SCANCODE_5,
	"6": sdl.SCANCODE_6, "7": sdl.SCANCODE_7, "8": sdl.SCANCODE_8,
	"9": sdl.SCANCODE_9,
	// Arrow keys
	"up":    sdl.SCANCODE_UP,
	"down":  sdl.SCANCODE_DOWN,
	"left":  sdl.SCANCODE_LEFT,
	"right": sdl.SCANCODE_RIGHT,
	// Named keys
	"space":     sdl.SCANCODE_SPACE,
	"enter":     sdl.SCANCODE_RETURN,
	"escape":    sdl.SCANCODE_ESCAPE,
	"backspace": sdl.SCANCODE_BACKSPACE,
	"tab":       sdl.SCANCODE_TAB,
	"lshift":    sdl.SCANCODE_LSHIFT,
	"rshift":    sdl.SCANCODE_RSHIFT,
	"lctrl":     sdl.SCANCODE_LCTRL,
	"rctrl":     sdl.SCANCODE_RCTRL,
	"lalt":      sdl.SCANCODE_LALT,
	"ralt":      sdl.SCANCODE_RALT,
	// Function keys
	"f1": sdl.SCANCODE_F1, "f2": sdl.SCANCODE_F2, "f3": sdl.SCANCODE_F3,
	"f4": sdl.SCANCODE_F4, "f5": sdl.SCANCODE_F5, "f6": sdl.SCANCODE_F6,
	"f7": sdl.SCANCODE_F7, "f8": sdl.SCANCODE_F8, "f9": sdl.SCANCODE_F9,
	"f10": sdl.SCANCODE_F10, "f11": sdl.SCANCODE_F11, "f12": sdl.SCANCODE_F12,
}

// buildKeyBindings converts a key-name→control map into resolved SDL bindings.
// Unknown key names are silently ignored.
func buildKeyBindings(km map[string]string) []keyBinding {
	bindings := make([]keyBinding, 0, len(km))
	for keyName, control := range km {
		if sc, ok := nameToScancode[keyName]; ok {
			bindings = append(bindings, keyBinding{scancode: sc, control: control})
		}
	}
	return bindings
}

type Runtime struct {
	window         *sdl.Window
	renderer       *sdl.Renderer
	libraryLoaded  bool
	sdlInitialized bool
	keyBindings    []keyBinding
}

// New creates an SDL3 window and renderer. keyMap maps key names (e.g. "space",
// "w") to control names used by the input system (e.g. "jump", "move_up").
func New(title string, width, height int, keyMap map[string]string) (*Runtime, error) {
	if err := sdl.LoadLibrary(sdl.Path()); err != nil {
		return nil, err
	}

	rt := &Runtime{
		libraryLoaded: true,
		keyBindings:   buildKeyBindings(keyMap),
	}

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

// UpdateInput reads the current keyboard state and feeds it into the input
// system so that action bindings (Pressed / JustPressed / Axis) work.
func (rt *Runtime) UpdateInput(state *input.State) {
	keys := sdl.GetKeyboardState()
	for _, kb := range rt.keyBindings {
		state.SetPressed(kb.control, keys[kb.scancode])
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

