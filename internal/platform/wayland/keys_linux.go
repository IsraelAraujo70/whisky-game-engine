//go:build linux && (amd64 || arm64)

package wayland

type keyBinding struct {
	keysym  uint32
	control string
}

// nameToKeysym maps human-readable key names to XKB keysyms.
// These values match the common X11 / xkbcommon definitions.
var nameToKeysym = map[string]uint32{
	// Letters
	"a": 0x0061, "b": 0x0062, "c": 0x0063, "d": 0x0064, "e": 0x0065, "f": 0x0066,
	"g": 0x0067, "h": 0x0068, "i": 0x0069, "j": 0x006A, "k": 0x006B, "l": 0x006C,
	"m": 0x006D, "n": 0x006E, "o": 0x006F, "p": 0x0070, "q": 0x0071, "r": 0x0072,
	"s": 0x0073, "t": 0x0074, "u": 0x0075, "v": 0x0076, "w": 0x0077, "x": 0x0078,
	"y": 0x0079, "z": 0x007A,
	// Digits
	"0": 0x0030, "1": 0x0031, "2": 0x0032, "3": 0x0033, "4": 0x0034,
	"5": 0x0035, "6": 0x0036, "7": 0x0037, "8": 0x0038, "9": 0x0039,
	// Arrow keys
	"up":    0xFF52,
	"down":  0xFF54,
	"left":  0xFF51,
	"right": 0xFF53,
	// Named keys
	"space":     0x0020,
	"enter":     0xFF0D,
	"escape":    0xFF1B,
	"backspace": 0xFF08,
	"tab":       0xFF09,
	"lshift":    0xFFE1,
	"rshift":    0xFFE2,
	"lctrl":     0xFFE3,
	"rctrl":     0xFFE4,
	"lalt":      0xFFE9,
	"ralt":      0xFFEA,
	// Function keys
	"f1":  0xFFBE,
	"f2":  0xFFBF,
	"f3":  0xFFC0,
	"f4":  0xFFC1,
	"f5":  0xFFC2,
	"f6":  0xFFC3,
	"f7":  0xFFC4,
	"f8":  0xFFC5,
	"f9":  0xFFC6,
	"f10": 0xFFC7,
	"f11": 0xFFC8,
	"f12": 0xFFC9,
}

func buildKeyBindings(keyMap map[string]string) []keyBinding {
	bindings := make([]keyBinding, 0, len(keyMap))
	for keyName, control := range keyMap {
		keysym, ok := nameToKeysym[keyName]
		if !ok {
			continue
		}
		bindings = append(bindings, keyBinding{keysym: keysym, control: control})
	}
	return bindings
}
