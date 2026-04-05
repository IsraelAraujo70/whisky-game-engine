package input

type State struct {
	bindings map[string][]string
	current  map[string]bool
	previous map[string]bool
}

func NewState() *State {
	return &State{
		bindings: make(map[string][]string),
		current:  make(map[string]bool),
		previous: make(map[string]bool),
	}
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

func (s *State) NextFrame() {
	next := make(map[string]bool, len(s.current))
	for control, pressed := range s.current {
		next[control] = pressed
	}

	s.previous = next
}
