package sdl3

import (
	"os"

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

// sdlButtonToMouse maps SDL mouse button indices (1-based) to our MouseButton
// enum. SDL uses: 1=Left, 2=Middle, 3=Right, 4=X1, 5=X2.
var sdlButtonToMouse = map[uint8]input.MouseButton{
	1: input.MouseButtonLeft,
	2: input.MouseButtonMiddle,
	3: input.MouseButtonRight,
	4: input.MouseButtonX1,
	5: input.MouseButtonX2,
}

// sdlButtonToGamepad maps SDL GamepadButton to our GamepadButton enum.
var sdlButtonToGamepad = map[sdl.GamepadButton]input.GamepadButton{
	sdl.GAMEPAD_BUTTON_SOUTH:          input.GamepadButtonA,
	sdl.GAMEPAD_BUTTON_EAST:           input.GamepadButtonB,
	sdl.GAMEPAD_BUTTON_WEST:           input.GamepadButtonX,
	sdl.GAMEPAD_BUTTON_NORTH:          input.GamepadButtonY,
	sdl.GAMEPAD_BUTTON_BACK:           input.GamepadButtonBack,
	sdl.GAMEPAD_BUTTON_GUIDE:          input.GamepadButtonGuide,
	sdl.GAMEPAD_BUTTON_START:          input.GamepadButtonStart,
	sdl.GAMEPAD_BUTTON_LEFT_STICK:     input.GamepadButtonLeftStick,
	sdl.GAMEPAD_BUTTON_RIGHT_STICK:    input.GamepadButtonRightStick,
	sdl.GAMEPAD_BUTTON_LEFT_SHOULDER:  input.GamepadButtonLB,
	sdl.GAMEPAD_BUTTON_RIGHT_SHOULDER: input.GamepadButtonRB,
	sdl.GAMEPAD_BUTTON_DPAD_UP:        input.GamepadButtonDPadUp,
	sdl.GAMEPAD_BUTTON_DPAD_DOWN:      input.GamepadButtonDPadDown,
	sdl.GAMEPAD_BUTTON_DPAD_LEFT:      input.GamepadButtonDPadLeft,
	sdl.GAMEPAD_BUTTON_DPAD_RIGHT:     input.GamepadButtonDPadRight,
}

// sdlAxisToGamepad maps SDL GamepadAxis to our GamepadAxis enum.
var sdlAxisToGamepad = map[sdl.GamepadAxis]input.GamepadAxis{
	sdl.GAMEPAD_AXIS_LEFTX:         input.GamepadAxisLX,
	sdl.GAMEPAD_AXIS_LEFTY:         input.GamepadAxisLY,
	sdl.GAMEPAD_AXIS_RIGHTX:        input.GamepadAxisRX,
	sdl.GAMEPAD_AXIS_RIGHTY:        input.GamepadAxisRY,
	sdl.GAMEPAD_AXIS_LEFT_TRIGGER:  input.GamepadAxisLT,
	sdl.GAMEPAD_AXIS_RIGHT_TRIGGER: input.GamepadAxisRT,
}

// openGamepad represents an active gamepad tracked by the runtime.
type openGamepad struct {
	pad   *sdl.Gamepad
	joyID sdl.JoystickID
	slot  int // index 0..MaxGamepads-1
}

type Runtime struct {
	window         *sdl.Window
	renderer       *sdl.Renderer
	textures       *textureCache
	libraryLoaded  bool
	sdlInitialized bool
	keyBindings    []keyBinding
	gamepads       []*openGamepad
}

// New creates an SDL3 window and renderer. keyMap maps key names (e.g. "space",
// "w") to control names used by the input system (e.g. "jump", "move_up").
func New(title string, width, height int, keyMap map[string]string) (*Runtime, error) {
	sdlPath := os.Getenv("WHISKY_SDL3_PATH")
	if sdlPath == "" {
		sdlPath = sdl.Path()
	}
	if err := sdl.LoadLibrary(sdlPath); err != nil {
		return nil, err
	}

	rt := &Runtime{
		libraryLoaded: true,
		keyBindings:   buildKeyBindings(keyMap),
	}

	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_GAMEPAD); err != nil {
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
	rt.textures = newTextureCache(renderer)
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
//
// Two passes are used so that multiple keys sharing the same control name
// (e.g. "w" and "up" both mapped to "move_up") are OR-ed correctly. A
// single-pass SetPressed would let the last-processed key overwrite a
// true value set by an earlier key, producing non-deterministic behavior.
func (rt *Runtime) UpdateInput(state *input.State) {
	keys := sdl.GetKeyboardState()
	// First pass: reset every tracked control to false.
	for _, kb := range rt.keyBindings {
		state.SetPressed(kb.control, false)
	}
	// Second pass: set to true if any bound key is currently pressed.
	for _, kb := range rt.keyBindings {
		if keys[kb.scancode] {
			state.SetPressed(kb.control, true)
		}
	}
}

// UpdateMouse reads the current mouse state via SDL3 polling.
// Button press/release events are handled in PumpEvents for precise edge
// detection; here we update the cursor position every frame.
func (rt *Runtime) UpdateMouse(mouse *input.MouseState) {
	buttons, mx, my := sdl.GetMouseState()
	mouse.SetPosition(float64(mx), float64(my))

	// Poll-based button state (OR with event-driven state from PumpEvents).
	mouse.SetButton(input.MouseButtonLeft, buttons&sdl.ButtonMask(sdl.BUTTON_LEFT) != 0)
	mouse.SetButton(input.MouseButtonMiddle, buttons&sdl.ButtonMask(sdl.BUTTON_MIDDLE) != 0)
	mouse.SetButton(input.MouseButtonRight, buttons&sdl.ButtonMask(sdl.BUTTON_RIGHT) != 0)
	mouse.SetButton(input.MouseButtonX1, buttons&sdl.ButtonMask(sdl.BUTTON_X1) != 0)
	mouse.SetButton(input.MouseButtonX2, buttons&sdl.ButtonMask(sdl.BUTTON_X2) != 0)
}

// UpdateGamepads reads each connected gamepad's axes and buttons.
func (rt *Runtime) UpdateGamepads(pads [input.MaxGamepads]*input.GamepadState) {
	for _, og := range rt.gamepads {
		if og == nil || og.pad == nil || og.slot < 0 || og.slot >= input.MaxGamepads {
			continue
		}
		gs := pads[og.slot]
		if gs == nil {
			continue
		}

		// Axes — SDL returns int16 in [-32768, 32767]; normalize to [-1, 1].
		for sdlAxis, gamepadAxis := range sdlAxisToGamepad {
			raw := og.pad.Axis(sdlAxis)
			var norm float64
			if raw >= 0 {
				norm = float64(raw) / 32767.0
			} else {
				norm = float64(raw) / 32768.0
			}
			gs.SetAxis(gamepadAxis, norm)
		}

		// Buttons.
		for sdlBtn, gamepadBtn := range sdlButtonToGamepad {
			gs.SetButton(gamepadBtn, og.pad.Button(sdlBtn))
		}
	}
}

// PumpEvents processes the SDL event queue. It returns true when the user
// requests quitting (window close or Escape). Mouse wheel deltas are
// accumulated on the given MouseState. Gamepad add/remove events are
// handled internally.
func (rt *Runtime) PumpEvents(mouse *input.MouseState, pads [input.MaxGamepads]*input.GamepadState) bool {
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

		// Mouse wheel — accumulate delta on the MouseState.
		case sdl.EVENT_MOUSE_WHEEL:
			if mouse != nil {
				we := event.MouseWheelEvent()
				if we != nil {
					dy := float64(we.Y)
					dx := float64(we.X)
					if we.Direction == sdl.MOUSEWHEEL_FLIPPED {
						dx = -dx
						dy = -dy
					}
					mouse.AddWheel(dx, dy)
				}
			}

		// Gamepad hotplug.
		case sdl.EVENT_GAMEPAD_ADDED:
			ge := event.GamepadDeviceEvent()
			if ge != nil {
				rt.openGamepadSlot(ge.Which, pads)
			}
		case sdl.EVENT_GAMEPAD_REMOVED:
			ge := event.GamepadDeviceEvent()
			if ge != nil {
				rt.closeGamepad(ge.Which, pads)
			}
		}
	}

	return false
}

