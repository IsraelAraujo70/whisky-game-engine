package platform

// NativeWindowKind identifies the OS windowing system backing a native handle.
type NativeWindowKind string

const (
	NativeWindowKindUnknown NativeWindowKind = ""
	NativeWindowKindWin32   NativeWindowKind = "win32"
	NativeWindowKindX11     NativeWindowKind = "x11"
	NativeWindowKindWayland NativeWindowKind = "wayland"
	NativeWindowKindCocoa   NativeWindowKind = "cocoa"
)

// NativeWindowHandle carries the raw handles required by graphics backends to
// create swapchains or rendering surfaces.
type NativeWindowHandle struct {
	Kind NativeWindowKind
	// Display is used by APIs such as X11 and Wayland. It is nil for Win32 and Cocoa.
	Display uintptr
	// Window is the native top-level window handle (for example HWND, X11 Window, or NSWindow).
	Window uintptr
	// View is used by view-backed systems such as Cocoa.
	View uintptr
	// Layer is used by layer-backed systems such as CAMetalLayer.
	Layer uintptr
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
