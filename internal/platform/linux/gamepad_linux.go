//go:build linux && (amd64 || arm64)

// Package linux provides shared Linux platform helpers (evdev gamepad polling)
// used by both X11 and Wayland backends.
package linux

import (
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/IsraelAraujo70/whisky-game-engine/input"
)

// evdev constants
const (
	evSyn = 0x00
	evKey = 0x01
	evAbs = 0x03

	btnGamepad  = 0x130 // BTN_SOUTH / BTN_A
	btnB        = 0x131 // BTN_EAST / BTN_B
	btnX        = 0x133 // BTN_NORTH / BTN_X
	btnY        = 0x134 // BTN_WEST / BTN_Y
	btnTL       = 0x136 // BTN_TL / LB
	btnTR       = 0x137 // BTN_TR / RB
	btnTL2      = 0x138 // BTN_TL2 / LT digital
	btnTR2      = 0x139 // BTN_TR2 / RT digital
	btnSelect   = 0x13a // BTN_SELECT / BACK
	btnStart    = 0x13b // BTN_START
	btnMode     = 0x13c // BTN_MODE / GUIDE
	btnThumbl   = 0x13d // BTN_THUMBL
	btnThumbr   = 0x13e // BTN_THUMBR

	btnDPadUp    = 0x220 // BTN_DPAD_UP
	btnDPadDown  = 0x221 // BTN_DPAD_DOWN
	btnDPadLeft  = 0x222 // BTN_DPAD_LEFT
	btnDPadRight = 0x223 // BTN_DPAD_RIGHT

	absX     = 0x00
	absY     = 0x01
	absZ     = 0x02
	absRx    = 0x03
	absRy    = 0x04
	absRz    = 0x05
	absHat0X = 0x10
	absHat0Y = 0x11

	ioctlEVIOCGBIT = 0x20 // EVIOCGBIT base; actual computed per macro
	ioctlEVIOCGABS = 0x40 // EVIOCGABS base
	ioctlEVIOCGID  = 0x45 // EVIOCGID

	thumbstickDeadzone = 8192

	keyMax = 0x2ff // 767
	absMax = 0x3f  // 63
)

// inputEvent mirrors struct input_event from linux/input.h.
type inputEvent struct {
	Time  unix.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

// inputAbsinfo mirrors struct input_absinfo from linux/input.h.
type inputAbsinfo struct {
	Value      int32
	Minimum    int32
	Maximum    int32
	Fuzz       int32
	Flat       int32
	Resolution int32
}

const (
	sizeofInputEvent   = int(unsafe.Sizeof(inputEvent{}))
	sizeofInputAbsinfo = int(unsafe.Sizeof(inputAbsinfo{}))
)

// GamepadPoller manages evdev gamepad devices and feeds state into input.State.
type GamepadPoller struct {
	mu      sync.Mutex
	devices []evdevDevice
	lastScan time.Time
}

type evdevDevice struct {
	fd       int
	path     string
	buttons  map[uint16]bool // kernel button code -> pressed
	axes     map[uint16]int32 // kernel axis code -> raw value
	absInfo  map[uint16]*inputAbsinfo
	assigned int // assigned gamepad slot (-1 if none)
}

// NewGamepadPoller creates a poller. Devices are discovered lazily.
func NewGamepadPoller() *GamepadPoller {
	return &GamepadPoller{
		lastScan: time.Now().Add(-time.Hour), // force scan on first use
	}
}

// Poll reads all discovered evdev gamepads and updates state.
func (p *GamepadPoller) Poll(state *input.State) {
	p.maybeRescan()

	p.mu.Lock()
	defer p.mu.Unlock()

	for i := range p.devices {
		dev := &p.devices[i]
		if dev.fd < 0 {
			continue
		}
		// Drain pending events
		for {
			var ev inputEvent
			n, err := unix.Read(dev.fd, (*[sizeofInputEvent]byte)(unsafe.Pointer(&ev))[:])
			if err != nil || n != sizeofInputEvent {
				break
			}
			p.handleEvent(dev, &ev)
		}

		// Assign slot if not yet assigned
		if dev.assigned < 0 {
			dev.assigned = p.findFreeSlot(state)
		}
		if dev.assigned >= 0 {
			p.syncToState(dev, state.Gamepad(dev.assigned))
		}
	}
}

func (p *GamepadPoller) maybeRescan() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if time.Since(p.lastScan) < 5*time.Second {
		return
	}
	p.lastScan = time.Now()

	matches, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		return
	}

	existing := make(map[string]bool, len(p.devices))
	for _, dev := range p.devices {
		existing[dev.path] = true
	}

	for _, path := range matches {
		if existing[path] {
			continue
		}
		dev, ok := p.tryOpenDevice(path)
		if ok {
			p.devices = append(p.devices, dev)
		}
	}

	// Remove disconnected devices
	alive := p.devices[:0]
	for _, dev := range p.devices {
		if _, err := unix.FcntlInt(uintptr(dev.fd), unix.F_GETFD, 0); err != nil {
			if dev.fd >= 0 {
				_ = unix.Close(dev.fd)
			}
			if dev.assigned >= 0 {
				// Mark disconnected in state
			}
			continue
		}
		alive = append(alive, dev)
	}
	p.devices = alive
}

