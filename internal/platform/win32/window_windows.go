//go:build windows

package win32

import (
	"fmt"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/IsraelAraujo70/whisky-game-engine/input"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

var _ platformapi.NativeWindow = (*Window)(nil)

const (
	csHRedraw          = 0x0002
	csVRedraw          = 0x0001
	csOwnDC            = 0x0020
	cwUseDefault       = 0x80000000
	pmRemove           = 0x0001
	swShowDefault      = 10
	wmDestroy          = 0x0002
	wmClose            = 0x0010
	wmQuit             = 0x0012
	wmSize             = 0x0005
	wmMouseMove        = 0x0200
	wmLButtonDown      = 0x0201
	wmLButtonUp        = 0x0202
	wmRButtonDown      = 0x0204
	wmRButtonUp        = 0x0205
	wmMButtonDown      = 0x0207
	wmMButtonUp        = 0x0208
	wmXButtonDown      = 0x020B
	wmXButtonUp        = 0x020C
	wmMouseWheel       = 0x020A
	wmMouseHWheel      = 0x020E
	wsOverlappedWindow = 0x00CF0000
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procAdjustWindowRectEx = user32.NewProc("AdjustWindowRectEx")
	procCreateWindowExW    = user32.NewProc("CreateWindowExW")
	procDefWindowProcW     = user32.NewProc("DefWindowProcW")
	procDestroyWindow      = user32.NewProc("DestroyWindow")
	procDispatchMessageW   = user32.NewProc("DispatchMessageW")
	procGetAsyncKeyState   = user32.NewProc("GetAsyncKeyState")
	procLoadCursorW        = user32.NewProc("LoadCursorW")
	procPeekMessageW       = user32.NewProc("PeekMessageW")
	procPostQuitMessage    = user32.NewProc("PostQuitMessage")
	procRegisterClassExW   = user32.NewProc("RegisterClassExW")
	procShowWindow         = user32.NewProc("ShowWindow")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procUnregisterClassW   = user32.NewProc("UnregisterClassW")
	procUpdateWindow       = user32.NewProc("UpdateWindow")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	windowClassOnce sync.Once
	windowClassErr  error
	windowClassName = windows.StringToUTF16Ptr("WhiskyWindowClass")
	wndProc         = syscall.NewCallback(windowProc)

	windowsMu       sync.RWMutex
	windowsByHandle = map[windows.Handle]*Window{}
)

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type point struct {
	X int32
	Y int32
}

type msg struct {
	HWnd    windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type Window struct {
	hwnd        windows.Handle
	instance    windows.Handle
	keyBindings []keyBinding
	closed      atomic.Bool
	width       atomic.Int32
	height      atomic.Int32

	mouseMu     sync.Mutex
	mouseX      float64
	mouseY      float64
	mouseButtons [5]bool
	mouseWheelX float64
	mouseWheelY float64
}

func New(title string, width, height int, keyMap map[string]string) (*Window, error) {
	instance, err := moduleHandle()
	if err != nil {
		return nil, err
	}
	if err := ensureWindowClass(instance); err != nil {
		return nil, err
	}

	clientRect := rect{Right: int32(width), Bottom: int32(height)}
	ok, _, callErr := procAdjustWindowRectEx.Call(
		uintptr(unsafe.Pointer(&clientRect)),
		uintptr(wsOverlappedWindow),
		0,
		0,
	)
	if ok == 0 {
		return nil, wrapCallErr("AdjustWindowRectEx", callErr)
	}

	titlePtr, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return nil, fmt.Errorf("win32: title conversion failed: %w", err)
	}

	hwnd, _, callErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(windowClassName)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(wsOverlappedWindow),
		uintptr(cwUseDefault),
		uintptr(cwUseDefault),
		uintptr(clientRect.Right-clientRect.Left),
		uintptr(clientRect.Bottom-clientRect.Top),
		0,
		0,
		uintptr(instance),
		0,
	)
	if hwnd == 0 {
		return nil, wrapCallErr("CreateWindowExW", callErr)
	}

	window := &Window{
		hwnd:        windows.Handle(hwnd),
		instance:    instance,
		keyBindings: buildKeyBindings(keyMap),
	}
	window.width.Store(int32(width))
	window.height.Store(int32(height))

	windowsMu.Lock()
	windowsByHandle[window.hwnd] = window
	windowsMu.Unlock()

	procShowWindow.Call(hwnd, uintptr(swShowDefault))
	procUpdateWindow.Call(hwnd)

	return window, nil
}

func (w *Window) UpdateInput(state *input.State) {
	for _, kb := range w.keyBindings {
		state.SetPressed(kb.control, false)
	}
	for _, kb := range w.keyBindings {
		if keyPressed(kb.virtualKey) {
			state.SetPressed(kb.control, true)
		}
	}

	w.mouseMu.Lock()
	mx, my := w.mouseX, w.mouseY
	mwx, mwy := w.mouseWheelX, w.mouseWheelY
	buttons := w.mouseButtons
	w.mouseWheelX = 0
	w.mouseWheelY = 0
	w.mouseMu.Unlock()

	mouse := state.Mouse()
	mouse.SetPosition(mx, my)
	mouse.SetButton(input.MouseButtonLeft, buttons[0])
	mouse.SetButton(input.MouseButtonRight, buttons[1])
	mouse.SetButton(input.MouseButtonMiddle, buttons[2])
	mouse.SetButton(input.MouseButtonX1, buttons[3])
	mouse.SetButton(input.MouseButtonX2, buttons[4])
	mouse.AddWheel(mwx, mwy)

	pollXInput(state)
}

