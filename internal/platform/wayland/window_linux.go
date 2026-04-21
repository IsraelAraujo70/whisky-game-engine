//go:build linux && (amd64 || arm64)

package wayland

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/unix"

	"github.com/IsraelAraujo70/whisky-game-engine/input"
	"github.com/IsraelAraujo70/whisky-game-engine/internal/platform/linux"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

var (
	ErrUnavailable = errors.New("wayland platform backend requires libwayland-client and an active compositor")
	// ErrNotImplemented is kept as a legacy alias for the old Linux selection fallback.
	ErrNotImplemented = ErrUnavailable
)

var _ platformapi.NativeWindow = (*Window)(nil)

const (
	wlMarshalFlagDestroy = 1 << 0

	wlDisplayGetRegistryOpcode = 1
	wlRegistryBindOpcode       = 0

	wlCompositorCreateSurfaceOpcode = 0

	wlSurfaceDestroyOpcode = 0
	wlSurfaceCommitOpcode  = 6

	wlSeatGetPointerOpcode  = 0
	wlSeatGetKeyboardOpcode = 1

	wlSeatCapabilityPointer  = 1 << 0
	wlSeatCapabilityKeyboard = 1 << 1

	wlKeyboardStateReleased = 0
	wlKeyboardStatePressed  = 1
	wlKeyboardStateRepeated = 2

	xdgWmBaseDestroyOpcode       = 0
	xdgWmBaseGetXDGSurfaceOpcode = 2
	xdgWmBasePongOpcode          = 3

	xdgSurfaceDestroyOpcode      = 0
	xdgSurfaceGetToplevelOpcode  = 1
	xdgSurfaceAckConfigureOpcode = 4

	xdgToplevelDestroyOpcode  = 0
	xdgToplevelSetTitleOpcode = 2
	xdgToplevelSetAppIDOpcode = 3

	maxCompositorVersion = 1
	maxSeatVersion       = 1
	maxXDGWMBaseVersion  = 1
)

type wlMessage struct {
	Name      *byte
	Signature *byte
	Types     **wlInterface
}

type wlInterface struct {
	Name        *byte
	Version     int32
	MethodCount int32
	Methods     *wlMessage
	EventCount  int32
	Events      *wlMessage
}

type wlArray struct {
	Size  uintptr
	Alloc uintptr
	Data  unsafe.Pointer
}

type wlRegistryListener struct {
	Global       uintptr
	GlobalRemove uintptr
}

type xdgWmBaseListener struct {
	Ping uintptr
}

type xdgSurfaceListener struct {
	Configure uintptr
}

type xdgToplevelListener struct {
	Configure uintptr
	Close     uintptr
}

type wlSeatListener struct {
	Capabilities uintptr
	Name         uintptr
}

type wlKeyboardListener struct {
	Keymap    uintptr
	Enter     uintptr
	Leave     uintptr
	Key       uintptr
	Modifiers uintptr
}

type wlPointerListener struct {
	Enter  uintptr
	Leave  uintptr
	Motion uintptr
	Button uintptr
	Axis   uintptr
}

type Window struct {
	token uintptr

	display    unsafe.Pointer
	registry   unsafe.Pointer
	compositor unsafe.Pointer
	xdgWMBase  unsafe.Pointer
	surface    unsafe.Pointer
	xdgSurface unsafe.Pointer
	toplevel   unsafe.Pointer
	seat       unsafe.Pointer
	keyboard   unsafe.Pointer
	pointer    unsafe.Pointer

	keyBindings []keyBinding

	pressedMu sync.RWMutex
	pressed   map[uint32]bool

	mouseMu     sync.Mutex
	mouseX      float64
	mouseY      float64
	mouseButtons [5]bool
	mouseWheelX float64
	mouseWheelY float64

	gamepadPoller *linux.GamepadPoller

	xkbContext unsafe.Pointer
	xkbKeymap  unsafe.Pointer
	xkbState   unsafe.Pointer

	closed     atomic.Bool
	configured atomic.Bool
	width      atomic.Int32
	height     atomic.Int32
}

var (
	waylandOnce sync.Once
	waylandErr  error

	waylandHandle uintptr

	wlDisplayConnect         func(name *byte) unsafe.Pointer
	wlDisplayDisconnect      func(display unsafe.Pointer)
	wlDisplayDispatch        func(display unsafe.Pointer) int32
	wlDisplayDispatchPending func(display unsafe.Pointer) int32
	wlDisplayRoundtrip       func(display unsafe.Pointer) int32
	wlDisplayFlush           func(display unsafe.Pointer) int32
	wlDisplayGetFD           func(display unsafe.Pointer) int32
	wlProxyAddListener       func(proxy unsafe.Pointer, implementation unsafe.Pointer, data unsafe.Pointer) int32
	wlProxyDestroy           func(proxy unsafe.Pointer)
	wlProxyGetVersion        func(proxy unsafe.Pointer) uint32

	wlProxyMarshalFlags uintptr

	wlRegistryInterface   *wlInterface
	wlCompositorInterface *wlInterface
	wlSurfaceInterface    *wlInterface
	wlSeatInterface       *wlInterface
	wlKeyboardInterface   *wlInterface
	wlPointerInterface    *wlInterface

	waylandWindows      sync.Map
	nextWaylandWindowID atomic.Uintptr
)