func (p *GamepadPoller) tryOpenDevice(path string) (evdevDevice, bool) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return evdevDevice{}, false
	}

	// Check if device has gamepad buttons using EVIOCGBIT(EV_KEY)
	var keyBits [(keyMax + 1) / 8]byte
	if err := ioctl(fd, _IOC(_IOC_READ, 'E', 0x20, uint(len(keyBits))), uintptr(unsafe.Pointer(&keyBits[0]))); err != nil {
		_ = unix.Close(fd)
		return evdevDevice{}, false
	}

	// Must have at least BTN_A / BTN_GAMEPAD
	if !testBit(keyBits[:], btnGamepad) {
		_ = unix.Close(fd)
		return evdevDevice{}, false
	}

	dev := evdevDevice{
		fd:      fd,
		path:    path,
		buttons: make(map[uint16]bool),
		axes:    make(map[uint16]int32),
		absInfo: make(map[uint16]*inputAbsinfo),
		assigned: -1,
	}

	// Query absolute axis info
	var absBits[(absMax + 1) / 8]byte
	if err := ioctl(fd, _IOC(_IOC_READ, 'E', 0x20, uint(len(absBits))), uintptr(unsafe.Pointer(&absBits[0]))); err == nil {
		for code := uint16(0); code < absMax; code++ {
			if testBit(absBits[:], code) {
				var info inputAbsinfo
				if err := ioctl(fd, _IOC(_IOC_READ, 'E', uint(0x40)+uint(code), uint(sizeofInputAbsinfo)), uintptr(unsafe.Pointer(&info))); err == nil {
					dev.absInfo[code] = &info
				}
			}
		}
	}

	return dev, true
}

func (p *GamepadPoller) handleEvent(dev *evdevDevice, ev *inputEvent) {
	switch ev.Type {
	case evKey:
		dev.buttons[ev.Code] = ev.Value != 0
	case evAbs:
		dev.axes[ev.Code] = ev.Value
	case evSyn:
		// sync: nothing to do, state already accumulated
	}
}

func (p *GamepadPoller) findFreeSlot(state *input.State) int {
	for i := 0; i < input.MaxGamepads; i++ {
		if !state.Gamepad(i).Connected() {
			return i
		}
	}
	return -1
}

