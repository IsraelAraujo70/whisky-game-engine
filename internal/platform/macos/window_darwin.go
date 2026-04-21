//go:build darwin

package macos

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/IsraelAraujo70/whisky-game-engine/input"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

var _ platformapi.NativeWindow = (*Window)(nil)

const (
	nsApplicationActivationPolicyRegular = 0
	nsBackingStoreBuffered               = 2

	nsWindowStyleMaskTitled         = 1 << 0
	nsWindowStyleMaskClosable       = 1 << 1
	nsWindowStyleMaskResizable      = 1 << 3
	nsWindowStyleMaskMiniaturizable = 1 << 2

	keyCodeA            = 0x00
	keyCodeS            = 0x01
	keyCodeD            = 0x02
	keyCodeF            = 0x03
	keyCodeH            = 0x04
	keyCodeG            = 0x05
	keyCodeZ            = 0x06
	keyCodeX            = 0x07
	keyCodeC            = 0x08
	keyCodeV            = 0x09
	keyCodeB            = 0x0B
	keyCodeQ            = 0x0C
	keyCodeW            = 0x0D
	keyCodeE            = 0x0E
	keyCodeR            = 0x0F
	keyCodeY            = 0x10
	keyCodeT            = 0x11
	keyCode1            = 0x12
	keyCode2            = 0x13
	keyCode3            = 0x14
	keyCode4            = 0x15
	keyCode6            = 0x16
	keyCode5            = 0x17
	keyCodeEqual        = 0x18
	keyCode9            = 0x19
	keyCode7            = 0x1A
	keyCodeMinus        = 0x1B
	keyCode8            = 0x1C
	keyCode0            = 0x1D
	keyCodeO            = 0x1F
	keyCodeU            = 0x20
	keyCodeI            = 0x22
	keyCodeP            = 0x23
	keyCodeL            = 0x25
	keyCodeJ            = 0x26
	keyCodeK            = 0x28
	keyCodeN            = 0x2D
	keyCodeM            = 0x2E
	keyCodeTab          = 0x30
	keyCodeSpace        = 0x31
	keyCodeBackspace    = 0x33
	keyCodeEnter        = 0x24
	keyCodeEscape       = 0x35
	keyCodeCommand      = 0x37
	keyCodeShift        = 0x38
	keyCodeCapsLock     = 0x39
	keyCodeOption       = 0x3A
	keyCodeControl      = 0x3B
	keyCodeRightShift   = 0x3C
	keyCodeRightOption  = 0x3D
	keyCodeRightControl = 0x3E
	keyCodeLeftArrow    = 0x7B
	keyCodeRightArrow   = 0x7C
	keyCodeDownArrow    = 0x7D
	keyCodeUpArrow      = 0x7E
	keyCodeF1           = 0x7A
	keyCodeF2           = 0x78
	keyCodeF3           = 0x63
	keyCodeF4           = 0x76
	keyCodeF5           = 0x60
	keyCodeF6           = 0x61
	keyCodeF7           = 0x62
	keyCodeF8           = 0x64
	keyCodeF9           = 0x65
	keyCodeF10          = 0x6D
	keyCodeF11          = 0x67
	keyCodeF12          = 0x6F
)

type NSPoint struct {
	X float64
	Y float64
}

type NSSize struct {
	Width  float64
	Height float64
}

type NSRect struct {
	Origin NSPoint
	Size   NSSize
}

type keyBinding struct {
	keyCode uint16
	control string
}