var (
	cEmpty                = []byte{0}
	cStringSig            = []byte("s\x00")
	cUintSig              = []byte("u\x00")
	cIntIntArraySig       = []byte("iia\x00")
	cNewIDSig             = []byte("n\x00")
	cNewIDObjectSig       = []byte("no\x00")
	cNewIDObjectObjectSig = []byte("noo\x00")
	cObjectSig            = []byte("o\x00")
	cNullableObjectSig    = []byte("?o\x00")
	cIntIntIntIntSig      = []byte("iiii\x00")

	cXDGWMBaseName     = []byte("xdg_wm_base\x00")
	cXDGSurfaceName    = []byte("xdg_surface\x00")
	cXDGToplevelName   = []byte("xdg_toplevel\x00")
	cXDGPositionerName = []byte("xdg_positioner\x00")

	cXDGWMBaseDestroyName    = []byte("destroy\x00")
	cXDGWMBasePositionerName = []byte("create_positioner\x00")
	cXDGWMBaseSurfaceName    = []byte("get_xdg_surface\x00")
	cXDGWMBasePongName       = []byte("pong\x00")
	cXDGWMBasePingName       = []byte("ping\x00")

	cXDGSurfaceDestroyName        = []byte("destroy\x00")
	cXDGSurfaceGetToplevelName    = []byte("get_toplevel\x00")
	cXDGSurfaceGetPopupName       = []byte("get_popup\x00")
	cXDGSurfaceSetGeometryName    = []byte("set_window_geometry\x00")
	cXDGSurfaceAckConfigureName   = []byte("ack_configure\x00")
	cXDGSurfaceConfigureEventName = []byte("configure\x00")

	cXDGToplevelDestroyName        = []byte("destroy\x00")
	cXDGToplevelSetParentName      = []byte("set_parent\x00")
	cXDGToplevelSetTitleName       = []byte("set_title\x00")
	cXDGToplevelSetAppIDName       = []byte("set_app_id\x00")
	cXDGToplevelShowWindowMenuName = []byte("show_window_menu\x00")
	cXDGToplevelMoveName           = []byte("move\x00")
	cXDGToplevelResizeName         = []byte("resize\x00")
	cXDGToplevelSetMaxSizeName     = []byte("set_max_size\x00")
	cXDGToplevelSetMinSizeName     = []byte("set_min_size\x00")
	cXDGToplevelSetMaximizedName   = []byte("set_maximized\x00")
	cXDGToplevelUnsetMaximizedName = []byte("unset_maximized\x00")
	cXDGToplevelSetFullscreenName  = []byte("set_fullscreen\x00")
	cXDGToplevelUnsetFullscreenName = []byte("unset_fullscreen\x00")
	cXDGToplevelSetMinimizedName   = []byte("set_minimized\x00")
	cXDGToplevelConfigureName      = []byte("configure\x00")
	cXDGToplevelCloseName          = []byte("close\x00")

	cObjectUintSig     = []byte("ou\x00")
	cObjectUintUintSig = []byte("ouu\x00")
	cObjectUintIntIntSig = []byte("ouii\x00")
	cIntIntSig         = []byte("ii\x00")
)

