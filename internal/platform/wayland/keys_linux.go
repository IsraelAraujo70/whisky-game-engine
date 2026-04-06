//go:build linux && (amd64 || arm64)

package wayland

type keyBinding struct {
	keycode uint32
	control string
}

// Wayland keyboard events expose Linux evdev keycodes.
var nameToEvdevKeycode = map[string]uint32{
	// Letters
	"a": 30, "b": 48, "c": 46, "d": 32, "e": 18, "f": 33,
	"g": 34, "h": 35, "i": 23, "j": 36, "k": 37, "l": 38,
	"m": 50, "n": 49, "o": 24, "p": 25, "q": 16, "r": 19,
	"s": 31, "t": 20, "u": 22, "v": 47, "w": 17, "x": 45,
	"y": 21, "z": 44,
	// Digits
	"0": 11, "1": 2, "2": 3, "3": 4, "4": 5,
	"5": 6, "6": 7, "7": 8, "8": 9, "9": 10,
	// Arrows
	"up":    103,
	"down":  108,
	"left":  105,
	"right": 106,
	// Named keys
	"space":     57,
	"enter":     28,
	"escape":    1,
	"backspace": 14,
	"tab":       15,
	"lshift":    42,
	"rshift":    54,
	"lctrl":     29,
	"rctrl":     97,
	"lalt":      56,
	"ralt":      100,
	// Function keys
	"f1":  59,
	"f2":  60,
	"f3":  61,
	"f4":  62,
	"f5":  63,
	"f6":  64,
	"f7":  65,
	"f8":  66,
	"f9":  67,
	"f10": 68,
	"f11": 87,
	"f12": 88,
}

func buildKeyBindings(keyMap map[string]string) []keyBinding {
	bindings := make([]keyBinding, 0, len(keyMap))
	for keyName, control := range keyMap {
		keycode, ok := nameToEvdevKeycode[keyName]
		if !ok {
			continue
		}
		bindings = append(bindings, keyBinding{keycode: keycode, control: control})
	}
	return bindings
}
