package x11

import "testing"

func TestBuildKeyBindingsKnownKeys(t *testing.T) {
	keyMap := map[string]string{
		"w":     "move_up",
		"a":     "move_left",
		"space": "jump",
	}

	bindings := buildKeyBindings(keyMap, func(keysym uintptr) byte {
		return map[uintptr]byte{
			0x0077: 24,
			0x0061: 38,
			0x0020: 65,
		}[keysym]
	})

	if len(bindings) != len(keyMap) {
		t.Fatalf("expected %d bindings, got %d", len(keyMap), len(bindings))
	}

	byKeycode := make(map[byte]string, len(bindings))
	for _, binding := range bindings {
		byKeycode[binding.keycode] = binding.control
	}

	cases := []struct {
		keycode byte
		control string
	}{
		{24, "move_up"},
		{38, "move_left"},
		{65, "jump"},
	}

	for _, c := range cases {
		control, ok := byKeycode[c.keycode]
		if !ok {
			t.Errorf("keycode %d not found in bindings", c.keycode)
			continue
		}
		if control != c.control {
			t.Errorf("keycode %d: expected control %q, got %q", c.keycode, c.control, control)
		}
	}
}

func TestBuildKeyBindingsUnknownKeyIgnored(t *testing.T) {
	keyMap := map[string]string{
		"w":         "move_up",
		"not_a_key": "some_action",
	}

	bindings := buildKeyBindings(keyMap, func(keysym uintptr) byte {
		if keysym == 0x0077 {
			return 24
		}
		return 0
	})

	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(bindings))
	}
	if bindings[0].control != "move_up" {
		t.Fatalf("expected control %q, got %q", "move_up", bindings[0].control)
	}
}

func TestBuildKeyBindingsSharedControl(t *testing.T) {
	keyMap := map[string]string{
		"w":  "move_up",
		"up": "move_up",
	}

	bindings := buildKeyBindings(keyMap, func(keysym uintptr) byte {
		return map[uintptr]byte{
			0x0077: 24,
			0xFF52: 111,
		}[keysym]
	})

	if len(bindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(bindings))
	}

	keycodes := map[byte]bool{}
	for _, binding := range bindings {
		if binding.control != "move_up" {
			t.Errorf("expected control %q, got %q", "move_up", binding.control)
		}
		keycodes[binding.keycode] = true
	}

	if !keycodes[24] {
		t.Error("expected keycode 24 (W) in bindings")
	}
	if !keycodes[111] {
		t.Error("expected keycode 111 (Up) in bindings")
	}
}

func TestNameToKeysymCoverage(t *testing.T) {
	cases := []struct {
		name   string
		keysym uintptr
	}{
		{"a", 0x0061},
		{"z", 0x007A},
		{"0", 0x0030},
		{"9", 0x0039},
		{"up", 0xFF52},
		{"down", 0xFF54},
		{"left", 0xFF51},
		{"right", 0xFF53},
		{"space", 0x0020},
		{"enter", 0xFF0D},
		{"escape", 0xFF1B},
		{"lshift", 0xFFE1},
		{"rshift", 0xFFE2},
		{"lctrl", 0xFFE3},
		{"tab", 0xFF09},
		{"backspace", 0xFF08},
		{"f1", 0xFFBE},
		{"f12", 0xFFC9},
	}

	for _, c := range cases {
		keysym, ok := nameToKeysym[c.name]
		if !ok {
			t.Errorf("key %q missing from nameToKeysym", c.name)
			continue
		}
		if keysym != c.keysym {
			t.Errorf("key %q: expected keysym %#x, got %#x", c.name, c.keysym, keysym)
		}
	}
}
