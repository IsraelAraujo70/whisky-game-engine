//go:build linux && (amd64 || arm64)

package x11

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/IsraelAraujo70/whisky-game-engine/input"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

var _ platformapi.NativeWindow = (*Window)(nil)

const (
	eventMaskKeyPress        = 1 << 0
	eventMaskKeyRelease      = 1 << 1
	eventMaskStructureNotify = 1 << 17

	eventTypeConfigureNotify = 22
	eventTypeClientMessage   = 33
)

var (
	xlibOnce sync.Once
	xlibErr  error

	xlibHandle uintptr

	xOpenDisplay        func(name *byte) unsafe.Pointer
	xCloseDisplay       func(display unsafe.Pointer) int32
	xDefaultScreen      func(display unsafe.Pointer) int32
	xRootWindow         func(display unsafe.Pointer, screenNumber int32) uintptr
	xCreateSimpleWindow func(display unsafe.Pointer, parent uintptr, x, y int32, width, height, borderWidth uint32, border, background uintptr) uintptr
	xStoreName          func(display unsafe.Pointer, window uintptr, name *byte) int32
	xInternAtom         func(display unsafe.Pointer, atomName *byte, onlyIfExists int32) uintptr
	xSetWMProtocols     func(display unsafe.Pointer, window uintptr, protocols *uintptr, count int32) int32
	xSelectInput        func(display unsafe.Pointer, window uintptr, eventMask int) int32
	xMapWindow          func(display unsafe.Pointer, window uintptr) int32
	xFlush              func(display unsafe.Pointer) int32
	xPending            func(display unsafe.Pointer) int32
	xNextEvent          func(display unsafe.Pointer, event unsafe.Pointer) int32
	xDestroyWindow      func(display unsafe.Pointer, window uintptr) int32
	xGetGeometry        func(display unsafe.Pointer, drawable uintptr, root *uintptr, x, y *int32, width, height, borderWidth, depth *uint32) int32
	xKeysymToKeycode    func(display unsafe.Pointer, keysym uintptr) byte
	xQueryKeymap        func(display unsafe.Pointer, keysReturn *byte) int32
)

var ErrUnavailable = errors.New("x11 platform backend requires libX11 at runtime")

type Window struct {
	display        unsafe.Pointer
	window         uintptr
	wmDeleteWindow uintptr
	keyBindings    []keyBinding
	closed         atomic.Bool
	width          atomic.Int32
	height         atomic.Int32
}

type xEvent struct {
	pad [24]uintptr
}

