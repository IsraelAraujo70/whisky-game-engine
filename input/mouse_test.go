package input

import "testing"

func TestMouseButtonPressedAndJustPressed(t *testing.T) {
	m := NewMouseState()
	m.SetButton(MouseButtonLeft, true)

	if !m.ButtonPressed(MouseButtonLeft) {
		t.Fatal("expected left button to be pressed")
	}
	if !m.JustPressed(MouseButtonLeft) {
		t.Fatal("expected left button to be just pressed")
	}

	m.NextFrame()

	if !m.ButtonPressed(MouseButtonLeft) {
		t.Fatal("expected left button to still be pressed after NextFrame")
	}
	if m.JustPressed(MouseButtonLeft) {
		t.Fatal("did not expect left button to be just pressed after NextFrame")
	}
}

func TestMouseJustReleased(t *testing.T) {
	m := NewMouseState()
	m.SetButton(MouseButtonRight, true)
	m.NextFrame()

	m.SetButton(MouseButtonRight, false)

	if m.ButtonPressed(MouseButtonRight) {
		t.Fatal("expected right button to not be pressed")
	}
	if !m.JustReleased(MouseButtonRight) {
		t.Fatal("expected right button to be just released")
	}

	m.NextFrame()

	if m.JustReleased(MouseButtonRight) {
		t.Fatal("did not expect right button to be just released after NextFrame")
	}
}

func TestMousePosition(t *testing.T) {
	m := NewMouseState()
	m.SetPosition(100.5, 200.75)

	x, y := m.Position()
	if x != 100.5 || y != 200.75 {
		t.Fatalf("expected position (100.5, 200.75), got (%.2f, %.2f)", x, y)
	}
}

func TestMouseWheel(t *testing.T) {
	m := NewMouseState()
	m.AddWheel(0, 3.0)
	m.AddWheel(0, -1.0) // multiple events per frame

	wx, wy := m.Wheel()
	if wx != 0 || wy != 2.0 {
		t.Fatalf("expected wheel (0, 2), got (%.2f, %.2f)", wx, wy)
	}

	m.NextFrame()

	wx, wy = m.Wheel()
	if wx != 0 || wy != 0 {
		t.Fatalf("expected wheel (0, 0) after NextFrame, got (%.2f, %.2f)", wx, wy)
	}
}

func TestMouseOutOfRangeButton(t *testing.T) {
	m := NewMouseState()

	// Should not panic.
	m.SetButton(MouseButton(-1), true)
	m.SetButton(MouseButton(99), true)

	if m.ButtonPressed(MouseButton(-1)) {
		t.Fatal("out-of-range button should not be pressed")
	}
	if m.JustPressed(MouseButton(99)) {
		t.Fatal("out-of-range button should not be just pressed")
	}
	if m.JustReleased(MouseButton(99)) {
		t.Fatal("out-of-range button should not be just released")
	}
}
