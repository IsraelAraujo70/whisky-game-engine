//go:build linux && (amd64 || arm64)

package wayland

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	xkbOnce sync.Once
	xkbErr  error
	xkbHandle uintptr

	xkbContextNew          func(flags uint32) unsafe.Pointer
	xkbContextUnref        func(ctx unsafe.Pointer)
	xkbKeymapNewFromString func(ctx unsafe.Pointer, str *byte, format uint32, flags uint32) unsafe.Pointer
	xkbKeymapUnref         func(keymap unsafe.Pointer)
	xkbStateNew            func(keymap unsafe.Pointer) unsafe.Pointer
	xkbStateUnref          func(state unsafe.Pointer)
	xkbStateUpdateMask     func(state unsafe.Pointer, depressed, latched, locked, depressedLayout, latchedLayout, lockedLayout uint32) uint32
	xkbStateKeyGetOneSym   func(state unsafe.Pointer, key uint32) uint32
)

func ensureXkbcommon() error {
	xkbOnce.Do(func() {
		for _, name := range []string{"libxkbcommon.so.0", "libxkbcommon.so"} {
			handle, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
			if err == nil {
				xkbHandle = handle
				break
			}
		}
		if xkbHandle == 0 {
			xkbErr = fmt.Errorf("wayland: libxkbcommon.so.0 not found")
			return
		}
		purego.RegisterLibFunc(&xkbContextNew, xkbHandle, "xkb_context_new")
		purego.RegisterLibFunc(&xkbContextUnref, xkbHandle, "xkb_context_unref")
		purego.RegisterLibFunc(&xkbKeymapNewFromString, xkbHandle, "xkb_keymap_new_from_string")
		purego.RegisterLibFunc(&xkbKeymapUnref, xkbHandle, "xkb_keymap_unref")
		purego.RegisterLibFunc(&xkbStateNew, xkbHandle, "xkb_state_new")
		purego.RegisterLibFunc(&xkbStateUnref, xkbHandle, "xkb_state_unref")
		purego.RegisterLibFunc(&xkbStateUpdateMask, xkbHandle, "xkb_state_update_mask")
		purego.RegisterLibFunc(&xkbStateKeyGetOneSym, xkbHandle, "xkb_state_key_get_one_sym")
	})
	return xkbErr
}
