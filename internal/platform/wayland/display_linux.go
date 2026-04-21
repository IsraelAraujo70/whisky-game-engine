//go:build linux && (amd64 || arm64)

package wayland

import (
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

// Ensure Window implements DisplayController.
var _ platformapi.DisplayController = (*Window)(nil)

const (
	// xdg_toplevel opcodes for size control.
	xdgToplevelSetMaxSizeOpcode = 7
	xdgToplevelSetMinSizeOpcode = 8
	// xdg_toplevel opcodes for fullscreen control.
	xdgToplevelSetFullscreenOpcode   = 11
	xdgToplevelUnsetFullscreenOpcode = 12
)

// SetWindowSize requests a new window size from the compositor.
// On Wayland the compositor ultimately decides the size, but we strongly hint
// via min_size == max_size which most compositors honor.
func (w *Window) SetWindowSize(width, height int) error {
	if w == nil || w.toplevel == nil || w.surface == nil {
		return platformapi.ErrNotSupported
	}

	// Set min and max size to the same value — this is the standard Wayland
	// idiom for requesting a fixed window size. The compositor will send a
	// configure event with the new dimensions. We keep the constraints in place
	// (no immediate reset) so the compositor has time to process them.
	version := wlProxyGetVersion(w.toplevel)
	marshalFlags(w.toplevel, xdgToplevelSetMinSizeOpcode, nil, version, 0,
		uintptr(width), uintptr(height))
	marshalFlags(w.toplevel, xdgToplevelSetMaxSizeOpcode, nil, version, 0,
		uintptr(width), uintptr(height))

	wlSurfaceCommit(w.surface)
	_ = wlDisplayFlush(w.display)

	// Roundtrip ensures the compositor processes the size change synchronously
	// before we return. Without this, the configure event may arrive too late.
	wlDisplayRoundtrip(w.display)

	w.width.Store(int32(width))
	w.height.Store(int32(height))
	return nil
}

// SetWindowMode sets the window to windowed or fullscreen.
func (w *Window) SetWindowMode(mode platformapi.WindowMode) error {
	if w == nil || w.toplevel == nil || w.surface == nil {
		return platformapi.ErrNotSupported
	}

	version := wlProxyGetVersion(w.toplevel)

	switch mode {
	case platformapi.WindowModeWindowed:
		// unset_fullscreen takes no arguments.
		marshalFlags(w.toplevel, xdgToplevelUnsetFullscreenOpcode, nil, version, 0)
	case platformapi.WindowModeBorderless, platformapi.WindowModeFullscreen:
		// set_fullscreen takes an optional output (nil = compositor choice).
		marshalFlags(w.toplevel, xdgToplevelSetFullscreenOpcode, nil, version, 0, 0)
	default:
		return platformapi.ErrNotSupported
	}

	wlSurfaceCommit(w.surface)
	_ = wlDisplayFlush(w.display)

	// Roundtrip ensures the compositor processes the mode change synchronously.
	wlDisplayRoundtrip(w.display)

	return nil
}

// Monitors returns a list of available monitors.
// Full Wayland output enumeration requires wl_output binding during registry.
// For now we return a default list with common resolutions.
func (w *Window) Monitors() ([]platformapi.MonitorInfo, error) {
	if w == nil {
		return nil, platformapi.ErrNotSupported
	}

	// Return a default single-monitor entry with common resolutions.
	// A full implementation would collect wl_output events from the registry.
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

// MoveToMonitor moves the window to the specified monitor.
// On Wayland, this is limited as the compositor controls window placement.
// We can use set_fullscreen with a specific output for fullscreen migration.
func (w *Window) MoveToMonitor(index int) error {
	if w == nil {
		return platformapi.ErrNotSupported
	}
	// Wayland doesn't allow arbitrary window positioning.
	// This is a known limitation of the protocol.
	return platformapi.ErrNotSupported
}