func (p *GamepadPoller) syncToState(dev *evdevDevice, pad *input.GamepadState) {
	if pad == nil {
		return
	}
	pad.SetConnected(true)

	// Buttons
	pad.SetButton(input.GamepadButtonA, dev.buttons[btnGamepad])
	pad.SetButton(input.GamepadButtonB, dev.buttons[btnB])
	pad.SetButton(input.GamepadButtonX, dev.buttons[btnX])
	pad.SetButton(input.GamepadButtonY, dev.buttons[btnY])
	pad.SetButton(input.GamepadButtonLB, dev.buttons[btnTL])
	pad.SetButton(input.GamepadButtonRB, dev.buttons[btnTR])
	pad.SetButton(input.GamepadButtonBack, dev.buttons[btnSelect])
	pad.SetButton(input.GamepadButtonStart, dev.buttons[btnStart])
	pad.SetButton(input.GamepadButtonGuide, dev.buttons[btnMode])
	pad.SetButton(input.GamepadButtonLeftStick, dev.buttons[btnThumbl])
	pad.SetButton(input.GamepadButtonRightStick, dev.buttons[btnThumbr])

	// D-Pad: prefer digital buttons, fallback to hat
	dpadUp := dev.buttons[btnDPadUp]
	dpadDown := dev.buttons[btnDPadDown]
	dpadLeft := dev.buttons[btnDPadLeft]
	dpadRight := dev.buttons[btnDPadRight]
	if hatX, ok := dev.axes[absHat0X]; ok {
		if !dpadLeft && !dpadRight {
			dpadLeft = hatX < 0
			dpadRight = hatX > 0
		}
	}
	if hatY, ok := dev.axes[absHat0Y]; ok {
		if !dpadUp && !dpadDown {
			dpadUp = hatY < 0
			dpadDown = hatY > 0
		}
	}
	pad.SetButton(input.GamepadButtonDPadUp, dpadUp)
	pad.SetButton(input.GamepadButtonDPadDown, dpadDown)
	pad.SetButton(input.GamepadButtonDPadLeft, dpadLeft)
	pad.SetButton(input.GamepadButtonDPadRight, dpadRight)

	// Axes
	pad.SetAxis(input.GamepadAxisLX, normalizeAxis(dev, absX, thumbstickDeadzone))
	pad.SetAxis(input.GamepadAxisLY, -normalizeAxis(dev, absY, thumbstickDeadzone))
	pad.SetAxis(input.GamepadAxisRX, normalizeAxis(dev, absRx, thumbstickDeadzone))
	pad.SetAxis(input.GamepadAxisRY, -normalizeAxis(dev, absRy, thumbstickDeadzone))
	pad.SetAxis(input.GamepadAxisLT, normalizeTrigger(dev, absZ))
	pad.SetAxis(input.GamepadAxisRT, normalizeTrigger(dev, absRz))
}

func normalizeAxis(dev *evdevDevice, code uint16, deadzone int32) float64 {
	info, ok := dev.absInfo[code]
	if !ok {
		return 0
	}
	raw := dev.axes[code]
	if raw > deadzone {
		raw -= deadzone
	} else if raw < -deadzone {
		raw += deadzone
	} else {
		raw = 0
	}
	maxVal := info.Maximum - deadzone
	minVal := info.Minimum + deadzone
	if raw >= 0 {
		if maxVal <= 0 {
			return 0
		}
		v := float64(raw) / float64(maxVal)
		if v > 1 {
			v = 1
		}
		return v
	}
	if minVal >= 0 {
		return 0
	}
	v := float64(raw) / float64(minVal)
	if v < -1 {
		v = -1
	}
	return v
}

func normalizeTrigger(dev *evdevDevice, code uint16) float64 {
	info, ok := dev.absInfo[code]
	if !ok {
		return 0
	}
	raw := dev.axes[code]
	minV := info.Minimum
	maxV := info.Maximum
	if maxV <= minV {
		return 0
	}
	v := float64(raw-minV) / float64(maxV-minV)
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	return v
}

func testBit(bits []byte, bit uint16) bool {
	idx := bit / 8
	if int(idx) >= len(bits) {
		return false
	}
	return bits[idx]&(1<<(bit%8)) != 0
}

func ioctl(fd int, req uint, arg uintptr) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(req), arg)
	if errno != 0 {
		return errno
	}
	return nil
}

// _IOC builds an ioctl request code.
func _IOC(dir, t, nr, size uint) uint {
	return (dir << 30) | (size << 16) | (t << 8) | nr
}

const _IOC_READ = 2