func (w *Window) PumpEvents() bool {
	if w.closed.Load() {
		return true
	}

	var message msg
	for {
		ret, _, _ := procPeekMessageW.Call(
			uintptr(unsafe.Pointer(&message)),
			0,
			0,
			0,
			uintptr(pmRemove),
		)
		if ret == 0 {
			break
		}
		if message.Message == wmQuit {
			w.closed.Store(true)
			return true
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}

	return w.closed.Load()
}

func (w *Window) Size() (width, height int) {
	return int(w.width.Load()), int(w.height.Load())
}

func (w *Window) NativeHandle() platformapi.NativeWindowHandle {
	return platformapi.NativeWindowHandle{
		Kind:     platformapi.NativeWindowKindWin32,
		Window:   uintptr(w.hwnd),
		Instance: uintptr(w.instance),
	}
}

func (w *Window) Destroy() error {
	if w == nil {
		return nil
	}

	hwnd := w.hwnd
	if hwnd == 0 {
		w.closed.Store(true)
		return nil
	}

	windowsMu.Lock()
	delete(windowsByHandle, hwnd)
	windowsMu.Unlock()

	w.hwnd = 0
	w.closed.Store(true)

	ret, _, callErr := procDestroyWindow.Call(uintptr(hwnd))
	if ret == 0 {
		return wrapCallErr("DestroyWindow", callErr)
	}

	return nil
}

func ensureWindowClass(instance windows.Handle) error {
	windowClassOnce.Do(func() {
		cursor, _, _ := procLoadCursorW.Call(0, uintptr(32512))
		class := wndClassEx{
			CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
			Style:         csHRedraw | csVRedraw | csOwnDC,
			LpfnWndProc:   wndProc,
			HInstance:     instance,
			HCursor:       windows.Handle(cursor),
			LpszClassName: windowClassName,
		}
		ret, _, callErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&class)))
		if ret == 0 {
			windowClassErr = wrapCallErr("RegisterClassExW", callErr)
		}
	})
	return windowClassErr
}

func moduleHandle() (windows.Handle, error) {
	ret, _, callErr := procGetModuleHandleW.Call(0)
	if ret == 0 {
		return 0, wrapCallErr("GetModuleHandleW", callErr)
	}
	return windows.Handle(ret), nil
}

func keyPressed(vk uint16) bool {
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
	return ret&0x8000 != 0
}

func windowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	windowsMu.RLock()
	window := windowsByHandle[windows.Handle(hwnd)]
	windowsMu.RUnlock()

	switch msg {
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		if window != nil {
			window.hwnd = 0
			window.closed.Store(true)
			windowsMu.Lock()
			delete(windowsByHandle, windows.Handle(hwnd))
			windowsMu.Unlock()
		}
		procPostQuitMessage.Call(0)
		return 0
	case wmSize:
		if window != nil {
			window.width.Store(int32(lowWord(uint32(lParam))))
			window.height.Store(int32(highWord(uint32(lParam))))
		}
		return 0
	case wmMouseMove:
		if window != nil {
			x := float64(int16(lowWord(uint32(lParam))))
			y := float64(int16(highWord(uint32(lParam))))
			window.mouseMu.Lock()
			window.mouseX = x
			window.mouseY = y
			window.mouseMu.Unlock()
		}
		return 0
	case wmLButtonDown:
		if window != nil {
			window.mouseMu.Lock()
			window.mouseButtons[0] = true
			window.mouseMu.Unlock()
		}
		return 0
	case wmLButtonUp:
		if window != nil {
			window.mouseMu.Lock()
			window.mouseButtons[0] = false
			window.mouseMu.Unlock()
		}
		return 0
	case wmRButtonDown:
		if window != nil {
			window.mouseMu.Lock()
			window.mouseButtons[1] = true
			window.mouseMu.Unlock()
		}
		return 0
	case wmRButtonUp:
		if window != nil {
			window.mouseMu.Lock()
			window.mouseButtons[1] = false
			window.mouseMu.Unlock()
		}
		return 0
	case wmMButtonDown:
		if window != nil {
			window.mouseMu.Lock()
			window.mouseButtons[2] = true
			window.mouseMu.Unlock()
		}
		return 0
	case wmMButtonUp:
		if window != nil {
			window.mouseMu.Lock()
			window.mouseButtons[2] = false
			window.mouseMu.Unlock()
		}
		return 0
	case wmXButtonDown:
		if window != nil {
			btn := 3
			if highWord(uint32(wParam)) == 2 {
				btn = 4
			}
			window.mouseMu.Lock()
			window.mouseButtons[btn] = true
			window.mouseMu.Unlock()
		}
		return 0
	case wmXButtonUp:
		if window != nil {
			btn := 3
			if highWord(uint32(wParam)) == 2 {
				btn = 4
			}
			window.mouseMu.Lock()
			window.mouseButtons[btn] = false
			window.mouseMu.Unlock()
		}
		return 0
	case wmMouseWheel:
		if window != nil {
			delta := float64(int16(highWord(uint32(wParam)))) / 120.0
			window.mouseMu.Lock()
			window.mouseWheelY += delta
			window.mouseMu.Unlock()
		}
		return 0
	case wmMouseHWheel:
		if window != nil {
			delta := float64(int16(highWord(uint32(wParam)))) / 120.0
			window.mouseMu.Lock()
			window.mouseWheelX += delta
			window.mouseMu.Unlock()
		}
		return 0
	default:
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}
}

func lowWord(v uint32) uint16 {
	return uint16(v & 0xFFFF)
}

func highWord(v uint32) uint16 {
	return uint16((v >> 16) & 0xFFFF)
}

func wrapCallErr(name string, callErr error) error {
	if callErr == nil || callErr == windows.ERROR_SUCCESS {
		return fmt.Errorf("win32: %s failed", name)
	}
	return fmt.Errorf("win32: %s failed: %w", name, callErr)
}
