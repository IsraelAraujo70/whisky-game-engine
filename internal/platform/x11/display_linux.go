//go:build linux && (amd64 || arm64)

package x11

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"

	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

// Ensure Window implements DisplayController.
var _ platformapi.DisplayController = (*Window)(nil)

// Additional X11 functions for display control.
var (
	xrandrOnce sync.Once
	xrandrErr  error

	xrandrHandle uintptr

	xResizeWindow     func(display unsafe.Pointer, window uintptr, width, height uint32) int32
	xMoveWindow       func(display unsafe.Pointer, window uintptr, x, y int32) int32
	xSendEvent        func(display unsafe.Pointer, window uintptr, propagate int32, eventMask int, event unsafe.Pointer) int32
	xChangeProperty   func(display unsafe.Pointer, window uintptr, property, propType uintptr, format int32, mode int32, data unsafe.Pointer, nelements int32) int32

	// XRandR functions (optional, for monitor enumeration)
	xrrGetScreenResourcesCurrent func(display unsafe.Pointer, window uintptr) unsafe.Pointer
	xrrFreeScreenResources       func(resources unsafe.Pointer)
	xrrGetOutputInfo             func(display unsafe.Pointer, resources unsafe.Pointer, output uintptr) unsafe.Pointer
	xrrFreeOutputInfo            func(info unsafe.Pointer)
	xrrGetCrtcInfo               func(display unsafe.Pointer, resources unsafe.Pointer, crtc uintptr) unsafe.Pointer
	xrrFreeCrtcInfo              func(info unsafe.Pointer)
)

// xrrScreenResources mirrors parts of XRRScreenResources.
type xrrScreenResources struct {
	Timestamp       uintptr
	ConfigTimestamp uintptr
	NCrtc           int32
	_               [4]byte
	Crtcs           unsafe.Pointer
	NOutput         int32
	_               [4]byte
	Outputs         unsafe.Pointer
	NMode           int32
	_               [4]byte
	Modes           unsafe.Pointer
}

// xrrOutputInfo mirrors parts of XRROutputInfo.
type xrrOutputInfo struct {
	Timestamp  uintptr
	Crtc       uintptr
	Name       *byte
	NameLen    int32
	_          [4]byte
	MmWidth    uintptr
	MmHeight   uintptr
	Connection uint16
	_          [6]byte
	// Followed by more fields we don't need
}

// xrrCrtcInfo mirrors parts of XRRCrtcInfo.
type xrrCrtcInfo struct {
	Timestamp uintptr
	X         int32
	Y         int32
	Width     uint32
	Height    uint32
	Mode      uintptr
	Rotation  uint16
	_         [6]byte
	NOutput   int32
	_         [4]byte
	Outputs   unsafe.Pointer
}

// xClientMessageEventRaw is for constructing XClientMessageEvent for XSendEvent.
type xClientMessageEventRaw struct {
	Type        int32
	_           [4]byte
	Serial      uintptr
	SendEvent   int32
	_           [4]byte
	Display     unsafe.Pointer
	Window      uintptr
	MessageType uintptr
	Format      int32
	_           [4]byte
	Data        [5]uintptr
}

const (
	substructureRedirectMask = 1 << 20
	substructureNotifyMask   = 1 << 19
	propModeReplace          = 0

	netWMStateRemove = 0
	netWMStateAdd    = 1
)

func ensureDisplayFunctions() {
	// These functions are from libX11, loaded in ensureXlib.
	// We register them as additional functions from the same library.
	if xlibHandle == 0 {
		return
	}
	purego.RegisterLibFunc(&xResizeWindow, xlibHandle, "XResizeWindow")
	purego.RegisterLibFunc(&xMoveWindow, xlibHandle, "XMoveWindow")
	purego.RegisterLibFunc(&xSendEvent, xlibHandle, "XSendEvent")
	purego.RegisterLibFunc(&xChangeProperty, xlibHandle, "XChangeProperty")
}

var displayFuncsOnce sync.Once

