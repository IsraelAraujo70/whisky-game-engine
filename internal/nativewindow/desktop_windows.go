//go:build windows

package nativewindow

import (
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
	"github.com/IsraelAraujo70/whisky-game-engine/internal/platform/win32"
)

// NewDesktop returns the native Windows window implementation used by future
// graphics backends.
func NewDesktop(title string, width, height int, keyMap map[string]string) (platformapi.NativeWindow, error) {
	return win32.New(title, width, height, keyMap)
}
