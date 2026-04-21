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

func TestAnyControlJustPressed(t *testing.T) {
	state := NewState()
	state.SetPressed("Space", true)

	control, ok := state.AnyControlJustPressed()
	if !ok || control != "Space" {
		t.Fatalf("expected Space to be just pressed, got %q, %v", control, ok)
	}

	state.NextFrame()
	state.SetPressed("Space", true)

	_, ok = state.AnyControlJustPressed()
	if ok {
		t.Fatal("expected no control to be just pressed after NextFrame with same state")
	}
}

func TestControls(t *testing.T) {
	state := NewState()
	state.SetPressed("W", true)
	state.SetPressed("A", true)

	controls := state.Controls()
	if len(controls) != 2 {
		t.Fatalf("expected 2 controls, got %d", len(controls))
	}
	found := map[string]bool{}
	for _, c := range controls {
		found[c] = true
	}
	if !found["W"] || !found["A"] {
		t.Fatalf("expected W and A in controls, got %v", controls)
	}
}