func ensureXrandr() error {
	xrandrOnce.Do(func() {
		var err error
		xrandrHandle, err = purego.Dlopen("libXrandr.so.2", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			xrandrErr = fmt.Errorf("xrandr: %v", err)
			return
		}

		purego.RegisterLibFunc(&xrrGetScreenResourcesCurrent, xrandrHandle, "XRRGetScreenResourcesCurrent")
		purego.RegisterLibFunc(&xrrFreeScreenResources, xrandrHandle, "XRRFreeScreenResources")
		purego.RegisterLibFunc(&xrrGetOutputInfo, xrandrHandle, "XRRGetOutputInfo")
		purego.RegisterLibFunc(&xrrFreeOutputInfo, xrandrHandle, "XRRFreeOutputInfo")
		purego.RegisterLibFunc(&xrrGetCrtcInfo, xrandrHandle, "XRRGetCrtcInfo")
		purego.RegisterLibFunc(&xrrFreeCrtcInfo, xrandrHandle, "XRRFreeCrtcInfo")
	})
	return xrandrErr
}

// SetWindowSize resizes the window.
func (w *Window) SetWindowSize(width, height int) error {
	if w == nil || w.display == nil || w.window == 0 {
		return platformapi.ErrNotSupported
	}
	displayFuncsOnce.Do(ensureDisplayFunctions)
	if xResizeWindow == nil {
		return platformapi.ErrNotSupported
	}

	xResizeWindow(w.display, w.window, uint32(width), uint32(height))
	xFlush(w.display)
	w.width.Store(int32(width))
	w.height.Store(int32(height))
	return nil
}

// SetWindowMode sets the window to windowed, borderless fullscreen, or fullscreen.
func (w *Window) SetWindowMode(mode platformapi.WindowMode) error {
	if w == nil || w.display == nil || w.window == 0 {
		return platformapi.ErrNotSupported
	}
	displayFuncsOnce.Do(ensureDisplayFunctions)
	if xSendEvent == nil {
		return platformapi.ErrNotSupported
	}

	screen := xDefaultScreen(w.display)
	root := xRootWindow(w.display, screen)

	// Intern the atoms we need.
	wmStateName := []byte("_NET_WM_STATE\x00")
	wmStateFullscreenName := []byte("_NET_WM_STATE_FULLSCREEN\x00")

	wmState := xInternAtom(w.display, &wmStateName[0], 0)
	wmStateFullscreen := xInternAtom(w.display, &wmStateFullscreenName[0], 0)

	if wmState == 0 || wmStateFullscreen == 0 {
		return platformapi.ErrNotSupported
	}

	var action uintptr
	switch mode {
	case platformapi.WindowModeWindowed:
		action = netWMStateRemove
	case platformapi.WindowModeBorderless, platformapi.WindowModeFullscreen:
		action = netWMStateAdd
	default:
		return platformapi.ErrNotSupported
	}

	// Send _NET_WM_STATE client message to root window.
	var event xClientMessageEventRaw
	event.Type = eventTypeClientMessage
	event.Window = w.window
	event.MessageType = wmState
	event.Format = 32
	event.Data[0] = action
	event.Data[1] = wmStateFullscreen
	event.Data[2] = 0 // no secondary property
	event.Data[3] = 1 // source indication: normal application

	xSendEvent(w.display, root, 0,
		substructureRedirectMask|substructureNotifyMask,
		unsafe.Pointer(&event))
	xFlush(w.display)
	return nil
}

