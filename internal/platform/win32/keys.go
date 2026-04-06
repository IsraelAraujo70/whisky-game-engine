package win32

// keyBinding associates a Win32 virtual-key code with an engine control name.
type keyBinding struct {
	virtualKey uint16
	control    string
}

// nameToVirtualKey maps human-readable key names used by whisky.Config.KeyMap
// to Win32 virtual-key codes.
var nameToVirtualKey = map[string]uint16{
	// Letters
	"a": 0x41, "b": 0x42, "c": 0x43, "d": 0x44, "e": 0x45, "f": 0x46,
	"g": 0x47, "h": 0x48, "i": 0x49, "j": 0x4A, "k": 0x4B, "l": 0x4C,
	"m": 0x4D, "n": 0x4E, "o": 0x4F, "p": 0x50, "q": 0x51, "r": 0x52,
	"s": 0x53, "t": 0x54, "u": 0x55, "v": 0x56, "w": 0x57, "x": 0x58,
	"y": 0x59, "z": 0x5A,
	// Digits
	"0": 0x30, "1": 0x31, "2": 0x32, "3": 0x33, "4": 0x34,
	"5": 0x35, "6": 0x36, "7": 0x37, "8": 0x38, "9": 0x39,
	// Arrow keys
	"up":    0x26,
	"down":  0x28,
	"left":  0x25,
	"right": 0x27,
	// Named keys
	"space":     0x20,
	"enter":     0x0D,
	"escape":    0x1B,
	"backspace": 0x08,
	"tab":       0x09,
	"lshift":    0xA0,
	"rshift":    0xA1,
	"lctrl":     0xA2,
	"rctrl":     0xA3,
	"lalt":      0xA4,
	"ralt":      0xA5,
	// Function keys
	"f1":  0x70,
	"f2":  0x71,
	"f3":  0x72,
	"f4":  0x73,
	"f5":  0x74,
	"f6":  0x75,
	"f7":  0x76,
	"f8":  0x77,
	"f9":  0x78,
	"f10": 0x79,
	"f11": 0x7A,
	"f12": 0x7B,
}

func buildKeyBindings(km map[string]string) []keyBinding {
	bindings := make([]keyBinding, 0, len(km))
	for keyName, control := range km {
		if vk, ok := nameToVirtualKey[keyName]; ok {
			bindings = append(bindings, keyBinding{virtualKey: vk, control: control})
		}
	}
	return bindings
}
