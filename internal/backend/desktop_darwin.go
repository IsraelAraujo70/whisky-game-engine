//go:build darwin

package backend

import platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"

// NewDesktop creates the native macOS desktop backend wired to the Metal runtime stack.
func NewDesktop(title string, width, height int, keyMap map[string]string) (platformapi.Backend, error) {
	return metalDesktopBackendFactory(title, width, height, keyMap)
}