// openGamepadSlot opens the gamepad and assigns it to the first available slot.
func (rt *Runtime) openGamepadSlot(joyID sdl.JoystickID, pads [input.MaxGamepads]*input.GamepadState) {
	// Already tracked?
	for _, og := range rt.gamepads {
		if og != nil && og.joyID == joyID {
			return
		}
	}

	pad, err := joyID.OpenGamepad()
	if err != nil || pad == nil {
		return
	}

	// Find first free slot.
	slot := -1
	for i := 0; i < input.MaxGamepads; i++ {
		taken := false
		for _, og := range rt.gamepads {
			if og != nil && og.slot == i {
				taken = true
				break
			}
		}
		if !taken {
			slot = i
			break
		}
	}
	if slot < 0 {
		pad.Close()
		return
	}

	og := &openGamepad{pad: pad, joyID: joyID, slot: slot}
	rt.gamepads = append(rt.gamepads, og)
	if pads[slot] != nil {
		pads[slot].SetConnected(true)
	}
}

// closeGamepad removes the gamepad from tracking and marks its slot disconnected.
func (rt *Runtime) closeGamepad(joyID sdl.JoystickID, pads [input.MaxGamepads]*input.GamepadState) {
	for i, og := range rt.gamepads {
		if og != nil && og.joyID == joyID {
			if og.slot >= 0 && og.slot < input.MaxGamepads && pads[og.slot] != nil {
				pads[og.slot].SetConnected(false)
			}
			og.pad.Close()
			rt.gamepads = append(rt.gamepads[:i], rt.gamepads[i+1:]...)
			return
		}
	}
}