type xClientMessageEvent struct {
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

func New(title string, width, height int, keyMap map[string]string) (*Window, error) {
	if err := ensureXlib(); err != nil {
		return nil, err
	}

	display := xOpenDisplay(nil)
	if display == nil {
		return nil, errors.New("x11: XOpenDisplay failed")
	}

	screen := xDefaultScreen(display)
	root := xRootWindow(display, screen)
	window := xCreateSimpleWindow(display, root, 0, 0, uint32(width), uint32(height), 0, 0, 0)
	if window == 0 {
		_ = xCloseDisplay(display)
		return nil, errors.New("x11: XCreateSimpleWindow failed")
	}

	titleBuf := append([]byte(title), 0)
	xStoreName(display, window, &titleBuf[0])

	wmDeleteBuf := []byte("WM_DELETE_WINDOW\x00")
	wmDeleteWindow := xInternAtom(display, &wmDeleteBuf[0], 0)
	if wmDeleteWindow == 0 {
		xDestroyWindow(display, window)
		_ = xCloseDisplay(display)
		return nil, errors.New("x11: XInternAtom(WM_DELETE_WINDOW) failed")
	}

	protocols := []uintptr{wmDeleteWindow}
	if xSetWMProtocols(display, window, &protocols[0], int32(len(protocols))) == 0 {
		xDestroyWindow(display, window)
		_ = xCloseDisplay(display)
		return nil, errors.New("x11: XSetWMProtocols failed")
	}

	eventMask := eventMaskKeyPress | eventMaskKeyRelease | eventMaskStructureNotify
	xSelectInput(display, window, eventMask)
	xMapWindow(display, window)
	xFlush(display)

	win := &Window{
		display:        display,
		window:         window,
		wmDeleteWindow: wmDeleteWindow,
	}
	win.width.Store(int32(width))
	win.height.Store(int32(height))
	win.keyBindings = buildKeyBindings(keyMap, func(keysym uintptr) byte {
		return xKeysymToKeycode(display, keysym)
	})

	return win, nil
}

func (w *Window) UpdateInput(state *input.State) {
	if w == nil || w.display == nil || w.closed.Load() {
		return
	}

	for _, binding := range w.keyBindings {
		state.SetPressed(binding.control, false)
	}

	var pressed [32]byte
	xQueryKeymap(w.display, &pressed[0])

	for _, binding := range w.keyBindings {
		index := binding.keycode / 8
		mask := byte(1 << (binding.keycode % 8))
		if index < byte(len(pressed)) && pressed[index]&mask != 0 {
			state.SetPressed(binding.control, true)
		}
	}
}

func (w *Window) PumpEvents() bool {
	if w == nil || w.display == nil || w.closed.Load() {
		return true
	}

	for xPending(w.display) > 0 {
		var event xEvent
		xNextEvent(w.display, unsafe.Pointer(&event))

		switch event.eventType() {
		case eventTypeConfigureNotify:
			w.refreshSize()
		case eventTypeClientMessage:
			client := (*xClientMessageEvent)(unsafe.Pointer(&event))
			if client.MessageType == 0 || client.Data[0] == w.wmDeleteWindow {
				w.closed.Store(true)
				return true
			}
		}
	}

	return w.closed.Load()
}

func (w *Window) Size() (width, height int) {
	return int(w.width.Load()), int(w.height.Load())
}

func (w *Window) NativeHandle() platformapi.NativeWindowHandle {
	return platformapi.NativeWindowHandle{
		Kind:    platformapi.NativeWindowKindX11,
		Display: uintptr(w.display),
		Window:  w.window,
	}
}

func (w *Window) Destroy() error {
	if w == nil {
		return nil
	}

	if w.window != 0 && w.display != nil {
		xDestroyWindow(w.display, w.window)
		xFlush(w.display)
		w.window = 0
	}
	if w.display != nil {
		_ = xCloseDisplay(w.display)
		w.display = nil
	}
	w.closed.Store(true)
	return nil
}

func (w *Window) refreshSize() {
	if w == nil || w.display == nil || w.window == 0 {
		return
	}

	var (
		root        uintptr
		x           int32
		y           int32
		width       uint32
		height      uint32
		borderWidth uint32
		depth       uint32
	)
	if xGetGeometry(w.display, w.window, &root, &x, &y, &width, &height, &borderWidth, &depth) == 0 {
		return
	}
	w.width.Store(int32(width))
	w.height.Store(int32(height))
}

func (e *xEvent) eventType() int32 {
	return *(*int32)(unsafe.Pointer(e))
}

func ensureXlib() error {
	xlibOnce.Do(func() {
		var err error
		xlibHandle, err = purego.Dlopen("libX11.so.6", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			xlibErr = fmt.Errorf("%w: %v", ErrUnavailable, err)
			return
		}

		purego.RegisterLibFunc(&xOpenDisplay, xlibHandle, "XOpenDisplay")
		purego.RegisterLibFunc(&xCloseDisplay, xlibHandle, "XCloseDisplay")
		purego.RegisterLibFunc(&xDefaultScreen, xlibHandle, "XDefaultScreen")
		purego.RegisterLibFunc(&xRootWindow, xlibHandle, "XRootWindow")
		purego.RegisterLibFunc(&xCreateSimpleWindow, xlibHandle, "XCreateSimpleWindow")
		purego.RegisterLibFunc(&xStoreName, xlibHandle, "XStoreName")
		purego.RegisterLibFunc(&xInternAtom, xlibHandle, "XInternAtom")
		purego.RegisterLibFunc(&xSetWMProtocols, xlibHandle, "XSetWMProtocols")
		purego.RegisterLibFunc(&xSelectInput, xlibHandle, "XSelectInput")
		purego.RegisterLibFunc(&xMapWindow, xlibHandle, "XMapWindow")
		purego.RegisterLibFunc(&xFlush, xlibHandle, "XFlush")
		purego.RegisterLibFunc(&xPending, xlibHandle, "XPending")
		purego.RegisterLibFunc(&xNextEvent, xlibHandle, "XNextEvent")
		purego.RegisterLibFunc(&xDestroyWindow, xlibHandle, "XDestroyWindow")
		purego.RegisterLibFunc(&xGetGeometry, xlibHandle, "XGetGeometry")
		purego.RegisterLibFunc(&xKeysymToKeycode, xlibHandle, "XKeysymToKeycode")
		purego.RegisterLibFunc(&xQueryKeymap, xlibHandle, "XQueryKeymap")
	})
	return xlibErr
}
