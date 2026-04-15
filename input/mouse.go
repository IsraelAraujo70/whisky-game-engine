package input

// MouseButton identifies a mouse button.
type MouseButton int

const (
	MouseButtonLeft   MouseButton = iota // Primary (left) button
	MouseButtonRight                     // Secondary (right) button
	MouseButtonMiddle                    // Middle (wheel) button
	MouseButtonX1                        // Extra button 1
	MouseButtonX2                        // Extra button 2
	mouseButtonCount
)

// MouseState tracks the position, button state, and scroll wheel of the mouse.
// It is updated once per frame by the platform layer. Game code reads it via
// the Context.Mouse() accessor.
type MouseState struct {
	x, y         float64
	wheelX, wheelY float64

	current  [mouseButtonCount]bool
	previous [mouseButtonCount]bool
}

// NewMouseState returns a zero-initialized MouseState ready for use.
func NewMouseState() *MouseState {
	return &MouseState{}
}

// SetPosition is called by the platform layer to update the cursor position.
func (m *MouseState) SetPosition(x, y float64) {
	m.x = x
	m.y = y
}

// Position returns the current cursor position relative to the window, in
// virtual (logical) coordinates when SDL logical presentation is active.
func (m *MouseState) Position() (x, y float64) {
	return m.x, m.y
}

// SetButton is called by the platform layer to update a button's pressed state.
func (m *MouseState) SetButton(btn MouseButton, pressed bool) {
	if btn >= 0 && btn < mouseButtonCount {
		m.current[btn] = pressed
	}
}

// ButtonPressed returns true if the given button is currently held down.
func (m *MouseState) ButtonPressed(btn MouseButton) bool {
	if btn >= 0 && btn < mouseButtonCount {
		return m.current[btn]
	}
	return false
}

// JustPressed returns true if the button transitioned from released to pressed
// this frame.
func (m *MouseState) JustPressed(btn MouseButton) bool {
	if btn >= 0 && btn < mouseButtonCount {
		return m.current[btn] && !m.previous[btn]
	}
	return false
}

// JustReleased returns true if the button transitioned from pressed to released
// this frame.
func (m *MouseState) JustReleased(btn MouseButton) bool {
	if btn >= 0 && btn < mouseButtonCount {
		return !m.current[btn] && m.previous[btn]
	}
	return false
}

// AddWheel accumulates scroll delta for this frame. Called by the platform
// layer; may be called multiple times per frame if several scroll events arrive.
func (m *MouseState) AddWheel(dx, dy float64) {
	m.wheelX += dx
	m.wheelY += dy
}

// Wheel returns the accumulated scroll delta since the last frame.
// Positive Y means the user scrolled away from themselves (up).
func (m *MouseState) Wheel() (x, y float64) {
	return m.wheelX, m.wheelY
}

// NextFrame copies the current button state into previous and resets the wheel
// accumulator. Must be called once per game loop iteration.
func (m *MouseState) NextFrame() {
	m.previous = m.current
	m.wheelX = 0
	m.wheelY = 0
}
