package input

type State struct {
	bindings map[string][]string
	current  map[string]bool
	previous map[string]bool

	mouse    *MouseState
	gamepads [MaxGamepads]*GamepadState
}

func NewState() *State {
	s := &State{
		bindings: make(map[string][]string),
		current:  make(map[string]bool),
		previous: make(map[string]bool),
		mouse:    NewMouseState(),
	}
	for i := range s.gamepads {
		s.gamepads[i] = NewGamepadState()
	}
	return s
}

func (s *State) Bind(action string, controls ...string) {
	s.bindings[action] = append(s.bindings[action], controls...)
}

func (s *State) SetPressed(control string, pressed bool) {
	s.current[control] = pressed
}

func (s *State) Pressed(action string) bool {
	for _, control := range s.bindings[action] {
		if s.current[control] {
			return true
		}
	}

	return false
}

func (s *State) JustPressed(action string) bool {
	for _, control := range s.bindings[action] {
		if s.current[control] && !s.previous[control] {
			return true
		}
	}

	return false
}

func (s *State) Axis(negativeAction, positiveAction string) float64 {
	var axis float64

	if s.Pressed(negativeAction) {
		axis -= 1
	}

	if s.Pressed(positiveAction) {
		axis += 1
	}

	return axis
}

// Mouse returns the shared MouseState managed by the platform layer.
func (s *State) Mouse() *MouseState {
	return s.mouse
}

// Gamepad returns the GamepadState for the given slot (0 to MaxGamepads-1).
func (s *State) Gamepad(index int) *GamepadState {
	if index < 0 || index >= MaxGamepads {
		return nil
	}
	return s.gamepads[index]
}

// AnyControlJustPressed returns the first control that transitioned from
// released to pressed this frame, or ("", false) if none did.
func (s *State) AnyControlJustPressed() (string, bool) {
	for control, pressed := range s.current {
		if pressed && !s.previous[control] {
			return control, true
		}
	}
	return "", false
}

// Controls returns a snapshot of all control names currently tracked.
func (s *State) Controls() []string {
	out := make([]string, 0, len(s.current))
	for control := range s.current {
		out = append(out, control)
	}
	return out
}

func (s *State) NextFrame() {
	next := make(map[string]bool, len(s.current))
	for control, pressed := range s.current {
		next[control] = pressed
	}

	s.previous = next
	s.mouse.NextFrame()
	for i := range s.gamepads {
		s.gamepads[i].NextFrame()
	}
}
