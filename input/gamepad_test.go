package input

import "testing"

func TestGamepadButtonPressedAndJustPressed(t *testing.T) {
	g := NewGamepadState()
	g.SetConnected(true)
	g.SetButton(GamepadButtonA, true)

	if !g.ButtonPressed(GamepadButtonA) {
		t.Fatal("expected A button to be pressed")
	}
	if !g.JustPressed(GamepadButtonA) {
		t.Fatal("expected A button to be just pressed")
	}

	g.NextFrame()

	if !g.ButtonPressed(GamepadButtonA) {
		t.Fatal("expected A button to still be pressed after NextFrame")
	}
	if g.JustPressed(GamepadButtonA) {
		t.Fatal("did not expect A button to be just pressed after NextFrame")
	}
}

func TestGamepadJustReleased(t *testing.T) {
	g := NewGamepadState()
	g.SetConnected(true)
	g.SetButton(GamepadButtonB, true)
	g.NextFrame()

	g.SetButton(GamepadButtonB, false)

	if g.ButtonPressed(GamepadButtonB) {
		t.Fatal("expected B button to not be pressed")
	}
	if !g.JustReleased(GamepadButtonB) {
		t.Fatal("expected B button to be just released")
	}

	g.NextFrame()

	if g.JustReleased(GamepadButtonB) {
		t.Fatal("did not expect B button to be just released after NextFrame")
	}
}

func TestGamepadAxis(t *testing.T) {
	g := NewGamepadState()
	g.SetConnected(true)
	g.SetAxis(GamepadAxisLX, -0.75)
	g.SetAxis(GamepadAxisRT, 1.0)

	if g.Axis(GamepadAxisLX) != -0.75 {
		t.Fatalf("expected LX=-0.75, got %v", g.Axis(GamepadAxisLX))
	}
	if g.Axis(GamepadAxisRT) != 1.0 {
		t.Fatalf("expected RT=1.0, got %v", g.Axis(GamepadAxisRT))
	}
}

func TestGamepadDisconnectResetsState(t *testing.T) {
	g := NewGamepadState()
	g.SetConnected(true)
	g.SetButton(GamepadButtonA, true)
	g.SetAxis(GamepadAxisLX, 0.5)

	g.SetConnected(false)

	if g.Connected() {
		t.Fatal("expected gamepad to be disconnected")
	}
	if g.ButtonPressed(GamepadButtonA) {
		t.Fatal("expected A button to be reset on disconnect")
	}
	if g.Axis(GamepadAxisLX) != 0 {
		t.Fatalf("expected LX=0 on disconnect, got %v", g.Axis(GamepadAxisLX))
	}
}

func TestGamepadOutOfRangeButton(t *testing.T) {
	g := NewGamepadState()

	// Should not panic.
	g.SetButton(GamepadButton(-1), true)
	g.SetButton(GamepadButton(99), true)

	if g.ButtonPressed(GamepadButton(-1)) {
		t.Fatal("out-of-range button should not be pressed")
	}
	if g.JustPressed(GamepadButton(99)) {
		t.Fatal("out-of-range button should not be just pressed")
	}
}

func TestGamepadOutOfRangeAxis(t *testing.T) {
	g := NewGamepadState()

	// Should not panic.
	g.SetAxis(GamepadAxis(-1), 1.0)
	g.SetAxis(GamepadAxis(99), 1.0)

	if g.Axis(GamepadAxis(-1)) != 0 {
		t.Fatal("out-of-range axis should return 0")
	}
	if g.Axis(GamepadAxis(99)) != 0 {
		t.Fatal("out-of-range axis should return 0")
	}
}