var (
	xdgPositionerInterface = wlInterface{
		Name:    &cXDGPositionerName[0],
		Version: 1,
	}

	xdgWmBaseCreatePositionerTypes = [...]*wlInterface{
		nil,
	}
	xdgWmBaseGetSurfaceTypes = [...]*wlInterface{
		nil,
		wlSurfaceInterface,
	}

	xdgWmBaseMethods = [...]wlMessage{
		{Name: &cXDGWMBaseDestroyName[0], Signature: &cEmpty[0]},
		{Name: &cXDGWMBasePositionerName[0], Signature: &cNewIDSig[0], Types: (**wlInterface)(unsafe.Pointer(&xdgWmBaseCreatePositionerTypes[0]))},
		{Name: &cXDGWMBaseSurfaceName[0], Signature: &cNewIDObjectSig[0], Types: (**wlInterface)(unsafe.Pointer(&xdgWmBaseGetSurfaceTypes[0]))},
		{Name: &cXDGWMBasePongName[0], Signature: &cUintSig[0]},
	}
	xdgWmBaseEvents = [...]wlMessage{
		{Name: &cXDGWMBasePingName[0], Signature: &cUintSig[0]},
	}
	xdgWmBaseInterface = wlInterface{
		Name:        &cXDGWMBaseName[0],
		Version:     1,
		MethodCount: int32(len(xdgWmBaseMethods)),
		Methods:     &xdgWmBaseMethods[0],
		EventCount:  int32(len(xdgWmBaseEvents)),
		Events:      &xdgWmBaseEvents[0],
	}

	xdgSurfaceGetToplevelTypes = [...]*wlInterface{
		nil,
	}
	xdgSurfaceGetPopupTypes = [...]*wlInterface{
		nil,
		nil,
		&xdgPositionerInterface,
	}

	xdgSurfaceMethods = [...]wlMessage{
		{Name: &cXDGSurfaceDestroyName[0], Signature: &cEmpty[0]},
		{Name: &cXDGSurfaceGetToplevelName[0], Signature: &cNewIDSig[0], Types: (**wlInterface)(unsafe.Pointer(&xdgSurfaceGetToplevelTypes[0]))},
		{Name: &cXDGSurfaceGetPopupName[0], Signature: &cNewIDObjectObjectSig[0], Types: (**wlInterface)(unsafe.Pointer(&xdgSurfaceGetPopupTypes[0]))},
		{Name: &cXDGSurfaceSetGeometryName[0], Signature: &cIntIntIntIntSig[0]},
		{Name: &cXDGSurfaceAckConfigureName[0], Signature: &cUintSig[0]},
	}
	xdgSurfaceEvents = [...]wlMessage{
		{Name: &cXDGSurfaceConfigureEventName[0], Signature: &cUintSig[0]},
	}
	xdgSurfaceInterface = wlInterface{
		Name:        &cXDGSurfaceName[0],
		Version:     1,
		MethodCount: int32(len(xdgSurfaceMethods)),
		Methods:     &xdgSurfaceMethods[0],
		EventCount:  int32(len(xdgSurfaceEvents)),
		Events:      &xdgSurfaceEvents[0],
	}

	xdgToplevelMethods = [...]wlMessage{
		{Name: &cXDGToplevelDestroyName[0], Signature: &cEmpty[0]},             // 0: destroy
		{Name: &cXDGToplevelSetParentName[0], Signature: &cNullableObjectSig[0]}, // 1: set_parent
		{Name: &cXDGToplevelSetTitleName[0], Signature: &cStringSig[0]},        // 2: set_title
		{Name: &cXDGToplevelSetAppIDName[0], Signature: &cStringSig[0]},        // 3: set_app_id
		{Name: &cXDGToplevelShowWindowMenuName[0], Signature: &cObjectUintIntIntSig[0]}, // 4: show_window_menu
		{Name: &cXDGToplevelMoveName[0], Signature: &cObjectUintSig[0]},        // 5: move
		{Name: &cXDGToplevelResizeName[0], Signature: &cObjectUintUintSig[0]},  // 6: resize
		{Name: &cXDGToplevelSetMaxSizeName[0], Signature: &cIntIntSig[0]},      // 7: set_max_size
		{Name: &cXDGToplevelSetMinSizeName[0], Signature: &cIntIntSig[0]},      // 8: set_min_size
		{Name: &cXDGToplevelSetMaximizedName[0], Signature: &cEmpty[0]},        // 9: set_maximized
		{Name: &cXDGToplevelUnsetMaximizedName[0], Signature: &cEmpty[0]},      // 10: unset_maximized
		{Name: &cXDGToplevelSetFullscreenName[0], Signature: &cNullableObjectSig[0]}, // 11: set_fullscreen
		{Name: &cXDGToplevelUnsetFullscreenName[0], Signature: &cEmpty[0]},     // 12: unset_fullscreen
		{Name: &cXDGToplevelSetMinimizedName[0], Signature: &cEmpty[0]},        // 13: set_minimized
	}
	xdgToplevelEvents = [...]wlMessage{
		{Name: &cXDGToplevelConfigureName[0], Signature: &cIntIntArraySig[0]},
		{Name: &cXDGToplevelCloseName[0], Signature: &cEmpty[0]},
	}
	xdgToplevelInterface = wlInterface{
		Name:        &cXDGToplevelName[0],
		Version:     1,
		MethodCount: int32(len(xdgToplevelMethods)),
		Methods:     &xdgToplevelMethods[0],
		EventCount:  int32(len(xdgToplevelEvents)),
		Events:      &xdgToplevelEvents[0],
	}
)

var (
	registryListener = wlRegistryListener{
		Global:       purego.NewCallback(registryGlobal),
		GlobalRemove: purego.NewCallback(registryGlobalRemove),
	}
	xdgWMBaseListener = xdgWmBaseListener{
		Ping: purego.NewCallback(xdgWMBasePing),
	}
	xdgSurfaceCallbacks = xdgSurfaceListener{
		Configure: purego.NewCallback(xdgSurfaceConfigure),
	}
	toplevelListener = xdgToplevelListener{
		Configure: purego.NewCallback(xdgToplevelConfigure),
		Close:     purego.NewCallback(xdgToplevelClose),
	}
	seatListener = wlSeatListener{
		Capabilities: purego.NewCallback(wlSeatCapabilities),
		Name:         purego.NewCallback(wlSeatName),
	}
	keyboardListener = wlKeyboardListener{
		Keymap:    purego.NewCallback(wlKeyboardKeymap),
		Enter:     purego.NewCallback(wlKeyboardEnter),
		Leave:     purego.NewCallback(wlKeyboardLeave),
		Key:       purego.NewCallback(wlKeyboardKey),
		Modifiers: purego.NewCallback(wlKeyboardModifiers),
	}
	pointerListener = wlPointerListener{
		Enter:  purego.NewCallback(wlPointerEnter),
		Leave:  purego.NewCallback(wlPointerLeave),
		Motion: purego.NewCallback(wlPointerMotion),
		Button: purego.NewCallback(wlPointerButton),
		Axis:   purego.NewCallback(wlPointerAxis),
	}
)

