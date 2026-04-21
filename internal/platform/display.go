package platform

import "errors"

// ErrNotSupported is returned when a display operation is not available on the
// current platform or backend.
var ErrNotSupported = errors.New("display: operation not supported")

// WindowMode describes how the window is presented.
type WindowMode int

const (
	// WindowModeWindowed is a standard decorated window.
	WindowModeWindowed WindowMode = iota
	// WindowModeBorderless is a borderless fullscreen window (sometimes called
	// "windowed fullscreen" or "borderless fullscreen").
	WindowModeBorderless
	// WindowModeFullscreen is exclusive fullscreen (may change display mode).
	WindowModeFullscreen
)

// DisplayMode describes a supported resolution and refresh rate.
type DisplayMode struct {
	Width     int
	Height    int
	RefreshHz int
}

// MonitorInfo describes a connected display.
type MonitorInfo struct {
	Name      string
	Index     int
	Modes     []DisplayMode
	IsPrimary bool
}

// DisplayController is an optional interface that NativeWindow implementations
// may satisfy to support runtime display configuration (resolution, fullscreen,
// monitor selection). Consumers should use a type assertion to check support.
type DisplayController interface {
	SetWindowSize(width, height int) error
	SetWindowMode(mode WindowMode) error
	Monitors() ([]MonitorInfo, error)
	MoveToMonitor(index int) error
}
