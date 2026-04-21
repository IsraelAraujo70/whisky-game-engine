//go:build windows

package win32

import (
	"syscall"
	"unsafe"

	"github.com/IsraelAraujo70/whisky-game-engine/input"
)

// XInput constants
const (
	xinputGamepadDPadUp        = 0x0001
	xinputGamepadDPadDown      = 0x0002
	xinputGamepadDPadLeft      = 0x0004
	xinputGamepadDPadRight     = 0x0008
	xinputGamepadStart         = 0x0010
	xinputGamepadBack          = 0x0020
	xinputGamepadLeftThumb     = 0x0040
	xinputGamepadRightThumb    = 0x0080
	xinputGamepadLeftShoulder  = 0x0100
	xinputGamepadRightShoulder = 0x0200
	xinputGamepadA             = 0x1000
	xinputGamepadB             = 0x2000
	xinputGamepadX             = 0x4000
	xinputGamepadY             = 0x8000

	xinputMaxControllerCount = 4
	thumbstickDeadzone       = 7849   // XINPUT_GAMEPAD_LEFT_THUMB_DEADZONE default
	triggerThreshold         = 30     // XINPUT_GAMEPAD_TRIGGER_THRESHOLD default
)

// xinputState mirrors XINPUT_STATE
type xinputState struct {
	PacketNumber uint32
	Gamepad      xinputGamepad
}

// xinputGamepad mirrors XINPUT_GAMEPAD
type xinputGamepad struct {
	Buttons      uint16
	LeftTrigger  uint8
	RightTrigger uint8
	ThumbLX      int16
	ThumbLY      int16
	ThumbRX      int16
	ThumbRY      int16
}

var (
	xinputDLL      *syscall.LazyDLL
	xinputGetState *syscall.LazyProc
	xinputLoaded   bool
)

func init() {
	// Prefer xinput1_4.dll (Windows 8+), fallback to xinput1_3.dll (Windows 7)
	for _, name := range []string{"xinput1_4.dll", "xinput1_3.dll"} {
		dll := syscall.NewLazyDLL(name)
		if err := dll.Load(); err == nil {
			xinputDLL = dll
			xinputGetState = dll.NewProc("XInputGetState")
			xinputLoaded = true
			break
		}
	}
}

func pollXInput(state *input.State) {
	if !xinputLoaded {
		return
	}

	for i := 0; i < xinputMaxControllerCount && i < input.MaxGamepads; i++ {
		pad := state.Gamepad(i)
		var st xinputState
		ret, _, _ := xinputGetState.Call(uintptr(i), uintptr(unsafe.Pointer(&st)))
		if ret != 0 {
			// ERROR_DEVICE_NOT_CONNECTED or other error
			if pad.Connected() {
				pad.SetConnected(false)
			}
			continue
		}

		if !pad.Connected() {
			pad.SetConnected(true)
		}

		gp := st.Gamepad
		buttons := gp.Buttons

		pad.SetButton(input.GamepadButtonA, buttons&xinputGamepadA != 0)
		pad.SetButton(input.GamepadButtonB, buttons&xinputGamepadB != 0)
		pad.SetButton(input.GamepadButtonX, buttons&xinputGamepadX != 0)
		pad.SetButton(input.GamepadButtonY, buttons&xinputGamepadY != 0)
		pad.SetButton(input.GamepadButtonBack, buttons&xinputGamepadBack != 0)
		pad.SetButton(input.GamepadButtonStart, buttons&xinputGamepadStart != 0)
		pad.SetButton(input.GamepadButtonGuide, false) // XInput does not expose Guide
		pad.SetButton(input.GamepadButtonLeftStick, buttons&xinputGamepadLeftThumb != 0)
		pad.SetButton(input.GamepadButtonRightStick, buttons&xinputGamepadRightThumb != 0)
		pad.SetButton(input.GamepadButtonLB, buttons&xinputGamepadLeftShoulder != 0)
		pad.SetButton(input.GamepadButtonRB, buttons&xinputGamepadRightShoulder != 0)
		pad.SetButton(input.GamepadButtonDPadUp, buttons&xinputGamepadDPadUp != 0)
		pad.SetButton(input.GamepadButtonDPadDown, buttons&xinputGamepadDPadDown != 0)
		pad.SetButton(input.GamepadButtonDPadLeft, buttons&xinputGamepadDPadLeft != 0)
		pad.SetButton(input.GamepadButtonDPadRight, buttons&xinputGamepadDPadRight != 0)

		pad.SetAxis(input.GamepadAxisLX, normalizeThumbstick(float64(gp.ThumbLX), thumbstickDeadzone))
		pad.SetAxis(input.GamepadAxisLY, -normalizeThumbstick(float64(gp.ThumbLY), thumbstickDeadzone))
		pad.SetAxis(input.GamepadAxisRX, normalizeThumbstick(float64(gp.ThumbRX), thumbstickDeadzone))
		pad.SetAxis(input.GamepadAxisRY, -normalizeThumbstick(float64(gp.ThumbRY), thumbstickDeadzone))
		pad.SetAxis(input.GamepadAxisLT, normalizeTrigger(float64(gp.LeftTrigger), triggerThreshold))
		pad.SetAxis(input.GamepadAxisRT, normalizeTrigger(float64(gp.RightTrigger), triggerThreshold))
	}
}

func normalizeThumbstick(raw float64, deadzone float64) float64 {
	if raw > 0 {
		raw -= deadzone
		if raw < 0 {
			raw = 0
		}
	} else if raw < 0 {
		raw += deadzone
		if raw > 0 {
			raw = 0
		}
	}
	// After deadzone removal, range is roughly [-32768+deadzone, 32767-deadzone]
	maxVal := 32767.0 - deadzone
	if maxVal <= 0 {
		return 0
	}
	v := raw / maxVal
	if v > 1 {
		v = 1
	} else if v < -1 {
		v = -1
	}
	return v
}

func normalizeTrigger(raw float64, threshold float64) float64 {
	if raw < threshold {
		return 0
	}
	v := (raw - threshold) / (255.0 - threshold)
	if v > 1 {
		v = 1
	}
	return v
}