func New(title string, width, height int, keyMap map[string]string) (*Window, error) {
	if err := ensureWayland(); err != nil {
		return nil, err
	}

	display := wlDisplayConnect(nil)
	if display == nil {
		return nil, fmt.Errorf("%w: wl_display_connect failed", ErrUnavailable)
	}

	registry := wlDisplayGetRegistry(display)
	if registry == nil {
		wlDisplayDisconnect(display)
		return nil, errors.New("wayland: wl_display_get_registry failed")
	}

	if title == "" {
		title = "whisky game"
	}

	win := &Window{
		display:       display,
		registry:      registry,
		keyBindings:   buildKeyBindings(keyMap),
		pressed:       make(map[uint32]bool),
		gamepadPoller: linux.NewGamepadPoller(),
	}
	win.width.Store(int32(width))
	win.height.Store(int32(height))
	win.token = nextWaylandWindowID.Add(1)
	waylandWindows.Store(win.token, win)

	if wlProxyAddListener(registry, unsafe.Pointer(&registryListener), unsafe.Pointer(win.token)) != 0 {
		_ = win.Destroy()
		return nil, errors.New("wayland: wl_registry_add_listener failed")
	}

	if wlDisplayRoundtrip(display) < 0 || wlDisplayRoundtrip(display) < 0 {
		_ = win.Destroy()
		return nil, fmt.Errorf("%w: registry discovery failed", ErrUnavailable)
	}
	if win.compositor == nil || win.xdgWMBase == nil {
		_ = win.Destroy()
		return nil, fmt.Errorf("%w: compositor missing wl_compositor or xdg_wm_base", ErrUnavailable)
	}

	win.surface = wlCompositorCreateSurface(win.compositor)
	if win.surface == nil {
		_ = win.Destroy()
		return nil, errors.New("wayland: wl_compositor_create_surface failed")
	}
	win.xdgSurface = xdgWMBaseGetSurface(win.xdgWMBase, win.surface)
	if win.xdgSurface == nil {
		_ = win.Destroy()
		return nil, errors.New("wayland: xdg_wm_base.get_xdg_surface failed")
	}
	if wlProxyAddListener(win.xdgSurface, unsafe.Pointer(&xdgSurfaceCallbacks), unsafe.Pointer(win.token)) != 0 {
		_ = win.Destroy()
		return nil, errors.New("wayland: xdg_surface_add_listener failed")
	}

	win.toplevel = xdgSurfaceGetToplevel(win.xdgSurface)
	if win.toplevel == nil {
		_ = win.Destroy()
		return nil, errors.New("wayland: xdg_surface.get_toplevel failed")
	}
	if wlProxyAddListener(win.toplevel, unsafe.Pointer(&toplevelListener), unsafe.Pointer(win.token)) != 0 {
		_ = win.Destroy()
		return nil, errors.New("wayland: xdg_toplevel_add_listener failed")
	}

	xdgToplevelSetTitle(win.toplevel, title)
	xdgToplevelSetAppID(win.toplevel, appIDFromTitle(title))
	wlSurfaceCommit(win.surface)
	_ = wlDisplayFlush(display)

	if wlDisplayRoundtrip(display) < 0 {
		_ = win.Destroy()
		return nil, fmt.Errorf("%w: initial xdg configure failed", ErrUnavailable)
	}

	return win, nil
}

