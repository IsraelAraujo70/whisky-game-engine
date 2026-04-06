package platform

// NativeWindowKind identifies the OS windowing system backing a native handle.
type NativeWindowKind string

const (
	NativeWindowKindUnknown NativeWindowKind = ""
	NativeWindowKindWin32   NativeWindowKind = "win32"
	NativeWindowKindX11     NativeWindowKind = "x11"
	NativeWindowKindWayland NativeWindowKind = "wayland"
)

// NativeWindowHandle carries the raw handles required by graphics backends to
// create swapchains or rendering surfaces.
type NativeWindowHandle struct {
	Kind NativeWindowKind
	// Display is used by APIs such as X11 and Wayland. It is nil for Win32.
	Display uintptr
	// Window is the native window handle (for example HWND or X11 Window).
	Window uintptr
	// Instance is used by APIs such as Win32 (HINSTANCE). It is zero elsewhere.
	Instance uintptr
}

// NativeWindow extends Platform with OS-window metadata required by future
// Vulkan, D3D12, and other native graphics backends.
type NativeWindow interface {
	Platform
	Size() (width, height int)
	NativeHandle() NativeWindowHandle
	Destroy() error
}
