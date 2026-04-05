package input

import "testing"

func TestPressedAndJustPressed(t *testing.T) {
	state := NewState()
	state.Bind("jump", "Space")
	state.SetPressed("Space", true)

	if !state.Pressed("jump") {
		t.Fatal("expected jump to be pressed")
	}

	if !state.JustPressed("jump") {
		t.Fatal("expected jump to be just pressed")
	}

	state.NextFrame()

	if state.JustPressed("jump") {
		t.Fatal("did not expect jump to be just pressed after NextFrame")
	}
}

func TestAxis(t *testing.T) {
	state := NewState()
	state.Bind("left", "A")
	state.Bind("right", "D")
	state.SetPressed("D", true)

	if state.Axis("left", "right") != 1 {
		t.Fatalf("expected axis to be 1, got %v", state.Axis("left", "right"))
	}
}