func (w *Window) UpdateInput(state *input.State) {
	if w == nil || state == nil {
		return
	}

	for _, binding := range w.keyBindings {
		state.SetPressed(binding.control, false)
	}

	w.pressedMu.RLock()
	for _, binding := range w.keyBindings {
		if w.pressed[binding.keysym] {
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
	mouse.SetButton(input.MouseButtonMiddle, buttons[1])
	mouse.SetButton(input.MouseButtonRight, buttons[2])
	mouse.SetButton(input.MouseButtonX1, buttons[3])
	mouse.SetButton(input.MouseButtonX2, buttons[4])
	mouse.AddWheel(mwx, mwy)

	w.gamepadPoller.Poll(state)
}

func (w *Window) PumpEvents() bool {
	if w == nil || w.display == nil || w.closed.Load() {
		return true
	}

	if wlDisplayDispatchPending(w.display) < 0 {
		w.closed.Store(true)
		return true
	}
	_ = wlDisplayFlush(w.display)

	fd := wlDisplayGetFD(w.display)
	if fd < 0 {
		w.closed.Store(true)
		return true
	}

	pollfds := []unix.PollFd{{
		Fd:     fd,
		Events: unix.POLLIN | unix.POLLERR | unix.POLLHUP,
	}}
	n, err := unix.Poll(pollfds, 0)
	if err == nil && n > 0 {
		revents := pollfds[0].Revents
		if revents&(unix.POLLERR|unix.POLLHUP) != 0 {
			w.closed.Store(true)
			return true
		}
		if revents&unix.POLLIN != 0 && wlDisplayDispatch(w.display) < 0 {
			w.closed.Store(true)
			return true
		}
	}

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
		Kind:    platformapi.NativeWindowKindWayland,
		Display: uintptr(w.display),
		Window:  uintptr(w.surface),
	}
}

func (w *Window) Destroy() error {
	if w == nil {
		return nil
	}

	w.closed.Store(true)
	if w.token != 0 {
		waylandWindows.Delete(w.token)
		w.token = 0
	}

	if w.keyboard != nil {
		wlProxyDestroy(w.keyboard)
		w.keyboard = nil
	}
	if w.seat != nil {
		wlProxyDestroy(w.seat)
		w.seat = nil
	}
	if w.toplevel != nil {
		xdgToplevelDestroy(w.toplevel)
		w.toplevel = nil
	}
	if w.xdgSurface != nil {
		xdgSurfaceDestroy(w.xdgSurface)
		w.xdgSurface = nil
	}
	if w.surface != nil {
		wlSurfaceDestroy(w.surface)
		w.surface = nil
	}
	if w.xdgWMBase != nil {
		xdgWMBaseDestroy(w.xdgWMBase)
		w.xdgWMBase = nil
	}
	if w.compositor != nil {
		wlProxyDestroy(w.compositor)
		w.compositor = nil
	}
	if w.registry != nil {
		wlProxyDestroy(w.registry)
		w.registry = nil
	}
	if w.display != nil {
		_ = wlDisplayFlush(w.display)
		wlDisplayDisconnect(w.display)
		w.display = nil
	}

	if w.xkbState != nil {
		xkbStateUnref(w.xkbState)
		w.xkbState = nil
	}
	if w.xkbKeymap != nil {
		xkbKeymapUnref(w.xkbKeymap)
		w.xkbKeymap = nil
	}
	if w.xkbContext != nil {
		xkbContextUnref(w.xkbContext)
		w.xkbContext = nil
	}

	w.clearPressed()
	return nil
}

func ensureWayland() error {
	waylandOnce.Do(func() {
		handle, err := loadWaylandLibrary()
		if err != nil {
			waylandErr = err
			return
		}
		waylandHandle = handle

		purego.RegisterLibFunc(&wlDisplayConnect, handle, "wl_display_connect")
		purego.RegisterLibFunc(&wlDisplayDisconnect, handle, "wl_display_disconnect")
		purego.RegisterLibFunc(&wlDisplayDispatch, handle, "wl_display_dispatch")
		purego.RegisterLibFunc(&wlDisplayDispatchPending, handle, "wl_display_dispatch_pending")
		purego.RegisterLibFunc(&wlDisplayRoundtrip, handle, "wl_display_roundtrip")
		purego.RegisterLibFunc(&wlDisplayFlush, handle, "wl_display_flush")
		purego.RegisterLibFunc(&wlDisplayGetFD, handle, "wl_display_get_fd")
		purego.RegisterLibFunc(&wlProxyAddListener, handle, "wl_proxy_add_listener")
		purego.RegisterLibFunc(&wlProxyDestroy, handle, "wl_proxy_destroy")
		purego.RegisterLibFunc(&wlProxyGetVersion, handle, "wl_proxy_get_version")

		wlProxyMarshalFlags, err = purego.Dlsym(handle, "wl_proxy_marshal_flags")
		if err != nil {
			waylandErr = fmt.Errorf("%w: missing wl_proxy_marshal_flags", ErrUnavailable)
			return
		}

		wlRegistryInterface, err = loadInterface(handle, "wl_registry_interface")
		if err != nil {
			waylandErr = err
			return
		}
		wlCompositorInterface, err = loadInterface(handle, "wl_compositor_interface")
		if err != nil {
			waylandErr = err
			return
		}
		wlSurfaceInterface, err = loadInterface(handle, "wl_surface_interface")
		if err != nil {
			waylandErr = err
			return
		}
		xdgWmBaseGetSurfaceTypes[1] = wlSurfaceInterface
		xdgSurfaceGetPopupTypes[1] = &xdgSurfaceInterface
		wlSeatInterface, err = loadInterface(handle, "wl_seat_interface")
		if err != nil {
			waylandErr = err
			return
		}
		wlKeyboardInterface, err = loadInterface(handle, "wl_keyboard_interface")
		if err != nil {
			waylandErr = err
			return
		}
		wlPointerInterface, err = loadInterface(handle, "wl_pointer_interface")
		if err != nil {
			waylandErr = err
			return
		}
	})
	return waylandErr
}

func loadWaylandLibrary() (uintptr, error) {
	for _, name := range []string{"libwayland-client.so.0", "libwayland-client.so"} {
		handle, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err == nil {
			return handle, nil
		}
	}
	return 0, fmt.Errorf("%w: unable to load libwayland-client", ErrUnavailable)
}

func loadInterface(handle uintptr, name string) (*wlInterface, error) {
	addr, err := purego.Dlsym(handle, name)
	if err != nil {
		return nil, fmt.Errorf("%w: missing %s", ErrUnavailable, name)
	}
	return (*wlInterface)(unsafe.Pointer(addr)), nil
}

func marshalFlags(proxy unsafe.Pointer, opcode uint32, iface *wlInterface, version uint32, flags uint32, args ...uintptr) uintptr {
	params := []uintptr{
		uintptr(proxy),
		uintptr(opcode),
		uintptr(unsafe.Pointer(iface)),
		uintptr(version),
		uintptr(flags),
	}
	params = append(params, args...)
	r1, _, _ := purego.SyscallN(wlProxyMarshalFlags, params...)
	return r1
}

func wlDisplayGetRegistry(display unsafe.Pointer) unsafe.Pointer {
	version := wlProxyGetVersion(display)
	return unsafe.Pointer(marshalFlags(display, wlDisplayGetRegistryOpcode, wlRegistryInterface, version, 0, 0))
}

func wlRegistryBind(registry unsafe.Pointer, name uint32, iface *wlInterface, version uint32) unsafe.Pointer {
	return unsafe.Pointer(marshalFlags(
		registry,
		wlRegistryBindOpcode,
		iface,
		version,
		0,
		uintptr(name),
		uintptr(unsafe.Pointer(iface.Name)),
		uintptr(version),
		0,
	))
}

func wlCompositorCreateSurface(compositor unsafe.Pointer) unsafe.Pointer {
	version := wlProxyGetVersion(compositor)
	return unsafe.Pointer(marshalFlags(compositor, wlCompositorCreateSurfaceOpcode, wlSurfaceInterface, version, 0, 0))
}

func wlSurfaceCommit(surface unsafe.Pointer) {
	marshalFlags(surface, wlSurfaceCommitOpcode, nil, wlProxyGetVersion(surface), 0)
}

func wlSurfaceDestroy(surface unsafe.Pointer) {
	marshalFlags(surface, wlSurfaceDestroyOpcode, nil, wlProxyGetVersion(surface), wlMarshalFlagDestroy)
}

func wlSeatGetPointer(seat unsafe.Pointer) unsafe.Pointer {
	version := wlProxyGetVersion(seat)
	return unsafe.Pointer(marshalFlags(seat, wlSeatGetPointerOpcode, wlPointerInterface, version, 0, 0))
}

func wlSeatGetKeyboard(seat unsafe.Pointer) unsafe.Pointer {
	version := wlProxyGetVersion(seat)
	return unsafe.Pointer(marshalFlags(seat, wlSeatGetKeyboardOpcode, wlKeyboardInterface, version, 0, 0))
}

func xdgWMBaseGetSurface(xdgWMBase, surface unsafe.Pointer) unsafe.Pointer {
	version := wlProxyGetVersion(xdgWMBase)
	return unsafe.Pointer(marshalFlags(xdgWMBase, xdgWmBaseGetXDGSurfaceOpcode, &xdgSurfaceInterface, version, 0, 0, uintptr(surface)))
}

func xdgWMBasePong(xdgWMBase unsafe.Pointer, serial uint32) {
	marshalFlags(xdgWMBase, xdgWmBasePongOpcode, nil, wlProxyGetVersion(xdgWMBase), 0, uintptr(serial))
}

func xdgWMBaseDestroy(xdgWMBase unsafe.Pointer) {
	marshalFlags(xdgWMBase, xdgWmBaseDestroyOpcode, nil, wlProxyGetVersion(xdgWMBase), wlMarshalFlagDestroy)
}

func xdgSurfaceGetToplevel(xdgSurface unsafe.Pointer) unsafe.Pointer {
	version := wlProxyGetVersion(xdgSurface)
	return unsafe.Pointer(marshalFlags(xdgSurface, xdgSurfaceGetToplevelOpcode, &xdgToplevelInterface, version, 0, 0))
}

func xdgSurfaceAckConfigure(xdgSurface unsafe.Pointer, serial uint32) {
	marshalFlags(xdgSurface, xdgSurfaceAckConfigureOpcode, nil, wlProxyGetVersion(xdgSurface), 0, uintptr(serial))
}

func xdgSurfaceDestroy(xdgSurface unsafe.Pointer) {
	marshalFlags(xdgSurface, xdgSurfaceDestroyOpcode, nil, wlProxyGetVersion(xdgSurface), wlMarshalFlagDestroy)
}

func xdgToplevelSetTitle(toplevel unsafe.Pointer, title string) {
	buf, ptr := cString(title)
	marshalFlags(toplevel, xdgToplevelSetTitleOpcode, nil, wlProxyGetVersion(toplevel), 0, uintptr(unsafe.Pointer(ptr)))
	_ = buf
}

func xdgToplevelSetAppID(toplevel unsafe.Pointer, appID string) {
	buf, ptr := cString(appID)
	marshalFlags(toplevel, xdgToplevelSetAppIDOpcode, nil, wlProxyGetVersion(toplevel), 0, uintptr(unsafe.Pointer(ptr)))
	_ = buf
}

func xdgToplevelDestroy(toplevel unsafe.Pointer) {
	marshalFlags(toplevel, xdgToplevelDestroyOpcode, nil, wlProxyGetVersion(toplevel), wlMarshalFlagDestroy)
}

func registryGlobal(data, registry uintptr, name uint32, interfaceName uintptr, version uint32) {
	win := lookupWindow(data)
	if win == nil {
		return
	}

	switch bytePtrString((*byte)(unsafe.Pointer(interfaceName))) {
	case "wl_compositor":
		if win.compositor == nil {
			win.compositor = wlRegistryBind(unsafe.Pointer(registry), name, wlCompositorInterface, minUint32(version, maxCompositorVersion))
		}
	case "xdg_wm_base":
		if win.xdgWMBase == nil {
			win.xdgWMBase = wlRegistryBind(unsafe.Pointer(registry), name, &xdgWmBaseInterface, minUint32(version, maxXDGWMBaseVersion))
			if win.xdgWMBase != nil {
				_ = wlProxyAddListener(win.xdgWMBase, unsafe.Pointer(&xdgWMBaseListener), unsafe.Pointer(win.token))
			}
		}
	case "wl_seat":
		if win.seat == nil {
			win.seat = wlRegistryBind(unsafe.Pointer(registry), name, wlSeatInterface, minUint32(version, maxSeatVersion))
			if win.seat != nil {
				_ = wlProxyAddListener(win.seat, unsafe.Pointer(&seatListener), unsafe.Pointer(win.token))
			}
		}
	}
}

func registryGlobalRemove(data, registry uintptr, name uint32) {
	_ = data
	_ = registry
	_ = name
}

func xdgWMBasePing(data, xdgWMBase uintptr, serial uint32) {
	win := lookupWindow(data)
	if win == nil || win.closed.Load() {
		return
	}
	xdgWMBasePong(unsafe.Pointer(xdgWMBase), serial)
}

func xdgSurfaceConfigure(data, xdgSurface uintptr, serial uint32) {
	win := lookupWindow(data)
	if win == nil || win.closed.Load() {
		return
	}
	xdgSurfaceAckConfigure(unsafe.Pointer(xdgSurface), serial)
	win.configured.Store(true)
}

func xdgToplevelConfigure(data, xdgToplevel uintptr, width, height int32, states uintptr) {
	_ = xdgToplevel
	_ = states

	win := lookupWindow(data)
	if win == nil {
		return
	}
	if width > 0 {
		win.width.Store(width)
	}
	if height > 0 {
		win.height.Store(height)
	}
}

func xdgToplevelClose(data, xdgToplevel uintptr) {
	_ = xdgToplevel
	win := lookupWindow(data)
	if win == nil {
		return
	}
	win.closed.Store(true)
}

func wlSeatCapabilities(data, seat uintptr, capabilities uint32) {
	win := lookupWindow(data)
	if win == nil {
		return
	}

	if capabilities&wlSeatCapabilityPointer != 0 {
		if win.pointer == nil {
			win.pointer = wlSeatGetPointer(unsafe.Pointer(seat))
			if win.pointer != nil {
				_ = wlProxyAddListener(win.pointer, unsafe.Pointer(&pointerListener), unsafe.Pointer(win.token))
			}
		}
	} else {
		if win.pointer != nil {
			wlProxyDestroy(win.pointer)
			win.pointer = nil
		}
	}

	if capabilities&wlSeatCapabilityKeyboard != 0 {
		if win.keyboard == nil {
			win.keyboard = wlSeatGetKeyboard(unsafe.Pointer(seat))
			if win.keyboard != nil {
				_ = wlProxyAddListener(win.keyboard, unsafe.Pointer(&keyboardListener), unsafe.Pointer(win.token))
			}
		}
	} else {
		if win.keyboard != nil {
			wlProxyDestroy(win.keyboard)
			win.keyboard = nil
			win.clearPressed()
		}
	}
}

func wlSeatName(data, seat, name uintptr) {
	_ = data
	_ = seat
	_ = name
}

func wlKeyboardKeymap(data, keyboard uintptr, format uint32, fd int32, size uint32) {
	_ = keyboard

	win := lookupWindow(data)
	if win == nil {
		if fd >= 0 {
			_ = unix.Close(int(fd))
		}
		return
	}

	// xkbcommon expects keymap format 1 (text v1).
	if format != 1 {
		if fd >= 0 {
			_ = unix.Close(int(fd))
		}
		return
	}

	if err := ensureXkbcommon(); err != nil {
		if fd >= 0 {
			_ = unix.Close(int(fd))
		}
		return
	}

	if fd < 0 {
		return
	}

	buf, err := unix.Mmap(int(fd), 0, int(size), unix.PROT_READ, unix.MAP_PRIVATE)
	_ = unix.Close(int(fd))
	if err != nil {
		return
	}
	defer unix.Munmap(buf)

	// Clean up previous xkb state.
	if win.xkbState != nil {
		xkbStateUnref(win.xkbState)
		win.xkbState = nil
	}
	if win.xkbKeymap != nil {
		xkbKeymapUnref(win.xkbKeymap)
		win.xkbKeymap = nil
	}
	if win.xkbContext != nil {
		xkbContextUnref(win.xkbContext)
		win.xkbContext = nil
	}

	win.xkbContext = xkbContextNew(0)
	if win.xkbContext == nil {
		return
	}

	win.xkbKeymap = xkbKeymapNewFromString(win.xkbContext, &buf[0], 1, 0)
	if win.xkbKeymap == nil {
		return
	}

	win.xkbState = xkbStateNew(win.xkbKeymap)
}

func wlKeyboardEnter(data, keyboard, serial, surface, keys uintptr) {
	_ = keyboard
	_ = serial

	win := lookupWindow(data)
	if win == nil {
		return
	}
	if unsafe.Pointer(surface) != win.surface {
		return
	}

	win.clearPressed()
	if keys == 0 {
		return
	}

	array := (*wlArray)(unsafe.Pointer(keys))
	if array == nil || array.Data == nil || array.Size == 0 {
		return
	}

	raw := unsafe.Slice((*uint32)(array.Data), array.Size/unsafe.Sizeof(uint32(0)))
	win.pressedMu.Lock()
	defer win.pressedMu.Unlock()
	for _, key := range raw {
		keysym := uint32(0)
		if win.xkbState != nil {
			keysym = xkbStateKeyGetOneSym(win.xkbState, key+8)
		}
		if keysym != 0 {
			win.pressed[keysym] = true
		}
	}
}

func wlKeyboardLeave(data, keyboard, serial, surface uintptr) {
	_ = keyboard
	_ = serial
	_ = surface

	win := lookupWindow(data)
	if win == nil {
		return
	}
	win.clearPressed()
}

func wlKeyboardKey(data, keyboard, serial, time uintptr, key uint32, state uint32) {
	_ = keyboard
	_ = serial
	_ = time

	win := lookupWindow(data)
	if win == nil {
		return
	}

	keysym := uint32(0)
	if win.xkbState != nil {
		keysym = xkbStateKeyGetOneSym(win.xkbState, key+8)
	}
	if keysym == 0 {
		return
	}

	switch state {
	case wlKeyboardStatePressed, wlKeyboardStateRepeated:
		win.setKeyPressed(keysym, true)
	case wlKeyboardStateReleased:
		win.setKeyPressed(keysym, false)
	}
}

func wlKeyboardModifiers(data, keyboard, serial, depressed, latched, locked, group uintptr) {
	_ = keyboard
	_ = serial

	win := lookupWindow(data)
	if win == nil {
		return
	}
	if win.xkbState != nil {
		xkbStateUpdateMask(win.xkbState,
			uint32(depressed), uint32(latched), uint32(locked),
			uint32(group), 0, 0)
	}
}

func wlPointerEnter(data, pointer, serial, surface uintptr, surfaceX, surfaceY int32) {
	_ = pointer
	_ = serial
	win := lookupWindow(data)
	if win == nil {
		return
	}
	win.mouseMu.Lock()
	win.mouseX = float64(surfaceX) / 256.0 // wl_fixed_t is 1/256
	win.mouseY = float64(surfaceY) / 256.0
	win.mouseMu.Unlock()
}

func wlPointerLeave(data, pointer, serial, surface uintptr) {
	_ = pointer
	_ = serial
	_ = surface
	win := lookupWindow(data)
	if win == nil {
		return
	}
	win.mouseMu.Lock()
	win.mouseX = -1
	win.mouseY = -1
	win.mouseMu.Unlock()
}

func wlPointerMotion(data, pointer, time uintptr, surfaceX, surfaceY int32) {
	_ = pointer
	_ = time
	win := lookupWindow(data)
	if win == nil {
		return
	}
	win.mouseMu.Lock()
	win.mouseX = float64(surfaceX) / 256.0
	win.mouseY = float64(surfaceY) / 256.0
	win.mouseMu.Unlock()
}

func wlPointerButton(data, pointer, serial, time uintptr, button uint32, buttonState uint32) {
	_ = pointer
	_ = serial
	_ = time
	win := lookupWindow(data)
	if win == nil {
		return
	}
	pressed := buttonState == 1 // WL_POINTER_BUTTON_STATE_PRESSED
	win.mouseMu.Lock()
	switch button {
	case 0x110: // BTN_LEFT
		win.mouseButtons[0] = pressed
	case 0x111: // BTN_RIGHT
		win.mouseButtons[2] = pressed
	case 0x112: // BTN_MIDDLE
		win.mouseButtons[1] = pressed
	case 0x113: // BTN_SIDE
		win.mouseButtons[3] = pressed
	case 0x114: // BTN_EXTRA
		win.mouseButtons[4] = pressed
	}
	win.mouseMu.Unlock()
}

func wlPointerAxis(data, pointer, time uintptr, axis uint32, value int32) {
	_ = pointer
	_ = time
	win := lookupWindow(data)
	if win == nil {
		return
	}
	delta := float64(value) / 256.0 // wl_fixed_t
	win.mouseMu.Lock()
	if axis == 0 { // WL_POINTER_AXIS_VERTICAL_SCROLL
		win.mouseWheelY += delta
	} else if axis == 1 { // WL_POINTER_AXIS_HORIZONTAL_SCROLL
		win.mouseWheelX += delta
	}
	win.mouseMu.Unlock()
}

func (w *Window) setKeyPressed(key uint32, pressed bool) {
	w.pressedMu.Lock()
	defer w.pressedMu.Unlock()
	w.pressed[key] = pressed
}

func (w *Window) clearPressed() {
	w.pressedMu.Lock()
	defer w.pressedMu.Unlock()
	for key := range w.pressed {
		delete(w.pressed, key)
	}
}

func lookupWindow(data uintptr) *Window {
	if data == 0 {
		return nil
	}
	win, ok := waylandWindows.Load(data)
	if !ok {
		return nil
	}
	return win.(*Window)
}

func minUint32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func cString(s string) ([]byte, *byte) {
	buf := append([]byte(s), 0)
	return buf, &buf[0]
}

func bytePtrString(ptr *byte) string {
	if ptr == nil {
		return ""
	}
	raw := make([]byte, 0, 32)
	for p := uintptr(unsafe.Pointer(ptr)); ; p++ {
		b := *(*byte)(unsafe.Pointer(p))
		if b == 0 {
			return string(raw)
		}
		raw = append(raw, b)
	}
}

func appIDFromTitle(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	if title == "" {
		return "whisky-engine"
	}

	var b strings.Builder
	lastDash := false
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '.' || r == '_' || r == '-':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	appID := strings.Trim(b.String(), "-")
	if appID == "" {
		return "whisky-engine"
	}
	return appID
}
