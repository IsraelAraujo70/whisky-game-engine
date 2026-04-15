package input

// GamepadButton identifies a gamepad button using Xbox-style naming.
type GamepadButton int

const (
	GamepadButtonA          GamepadButton = iota // South face button (Xbox A, PS Cross)
	GamepadButtonB                               // East face button (Xbox B, PS Circle)
	GamepadButtonX                               // West face button (Xbox X, PS Square)
	GamepadButtonY                               // North face button (Xbox Y, PS Triangle)
	GamepadButtonBack                            // Back / Select / Share
	GamepadButtonGuide                           // Guide / PS / Home
	GamepadButtonStart                           // Start / Options / Menu
	GamepadButtonLeftStick                       // Left stick click (L3)
	GamepadButtonRightStick                      // Right stick click (R3)
	GamepadButtonLB                              // Left shoulder (LB / L1)
	GamepadButtonRB                              // Right shoulder (RB / R1)
	GamepadButtonDPadUp                          // D-Pad up
	GamepadButtonDPadDown                        // D-Pad down
	GamepadButtonDPadLeft                        // D-Pad left
	GamepadButtonDPadRight                       // D-Pad right
	gamepadButtonCount
)

// GamepadAxis identifies an analog axis on a gamepad.
type GamepadAxis int

const (
	GamepadAxisLX GamepadAxis = iota // Left stick horizontal (-1 left, +1 right)
	GamepadAxisLY                    // Left stick vertical (-1 up, +1 down)
	GamepadAxisRX                    // Right stick horizontal
	GamepadAxisRY                    // Right stick vertical
	GamepadAxisLT                    // Left trigger (0 released, +1 fully pressed)
	GamepadAxisRT                    // Right trigger
	gamepadAxisCount
)

// MaxGamepads is the maximum number of gamepads tracked simultaneously.
const MaxGamepads = 4

// GamepadState tracks button and axis state for a single gamepad. It is
// updated once per frame by the platform layer. Game code reads it via
// Context.Gamepad(index).
type GamepadState struct {
	connected bool

	axes [gamepadAxisCount]float64

	current  [gamepadButtonCount]bool
	previous [gamepadButtonCount]bool
}

// NewGamepadState returns a zero-initialized GamepadState.
func NewGamepadState() *GamepadState {
	return &GamepadState{}
}

// Connected returns true if a physical gamepad is currently plugged in for
// this slot.
func (g *GamepadState) Connected() bool {
	return g.connected
}

// SetConnected is called by the platform layer to mark the pad as plugged
// in or removed.
func (g *GamepadState) SetConnected(connected bool) {
	g.connected = connected
	if !connected {
		// Reset all state when disconnected.
		g.axes = [gamepadAxisCount]float64{}
		g.current = [gamepadButtonCount]bool{}
		g.previous = [gamepadButtonCount]bool{}
	}
}

// SetButton is called by the platform layer to update a button's state.
func (g *GamepadState) SetButton(btn GamepadButton, pressed bool) {
	if btn >= 0 && btn < gamepadButtonCount {
		g.current[btn] = pressed
	}
}

// ButtonPressed returns true if the button is currently held.
func (g *GamepadState) ButtonPressed(btn GamepadButton) bool {
	if btn >= 0 && btn < gamepadButtonCount {
		return g.current[btn]
	}
	return false
}

// JustPressed returns true if the button transitioned from released to pressed
// this frame.
func (g *GamepadState) JustPressed(btn GamepadButton) bool {
	if btn >= 0 && btn < gamepadButtonCount {
		return g.current[btn] && !g.previous[btn]
	}
	return false
}

// JustReleased returns true if the button transitioned from pressed to released
// this frame.
func (g *GamepadState) JustReleased(btn GamepadButton) bool {
	if btn >= 0 && btn < gamepadButtonCount {
		return !g.current[btn] && g.previous[btn]
	}
	return false
}

// SetAxis is called by the platform layer to update an axis value.
// Values are normalized to [-1, +1] for sticks and [0, +1] for triggers.
func (g *GamepadState) SetAxis(axis GamepadAxis, value float64) {
	if axis >= 0 && axis < gamepadAxisCount {
		g.axes[axis] = value
	}
}

// Axis returns the current value of the given analog axis.
func (g *GamepadState) Axis(axis GamepadAxis) float64 {
	if axis >= 0 && axis < gamepadAxisCount {
		return g.axes[axis]
	}
	return 0
}

// NextFrame copies current button state into previous. Must be called once
// per game loop iteration.
func (g *GamepadState) NextFrame() {
	g.previous = g.current
}
