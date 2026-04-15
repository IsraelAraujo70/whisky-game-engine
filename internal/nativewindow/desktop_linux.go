//go:build linux

package nativewindow

import (
	"errors"
	"fmt"
	"os"
	"strings"

	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
	"github.com/IsraelAraujo70/whisky-game-engine/internal/platform/wayland"
	"github.com/IsraelAraujo70/whisky-game-engine/internal/platform/x11"
)

const (
	windowSystemAuto    = "auto"
	windowSystemX11     = "x11"
	windowSystemWayland = "wayland"
)

var ErrUnsupportedWindowSystem = errors.New("unsupported window system")

// NewDesktop returns the native Linux window implementation used by future
// graphics backends. The selection order is:
// 1. WHISKY_WINDOW_SYSTEM when explicitly set to x11 or wayland
// 2. wayland session detection with x11 fallback
// 3. x11 when DISPLAY is present
func NewDesktop(title string, width, height int, keyMap map[string]string) (platformapi.NativeWindow, error) {
	system := selectedLinuxWindowSystem()

	switch system {
	case windowSystemWayland:
		window, err := wayland.New(title, width, height, keyMap)
		if err == nil {
			return window, nil
		}
		if shouldFallbackToX11(err) {
			return x11.New(title, width, height, keyMap)
		}
		return nil, err
	case windowSystemX11:
		return x11.New(title, width, height, keyMap)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedWindowSystem, system)
	}
}

func selectedLinuxWindowSystem() string {
	explicit := strings.ToLower(strings.TrimSpace(os.Getenv("WHISKY_WINDOW_SYSTEM")))
	switch explicit {
	case "", windowSystemAuto:
		return detectLinuxWindowSystem()
	case windowSystemX11, windowSystemWayland:
		return explicit
	default:
		return explicit
	}
}

func detectLinuxWindowSystem() string {
	sessionType := strings.ToLower(strings.TrimSpace(os.Getenv("XDG_SESSION_TYPE")))
	hasWayland := strings.TrimSpace(os.Getenv("WAYLAND_DISPLAY")) != ""
	hasX11 := strings.TrimSpace(os.Getenv("DISPLAY")) != ""

	if sessionType == windowSystemWayland && hasWayland {
		return windowSystemWayland
	}
	if hasX11 {
		return windowSystemX11
	}
	if hasWayland {
		return windowSystemWayland
	}
	return windowSystemX11
}

func shouldFallbackToX11(err error) bool {
	return errors.Is(err, wayland.ErrNotImplemented) && strings.TrimSpace(os.Getenv("DISPLAY")) != ""
}
