package sdl3

import (
	"testing"

	"github.com/Zyko0/go-sdl3/sdl"
)

func TestBuildKeyBindings_knownKeys(t *testing.T) {
	// Each key maps to a unique control so the scancode→control assertion
	// is deterministic (no two keys share the same control here).
	km := map[string]string{
		"w":     "move_up",
		"a":     "move_left",
		"space": "jump",
	}

	bindings := buildKeyBindings(km)

	if len(bindings) != len(km) {
		t.Fatalf("expected %d bindings, got %d", len(km), len(bindings))
	}

	// Index by scancode for assertion.
	byScancode := make(map[sdl.Scancode]string, len(bindings))
	for _, b := range bindings {
		byScancode[b.scancode] = b.control
	}

	cases := []struct {
		scancode sdl.Scancode
		control  string
	}{
		{sdl.SCANCODE_W, "move_up"},
		{sdl.SCANCODE_A, "move_left"},
		{sdl.SCANCODE_SPACE, "jump"},
	}

	for _, c := range cases {
		ctrl, ok := byScancode[c.scancode]
		if !ok {
			t.Errorf("scancode %v not found in bindings", c.scancode)
			continue
		}
		if ctrl != c.control {
			t.Errorf("scancode %v: expected control %q, got %q", c.scancode, c.control, ctrl)
		}
	}
}

func TestBuildKeyBindings_unknownKeyIgnored(t *testing.T) {
	km := map[string]string{
		"w":           "move_up",
		"not_a_key":   "some_action",
		"also_bogus":  "another_action",
	}

	bindings := buildKeyBindings(km)

	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding (only 'w' is valid), got %d", len(bindings))
	}
	if bindings[0].control != "move_up" {
		t.Errorf("expected control 'move_up', got %q", bindings[0].control)
	}
}

func TestBuildKeyBindings_emptyMap(t *testing.T) {
	bindings := buildKeyBindings(map[string]string{})

	if len(bindings) != 0 {
		t.Fatalf("expected 0 bindings for empty map, got %d", len(bindings))
	}
}

func TestBuildKeyBindings_nilMap(t *testing.T) {
	bindings := buildKeyBindings(nil)

	if len(bindings) != 0 {
		t.Fatalf("expected 0 bindings for nil map, got %d", len(bindings))
	}
}

// TestBuildKeyBindings_sharedControl verifies that multiple keys can map to
// the same control name. This is the foundation for the two-pass OR logic in
// UpdateInput: if "w" and "up" both map to "move_up", both bindings must be
// present so the second pass can set the control true when either key is held.
func TestBuildKeyBindings_sharedControl(t *testing.T) {
	km := map[string]string{
		"w":  "move_up",
		"up": "move_up",
	}

	bindings := buildKeyBindings(km)

	if len(bindings) != 2 {
		t.Fatalf("expected 2 bindings for two keys sharing a control, got %d", len(bindings))
	}

	scancodes := map[sdl.Scancode]bool{}
	for _, b := range bindings {
		if b.control != "move_up" {
			t.Errorf("expected control 'move_up', got %q", b.control)
		}
		scancodes[b.scancode] = true
	}

	if !scancodes[sdl.SCANCODE_W] {
		t.Error("expected SCANCODE_W in bindings")
	}
	if !scancodes[sdl.SCANCODE_UP] {
		t.Error("expected SCANCODE_UP in bindings")
	}
}

func TestNameToScancode_coverage(t *testing.T) {
	// Spot-check that key categories are present and map to the right scancodes.
	cases := []struct {
		name     string
		scancode sdl.Scancode
	}{
		// Letters
		{"a", sdl.SCANCODE_A},
		{"z", sdl.SCANCODE_Z},
		// Digits
		{"0", sdl.SCANCODE_0},
		{"9", sdl.SCANCODE_9},
		// Arrow keys
		{"up", sdl.SCANCODE_UP},
		{"down", sdl.SCANCODE_DOWN},
		{"left", sdl.SCANCODE_LEFT},
		{"right", sdl.SCANCODE_RIGHT},
		// Named keys
		{"space", sdl.SCANCODE_SPACE},
		{"enter", sdl.SCANCODE_RETURN},
		{"escape", sdl.SCANCODE_ESCAPE},
		{"lshift", sdl.SCANCODE_LSHIFT},
		{"rshift", sdl.SCANCODE_RSHIFT},
		{"lctrl", sdl.SCANCODE_LCTRL},
		{"tab", sdl.SCANCODE_TAB},
		{"backspace", sdl.SCANCODE_BACKSPACE},
		// Function keys
		{"f1", sdl.SCANCODE_F1},
		{"f12", sdl.SCANCODE_F12},
	}

	for _, c := range cases {
		sc, ok := nameToScancode[c.name]
		if !ok {
			t.Errorf("key %q missing from nameToScancode", c.name)
			continue
		}
		if sc != c.scancode {
			t.Errorf("key %q: expected scancode %v, got %v", c.name, c.scancode, sc)
		}
	}
}