// OpenExistingGamepads scans for already-connected gamepads at startup.
func (rt *Runtime) OpenExistingGamepads(pads [input.MaxGamepads]*input.GamepadState) {
	ids, err := sdl.GetGamepads()
	if err != nil {
		return
	}
	for _, joyID := range ids {
		rt.openGamepadSlot(joyID, pads)
	}
}

func (rt *Runtime) LoadTexture(path string) (render.TextureID, int, int, error) {
	return rt.textures.Load(path)
}

func (rt *Runtime) DrawFrame(clearColor geom.Color, cmds []render.DrawCmd, lines []string) error {
	if err := rt.renderer.SetDrawColorFloat(
		clearColor.R, clearColor.G, clearColor.B, clearColor.A,
	); err != nil {
		return err
	}

	if err := rt.renderer.Clear(); err != nil {
		return err
	}

	for _, cmd := range cmds {
		switch c := cmd.(type) {
		case render.FillRect:
			if err := rt.renderer.SetDrawColorFloat(
				c.Color.R, c.Color.G, c.Color.B, c.Color.A,
			); err != nil {
				return err
			}
			fr := sdl.FRect{
				X: float32(c.Rect.X),
				Y: float32(c.Rect.Y),
				W: float32(c.Rect.W),
				H: float32(c.Rect.H),
			}
			if err := rt.renderer.RenderFillRect(&fr); err != nil {
				return err
			}
		case render.SpriteCmd:
			texture := rt.textures.Get(c.Texture)
			if texture == nil {
				continue
			}
			src := sdl.FRect{
				X: float32(c.Src.X),
				Y: float32(c.Src.Y),
				W: float32(c.Src.W),
				H: float32(c.Src.H),
			}
			dst := sdl.FRect{
				X: float32(c.Dst.X),
				Y: float32(c.Dst.Y),
				W: float32(c.Dst.W),
				H: float32(c.Dst.H),
			}

			if c.FlipH || c.FlipV {
				flip := sdl.FLIP_NONE
				if c.FlipH {
					flip |= sdl.FLIP_HORIZONTAL
				}
				if c.FlipV {
					flip |= sdl.FLIP_VERTICAL
				}
				if err := rt.renderer.RenderTextureRotated(texture, &src, &dst, 0, nil, flip); err != nil {
					return err
				}
				continue
			}

			if err := rt.renderer.RenderTexture(texture, &src, &dst); err != nil {
				return err
			}
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
	for _, og := range rt.gamepads {
		if og != nil && og.pad != nil {
			og.pad.Close()
		}
	}
	rt.gamepads = nil

	if rt.textures != nil {
		rt.textures.DestroyAll()
		rt.textures = nil
	}

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
