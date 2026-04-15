//go:build darwin

package nativewindow

import (
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
	"github.com/IsraelAraujo70/whisky-game-engine/internal/platform/macos"
)

// NewDesktop returns the native macOS window implementation used by the Metal backend.
func NewDesktop(title string, width, height int, keyMap map[string]string) (platformapi.NativeWindow, error) {
	return macos.New(title, width, height, keyMap)
}