// Monitors returns the list of connected monitors using XRandR.
func (w *Window) Monitors() ([]platformapi.MonitorInfo, error) {
	if w == nil || w.display == nil || w.window == 0 {
		return nil, platformapi.ErrNotSupported
	}

	if err := ensureXrandr(); err != nil {
		// If XRandR is not available, return a single default monitor.
		return []platformapi.MonitorInfo{
			{
				Name:      "Default",
				Index:     0,
				IsPrimary: true,
				Modes: []platformapi.DisplayMode{
					{Width: 1280, Height: 720, RefreshHz: 60},
					{Width: 1600, Height: 900, RefreshHz: 60},
					{Width: 1920, Height: 1080, RefreshHz: 60},
					{Width: 2560, Height: 1440, RefreshHz: 60},
				},
			},
		}, nil
	}

	screen := xDefaultScreen(w.display)
	root := xRootWindow(w.display, screen)

	resources := xrrGetScreenResourcesCurrent(w.display, root)
	if resources == nil {
		return nil, fmt.Errorf("XRRGetScreenResourcesCurrent failed")
	}
	defer xrrFreeScreenResources(resources)

	res := (*xrrScreenResources)(resources)
	if res.NOutput <= 0 {
		return nil, nil
	}

	outputs := unsafe.Slice((*uintptr)(res.Outputs), res.NOutput)
	var monitors []platformapi.MonitorInfo

	for i, output := range outputs {
		info := xrrGetOutputInfo(w.display, resources, output)
		if info == nil {
			continue
		}
		outInfo := (*xrrOutputInfo)(info)

		// Connection status: 0 = connected
		if outInfo.Connection != 0 {
			xrrFreeOutputInfo(info)
			continue
		}

		name := "Monitor"
		if outInfo.Name != nil && outInfo.NameLen > 0 {
			name = string(unsafe.Slice(outInfo.Name, outInfo.NameLen))
		}

		monitor := platformapi.MonitorInfo{
			Name:      name,
			Index:     len(monitors),
			IsPrimary: len(monitors) == 0, // First connected output is primary
		}

		// Get current mode from CRTC.
		if outInfo.Crtc != 0 {
			crtcInfo := xrrGetCrtcInfo(w.display, resources, outInfo.Crtc)
			if crtcInfo != nil {
				crtc := (*xrrCrtcInfo)(crtcInfo)
				if crtc.Width > 0 && crtc.Height > 0 {
					monitor.Modes = append(monitor.Modes, platformapi.DisplayMode{
						Width:     int(crtc.Width),
						Height:    int(crtc.Height),
						RefreshHz: 60, // approximate
					})
				}
				xrrFreeCrtcInfo(crtcInfo)
			}
		}

		// Add common resolutions as options if they fit.
		commonModes := []platformapi.DisplayMode{
			{Width: 1280, Height: 720, RefreshHz: 60},
			{Width: 1600, Height: 900, RefreshHz: 60},
			{Width: 1920, Height: 1080, RefreshHz: 60},
			{Width: 2560, Height: 1440, RefreshHz: 60},
			{Width: 3840, Height: 2160, RefreshHz: 60},
		}
		for _, mode := range commonModes {
			if !hasModeResolution(monitor.Modes, mode.Width, mode.Height) {
				monitor.Modes = append(monitor.Modes, mode)
			}
		}

		monitors = append(monitors, monitor)
		xrrFreeOutputInfo(info)

		if i >= 7 { // reasonable limit
			break
		}
	}

	if len(monitors) == 0 {
		monitors = append(monitors, platformapi.MonitorInfo{
			Name:      "Default",
			Index:     0,
			IsPrimary: true,
			Modes: []platformapi.DisplayMode{
				{Width: 1280, Height: 720, RefreshHz: 60},
				{Width: 1920, Height: 1080, RefreshHz: 60},
			},
		})
	}

	return monitors, nil
}

// MoveToMonitor moves the window to the specified monitor.
func (w *Window) MoveToMonitor(index int) error {
	if w == nil || w.display == nil || w.window == 0 {
		return platformapi.ErrNotSupported
	}
	displayFuncsOnce.Do(ensureDisplayFunctions)
	if xMoveWindow == nil {
		return platformapi.ErrNotSupported
	}

	if err := ensureXrandr(); err != nil {
		// Can't determine monitor positions without XRandR
		return platformapi.ErrNotSupported
	}

	screen := xDefaultScreen(w.display)
	root := xRootWindow(w.display, screen)
	resources := xrrGetScreenResourcesCurrent(w.display, root)
	if resources == nil {
		return platformapi.ErrNotSupported
	}
	defer xrrFreeScreenResources(resources)

	res := (*xrrScreenResources)(resources)
	if res.NOutput <= 0 {
		return platformapi.ErrNotSupported
	}

	outputs := unsafe.Slice((*uintptr)(res.Outputs), res.NOutput)

	// Find the nth connected output.
	connectedIdx := 0
	for _, output := range outputs {
		info := xrrGetOutputInfo(w.display, resources, output)
		if info == nil {
			continue
		}
		outInfo := (*xrrOutputInfo)(info)
		if outInfo.Connection != 0 {
			xrrFreeOutputInfo(info)
			continue
		}

		if connectedIdx == index {
			if outInfo.Crtc != 0 {
				crtcInfo := xrrGetCrtcInfo(w.display, resources, outInfo.Crtc)
				if crtcInfo != nil {
					crtc := (*xrrCrtcInfo)(crtcInfo)
					xMoveWindow(w.display, w.window, crtc.X, crtc.Y)
					xFlush(w.display)
					xrrFreeCrtcInfo(crtcInfo)
				}
			}
			xrrFreeOutputInfo(info)
			return nil
		}

		connectedIdx++
		xrrFreeOutputInfo(info)
	}

	return fmt.Errorf("monitor index %d not found", index)
}

func hasModeResolution(modes []platformapi.DisplayMode, w, h int) bool {
	for _, m := range modes {
		if m.Width == w && m.Height == h {
			return true
		}
	}
	return false
}
