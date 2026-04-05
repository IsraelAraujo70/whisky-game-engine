package sdl3

import (
	"testing"

	"github.com/Zyko0/go-sdl3/sdl"
)

func TestBuildKeyBindings_knownKeys(t *testing.T) {
	km := map[string]string{
		"w":     "move_up",
		"a":     "move_left",
		"space": "jump",
		"up":    "move_up",
	}

	bindings := buildKeyBindings(km)

	if len(bindings) != len(km) {
		t.Fatalf("expected %d bindings, got %d", len(km), len(bindings))
	}

	// Index by control name for assertion.
	byControl := make(map[string]sdl.Scancode, len(bindings))
	for _, b := range bindings {
		byControl[b.control] = b.scancode
	}

	cases := []struct {
		control  string
		scancode sdl.Scancode
	}{
		{"move_up", sdl.SCANCODE_W},    // "w" → move_up resolves to W scancode
		{"move_left", sdl.SCANCODE_A},
		{"jump", sdl.SCANCODE_SPACE},
	}

	for _, c := range cases {
		sc, ok := byControl[c.control]
		if !ok {
			t.Errorf("control %q not found in bindings", c.control)
			continue
		}
		if sc != c.scancode {
			t.Errorf("control %q: expected scancode %v, got %v", c.control, c.scancode, sc)
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