type Window struct {
	window      objc.ID
	view        objc.ID
	layer       objc.ID
	delegate    objc.ID
	keyBindings []keyBinding
	pressedMu   sync.RWMutex
	pressed     map[uint16]bool
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

var (
	appKitOnce sync.Once
	appKitErr  error

	windowsMu           sync.RWMutex
	windowsByView       = map[uintptr]*Window{}
	windowDelegateClass objc.Class

	selSharedApplication            = objc.RegisterName("sharedApplication")
	selSetActivationPolicy          = objc.RegisterName("setActivationPolicy:")
	selActivateIgnoringOtherApps    = objc.RegisterName("activateIgnoringOtherApps:")
	selFinishLaunching              = objc.RegisterName("finishLaunching")
	selNextEventMatchingMask        = objc.RegisterName("nextEventMatchingMask:untilDate:inMode:dequeue:")
	selSendEvent                    = objc.RegisterName("sendEvent:")
	selUpdateWindows                = objc.RegisterName("updateWindows")
	selDateWithTimeIntervalSinceNow = objc.RegisterName("dateWithTimeIntervalSinceNow:")
	selDefaultRunLoopMode           = objc.RegisterName("defaultRunLoopMode")
	selAlloc                        = objc.RegisterName("alloc")
	selInitWithContentRect          = objc.RegisterName("initWithContentRect:styleMask:backing:defer:")
	selSetTitle                     = objc.RegisterName("setTitle:")
	selCenter                       = objc.RegisterName("center")
	selMakeKeyAndOrderFront         = objc.RegisterName("makeKeyAndOrderFront:")
	selSetReleasedWhenClosed        = objc.RegisterName("setReleasedWhenClosed:")
	selSetDelegate                  = objc.RegisterName("setDelegate:")
	selContentView                  = objc.RegisterName("contentView")
	selSetWantsLayer                = objc.RegisterName("setWantsLayer:")
	selSetLayer                     = objc.RegisterName("setLayer:")
	selSetNeedsDisplay              = objc.RegisterName("setNeedsDisplay:")
	selSetAutoresizingMask          = objc.RegisterName("setAutoresizingMask:")
	selFrame                        = objc.RegisterName("frame")
	selBounds                       = objc.RegisterName("bounds")
	selClose                        = objc.RegisterName("close")
	selRelease                      = objc.RegisterName("release")
	selStringWithUTF8String         = objc.RegisterName("stringWithUTF8String:")
	selWindowShouldClose            = objc.RegisterName("windowShouldClose:")
	selWindowDidResize              = objc.RegisterName("windowDidResize:")
	selAcceptsFirstResponder        = objc.RegisterName("acceptsFirstResponder")
	selCanBecomeKeyView             = objc.RegisterName("canBecomeKeyView")
	selViewDidMoveToWindow          = objc.RegisterName("viewDidMoveToWindow")
	selKeyDown                      = objc.RegisterName("keyDown:")
	selKeyUp                        = objc.RegisterName("keyUp:")
	selFlagsChanged                 = objc.RegisterName("flagsChanged:")
	selWindow                       = objc.RegisterName("window")
	selKeyCode                      = objc.RegisterName("keyCode")
	selType                         = objc.RegisterName("type")
	selClassLayer                   = objc.RegisterName("layer")
	selLocationInWindow             = objc.RegisterName("locationInWindow")
	selMouseDown                    = objc.RegisterName("mouseDown:")
	selMouseUp                      = objc.RegisterName("mouseUp:")
	selRightMouseDown               = objc.RegisterName("rightMouseDown:")
	selRightMouseUp                 = objc.RegisterName("rightMouseUp:")
	selOtherMouseDown               = objc.RegisterName("otherMouseDown:")
	selOtherMouseUp                 = objc.RegisterName("otherMouseUp:")
	selMouseMoved                   = objc.RegisterName("mouseMoved:")
	selMouseDragged                 = objc.RegisterName("mouseDragged:")
	selScrollWheel                  = objc.RegisterName("scrollWheel:")
	selDeltaX                       = objc.RegisterName("deltaX")
	selDeltaY                       = objc.RegisterName("deltaY")
	selButtonNumber                 = objc.RegisterName("buttonNumber")
)

func New(title string, width, height int, keyMap map[string]string) (*Window, error) {
	runtime.LockOSThread()
	if err := loadAppKit(); err != nil {
		return nil, err
	}
	if width <= 0 {
		width = 1280
	}
	if height <= 0 {
		height = 720
	}
	if err := ensureWindowDelegateClass(); err != nil {
		return nil, err
	}

	app := objc.ID(objc.GetClass("NSApplication")).Send(selSharedApplication)
	app.Send(selSetActivationPolicy, nsApplicationActivationPolicyRegular)
	app.Send(selFinishLaunching)

	window := objc.ID(objc.GetClass("NSWindow")).Send(selAlloc)
	window = window.Send(selInitWithContentRect,
		nsMakeRect(0, 0, float64(width), float64(height)),
		nsWindowStyleMaskTitled|nsWindowStyleMaskClosable|nsWindowStyleMaskMiniaturizable|nsWindowStyleMaskResizable,
		nsBackingStoreBuffered,
		false,
	)
	if window == 0 {
		return nil, fmt.Errorf("macos: failed to create NSWindow")
	}

	titleString := objc.ID(objc.GetClass("NSString")).Send(selStringWithUTF8String, title)
	window.Send(selSetTitle, titleString)
	window.Send(selSetReleasedWhenClosed, false)
	window.Send(selCenter)

	view := objc.Send[objc.ID](window, selContentView)
	if view == 0 {
		window.Send(selRelease)
		return nil, fmt.Errorf("macos: failed to get content view")
	}
	view.Send(selSetWantsLayer, true)
	view.Send(selSetAutoresizingMask, uintptr(1<<1|1<<4))

	win := &Window{
		window:      window,
		view:        view,
		keyBindings: buildKeyBindings(keyMap),
		pressed:     make(map[uint16]bool),
	}
	win.width.Store(int32(width))
	win.height.Store(int32(height))

	delegate, err := newWindowDelegate(win)
	if err != nil {
		window.Send(selRelease)
		return nil, err
	}
	win.delegate = delegate
	window.Send(selSetDelegate, delegate)

	windowsMu.Lock()
	windowsByView[uintptr(view)] = win
	windowsMu.Unlock()

	window.Send(selMakeKeyAndOrderFront, objc.ID(0))
	app.Send(selActivateIgnoringOtherApps, true)
	win.refreshSize()
	return win, nil
}

func (w *Window) UpdateInput(state *input.State) {
	if w == nil || state == nil {
		return
	}
	w.pressedMu.RLock()
	for _, binding := range w.keyBindings {
		state.SetPressed(binding.control, false)
	}
	for _, binding := range w.keyBindings {
		if w.pressed[binding.keyCode] {
			state.SetPressed(binding.control, true)
		}
	}
	w.pressedMu.RUnlock()

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

	pollGCControllers(state)
}

func (w *Window) PumpEvents() bool {
	if w == nil || w.closed.Load() {
		return true
	}
	app := objc.ID(objc.GetClass("NSApplication")).Send(selSharedApplication)
	mode := objc.ID(objc.GetClass("NSString")).Send(selStringWithUTF8String, "kCFRunLoopDefaultMode")
	date := objc.ID(objc.GetClass("NSDate")).Send(selDateWithTimeIntervalSinceNow, 0.0)
	for {
		event := objc.Send[objc.ID](app, selNextEventMatchingMask, ^uintptr(0), date, mode, true)
		if event == 0 {
			break
		}
		app.Send(selSendEvent, event)
	}
	app.Send(selUpdateWindows)
	return w.closed.Load()
}

func (w *Window) Size() (width, height int) {
	if w == nil {
		return 0, 0
	}
	return int(w.width.Load()), int(w.height.Load())
}

func (w *Window) NativeHandle() platformapi.NativeWindowHandle {
	if w == nil {
		return platformapi.NativeWindowHandle{}
	}
	return platformapi.NativeWindowHandle{
		Kind:   platformapi.NativeWindowKindCocoa,
		Window: uintptr(w.window),
		View:   uintptr(w.view),
		Layer:  uintptr(w.layer),
	}
}

func (w *Window) Destroy() error {
	if w == nil {
		return nil
	}
	w.closed.Store(true)
	windowsMu.Lock()
	delete(windowsByView, uintptr(w.view))
	windowsMu.Unlock()
	if w.window != 0 {
		w.window.Send(selClose)
		w.window.Send(selRelease)
		w.window = 0
	}
	if w.delegate != 0 {
		w.delegate.Send(selRelease)
		w.delegate = 0
	}
	w.view = 0
	w.layer = 0
	return nil
}

func (w *Window) refreshSize() {
	if w == nil || w.view == 0 {
		return
	}
	bounds := objc.Send[NSRect](w.view, selBounds)
	if bounds.Size.Width <= 0 || bounds.Size.Height <= 0 {
		return
	}
	w.width.Store(int32(bounds.Size.Width))
	w.height.Store(int32(bounds.Size.Height))
}

func (w *Window) setLayer(layer objc.ID) {
	if w == nil {
		return
	}
	w.layer = layer
	if w.view != 0 && layer != 0 {
		w.view.Send(selSetLayer, layer)
		w.view.Send(selSetNeedsDisplay, true)
	}
}

func (w *Window) AttachLayer(layer objc.ID) {
	w.setLayer(layer)
}

func (w *Window) setKeyPressed(keyCode uint16, pressed bool) {
	if w == nil {
		return
	}
	w.pressedMu.Lock()
	if pressed {
		w.pressed[keyCode] = true
		if keyCode == keyCodeEscape {
			w.closed.Store(true)
		}
	} else {
		delete(w.pressed, keyCode)
	}
	w.pressedMu.Unlock()
}

func nsMakeRect(x, y, width, height float64) NSRect {
	return NSRect{Origin: NSPoint{X: x, Y: y}, Size: NSSize{Width: width, Height: height}}
}

func buildKeyBindings(keyMap map[string]string) []keyBinding {
	bindings := make([]keyBinding, 0, len(keyMap))
	for keyName, control := range keyMap {
		keyCode, ok := nameToMacVirtualKey[keyName]
		if !ok {
			continue
		}
		bindings = append(bindings, keyBinding{keyCode: keyCode, control: control})
	}
	return bindings
}

var nameToMacVirtualKey = map[string]uint16{
	"a": keyCodeA, "b": keyCodeB, "c": keyCodeC, "d": keyCodeD, "e": keyCodeE, "f": keyCodeF,
	"g": keyCodeG, "h": keyCodeH, "i": keyCodeI, "j": keyCodeJ, "k": keyCodeK, "l": keyCodeL,
	"m": keyCodeM, "n": keyCodeN, "o": keyCodeO, "p": keyCodeP, "q": keyCodeQ, "r": keyCodeR,
	"s": keyCodeS, "t": keyCodeT, "u": keyCodeU, "v": keyCodeV, "w": keyCodeW, "x": keyCodeX,
	"y": keyCodeY, "z": keyCodeZ,
	"0": keyCode0, "1": keyCode1, "2": keyCode2, "3": keyCode3, "4": keyCode4,
	"5": keyCode5, "6": keyCode6, "7": keyCode7, "8": keyCode8, "9": keyCode9,
	"up": keyCodeUpArrow, "down": keyCodeDownArrow, "left": keyCodeLeftArrow, "right": keyCodeRightArrow,
	"space": keyCodeSpace, "enter": keyCodeEnter, "escape": keyCodeEscape, "backspace": keyCodeBackspace,
	"tab": keyCodeTab, "lshift": keyCodeShift, "rshift": keyCodeRightShift, "lctrl": keyCodeControl,
	"rctrl": keyCodeRightControl, "lalt": keyCodeOption, "ralt": keyCodeRightOption,
	"f1": keyCodeF1, "f2": keyCodeF2, "f3": keyCodeF3, "f4": keyCodeF4, "f5": keyCodeF5, "f6": keyCodeF6,
	"f7": keyCodeF7, "f8": keyCodeF8, "f9": keyCodeF9, "f10": keyCodeF10, "f11": keyCodeF11, "f12": keyCodeF12,
}

func loadAppKit() error {
	appKitOnce.Do(func() {
		if _, err := purego.Dlopen("/System/Library/Frameworks/Cocoa.framework/Cocoa", purego.RTLD_GLOBAL|purego.RTLD_LAZY); err != nil {
			appKitErr = fmt.Errorf("macos: load Cocoa: %w", err)
			return
		}
		if _, err := purego.Dlopen("/System/Library/Frameworks/QuartzCore.framework/QuartzCore", purego.RTLD_GLOBAL|purego.RTLD_LAZY); err != nil {
			appKitErr = fmt.Errorf("macos: load QuartzCore: %w", err)
			return
		}
		if _, err := purego.Dlopen("/System/Library/Frameworks/Metal.framework/Metal", purego.RTLD_GLOBAL|purego.RTLD_LAZY); err != nil {
			appKitErr = fmt.Errorf("macos: load Metal: %w", err)
			return
		}
	})
	return appKitErr
}

func ensureWindowDelegateClass() error {
	if windowDelegateClass != 0 {
		return nil
	}
	class, err := objc.RegisterClass(
		"WhiskyWindowDelegate",
		objc.GetClass("NSObject"),
		nil,
		nil,
		[]objc.MethodDef{
			{Cmd: selWindowShouldClose, Fn: windowShouldClose},
			{Cmd: selWindowDidResize, Fn: windowDidResize},
			{Cmd: selKeyDown, Fn: viewKeyDown},
			{Cmd: selKeyUp, Fn: viewKeyUp},
			{Cmd: selFlagsChanged, Fn: viewFlagsChanged},
			{Cmd: selAcceptsFirstResponder, Fn: acceptsFirstResponder},
			{Cmd: selCanBecomeKeyView, Fn: acceptsFirstResponder},
			{Cmd: selMouseDown, Fn: viewMouseDown},
			{Cmd: selMouseUp, Fn: viewMouseUp},
			{Cmd: selRightMouseDown, Fn: viewRightMouseDown},
			{Cmd: selRightMouseUp, Fn: viewRightMouseUp},
			{Cmd: selOtherMouseDown, Fn: viewOtherMouseDown},
			{Cmd: selOtherMouseUp, Fn: viewOtherMouseUp},
			{Cmd: selMouseMoved, Fn: viewMouseMoved},
			{Cmd: selMouseDragged, Fn: viewMouseMoved},
			{Cmd: selScrollWheel, Fn: viewScrollWheel},
		},
	)
	if err != nil {
		return err
	}
	windowDelegateClass = class
	return nil
}

func newWindowDelegate(w *Window) (objc.ID, error) {
	delegate := objc.ID(windowDelegateClass).Send(objc.RegisterName("new"))
	if delegate == 0 {
		return 0, fmt.Errorf("macos: failed to instantiate delegate")
	}
	return delegate, nil
}

func lookupWindowForEvent(event objc.ID) *Window {
	if event == 0 {
		return nil
	}
	window := objc.Send[objc.ID](event, selWindow)
	if window == 0 {
		return nil
	}
	view := objc.Send[objc.ID](window, selContentView)
	if view == 0 {
		return nil
	}
	windowsMu.RLock()
	win := windowsByView[uintptr(view)]
	windowsMu.RUnlock()
	return win
}

func windowShouldClose(self objc.ID, cmd objc.SEL, sender objc.ID) bool {
	_ = self
	_ = cmd
	windowsMu.RLock()
	win := windowsByView[uintptr(objc.Send[objc.ID](sender, selContentView))]
	windowsMu.RUnlock()
	if win != nil {
		win.closed.Store(true)
	}
	return true
}

func windowDidResize(self objc.ID, cmd objc.SEL, notification objc.ID) {
	_ = self
	_ = cmd
	window := objc.Send[objc.ID](notification, objc.RegisterName("object"))
	if window == 0 {
		return
	}
	windowsMu.RLock()
	win := windowsByView[uintptr(objc.Send[objc.ID](window, selContentView))]
	windowsMu.RUnlock()
	if win != nil {
		win.refreshSize()
	}
}

func viewKeyDown(self objc.ID, cmd objc.SEL, event objc.ID) {
	_ = self
	_ = cmd
	win := lookupWindowForEvent(event)
	if win != nil {
		keyCode := objc.Send[uint16](event, selKeyCode)
		win.setKeyPressed(keyCode, true)
	}
}

func viewKeyUp(self objc.ID, cmd objc.SEL, event objc.ID) {
	_ = self
	_ = cmd
	win := lookupWindowForEvent(event)
	if win != nil {
		keyCode := objc.Send[uint16](event, selKeyCode)
		win.setKeyPressed(keyCode, false)
	}
}

func viewFlagsChanged(self objc.ID, cmd objc.SEL, event objc.ID) {
	_ = self
	_ = cmd
	win := lookupWindowForEvent(event)
	if win == nil {
		return
	}
	keyCode := objc.Send[uint16](event, selKeyCode)
	typ := objc.Send[uint64](event, selType)
	switch typ {
	case 10:
		win.setKeyPressed(keyCode, true)
	case 12:
		win.setKeyPressed(keyCode, false)
	}
}

func acceptsFirstResponder(self objc.ID, cmd objc.SEL) bool {
	_ = self
	_ = cmd
	return true
}

func viewMouseDown(self objc.ID, cmd objc.SEL, event objc.ID) {
	_ = self
	_ = cmd
	win := lookupWindowForEvent(event)
	if win != nil {
		win.updateMousePosition(event)
		win.mouseMu.Lock()
		win.mouseButtons[0] = true
		win.mouseMu.Unlock()
	}
}

func viewMouseUp(self objc.ID, cmd objc.SEL, event objc.ID) {
	_ = self
	_ = cmd
	win := lookupWindowForEvent(event)
	if win != nil {
		win.updateMousePosition(event)
		win.mouseMu.Lock()
		win.mouseButtons[0] = false
		win.mouseMu.Unlock()
	}
}

func viewRightMouseDown(self objc.ID, cmd objc.SEL, event objc.ID) {
	_ = self
	_ = cmd
	win := lookupWindowForEvent(event)
	if win != nil {
		win.updateMousePosition(event)
		win.mouseMu.Lock()
		win.mouseButtons[1] = true
		win.mouseMu.Unlock()
	}
}

func viewRightMouseUp(self objc.ID, cmd objc.SEL, event objc.ID) {
	_ = self
	_ = cmd
	win := lookupWindowForEvent(event)
	if win != nil {
		win.updateMousePosition(event)
		win.mouseMu.Lock()
		win.mouseButtons[1] = false
		win.mouseMu.Unlock()
	}
}

func viewOtherMouseDown(self objc.ID, cmd objc.SEL, event objc.ID) {
	_ = self
	_ = cmd
	win := lookupWindowForEvent(event)
	if win != nil {
		win.updateMousePosition(event)
		btn := int(objc.Send[int64](event, selButtonNumber))
		win.mouseMu.Lock()
		if btn == 2 {
			win.mouseButtons[2] = true
		} else if btn == 3 {
			win.mouseButtons[3] = true
		} else if btn == 4 {
			win.mouseButtons[4] = true
		}
		win.mouseMu.Unlock()
	}
}

func viewOtherMouseUp(self objc.ID, cmd objc.SEL, event objc.ID) {
	_ = self
	_ = cmd
	win := lookupWindowForEvent(event)
	if win != nil {
		win.updateMousePosition(event)
		btn := int(objc.Send[int64](event, selButtonNumber))
		win.mouseMu.Lock()
		if btn == 2 {
			win.mouseButtons[2] = false
		} else if btn == 3 {
			win.mouseButtons[3] = false
		} else if btn == 4 {
			win.mouseButtons[4] = false
		}
		win.mouseMu.Unlock()
	}
}

func viewMouseMoved(self objc.ID, cmd objc.SEL, event objc.ID) {
	_ = self
	_ = cmd
	win := lookupWindowForEvent(event)
	if win != nil {
		win.updateMousePosition(event)
	}
}

func viewScrollWheel(self objc.ID, cmd objc.SEL, event objc.ID) {
	_ = self
	_ = cmd
	win := lookupWindowForEvent(event)
	if win != nil {
		dx := objc.Send[float64](event, selDeltaX)
		dy := objc.Send[float64](event, selDeltaY)
		win.mouseMu.Lock()
		win.mouseWheelX += dx
		win.mouseWheelY += dy
		win.mouseMu.Unlock()
	}
}

func (w *Window) updateMousePosition(event objc.ID) {
	if w == nil || event == 0 {
		return
	}
	pt := objc.Send[NSPoint](event, selLocationInWindow)
	w.mouseMu.Lock()
	w.mouseX = pt.X
	w.mouseY = pt.Y
	w.mouseMu.Unlock()
}
