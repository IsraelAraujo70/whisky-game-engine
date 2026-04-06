package win32

import "testing"

func TestBuildKeyBindingsKnownKeys(t *testing.T) {
	km := map[string]string{
		"w":     "move_up",
		"a":     "move_left",
		"space": "jump",
	}

	bindings := buildKeyBindings(km)
	if len(bindings) != len(km) {
		t.Fatalf("expected %d bindings, got %d", len(km), len(bindings))
	}

	byVK := make(map[uint16]string, len(bindings))
	for _, b := range bindings {
		byVK[b.virtualKey] = b.control
	}

	cases := []struct {
		virtualKey uint16
		control    string
	}{
		{0x57, "move_up"},
		{0x41, "move_left"},
		{0x20, "jump"},
	}

	for _, c := range cases {
		control, ok := byVK[c.virtualKey]
		if !ok {
			t.Errorf("virtual key %x not found in bindings", c.virtualKey)
			continue
		}
		if control != c.control {
			t.Errorf("virtual key %x: expected control %q, got %q", c.virtualKey, c.control, control)
		}
	}
}

func TestBuildKeyBindingsUnknownKeyIgnored(t *testing.T) {
	km := map[string]string{
		"w":          "move_up",
		"not_a_key":  "some_action",
		"also_bogus": "another_action",
	}

	bindings := buildKeyBindings(km)
	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(bindings))
	}
	if bindings[0].control != "move_up" {
		t.Fatalf("expected control %q, got %q", "move_up", bindings[0].control)
	}
}

func TestBuildKeyBindingsSharedControl(t *testing.T) {
	km := map[string]string{
		"w":  "move_up",
		"up": "move_up",
	}

	bindings := buildKeyBindings(km)
	if len(bindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(bindings))
	}

	keys := map[uint16]bool{}
	for _, b := range bindings {
		if b.control != "move_up" {
			t.Errorf("expected control %q, got %q", "move_up", b.control)
		}
		keys[b.virtualKey] = true
	}

	if !keys[0x57] {
		t.Error("expected virtual key 0x57 (W) in bindings")
	}
	if !keys[0x26] {
		t.Error("expected virtual key 0x26 (Up) in bindings")
	}
}

func TestNameToVirtualKeyCoverage(t *testing.T) {
	cases := []struct {
		name       string
		virtualKey uint16
	}{
		{"a", 0x41},
		{"z", 0x5A},
		{"0", 0x30},
		{"9", 0x39},
		{"up", 0x26},
		{"down", 0x28},
		{"left", 0x25},
		{"right", 0x27},
		{"space", 0x20},
		{"enter", 0x0D},
		{"escape", 0x1B},
		{"lshift", 0xA0},
		{"rshift", 0xA1},
		{"lctrl", 0xA2},
		{"tab", 0x09},
		{"backspace", 0x08},
		{"f1", 0x70},
		{"f12", 0x7B},
	}

	for _, c := range cases {
		virtualKey, ok := nameToVirtualKey[c.name]
		if !ok {
			t.Errorf("key %q missing from nameToVirtualKey", c.name)
			continue
		}
		if virtualKey != c.virtualKey {
			t.Errorf("key %q: expected virtual key %x, got %x", c.name, c.virtualKey, virtualKey)
		}
	}
}
