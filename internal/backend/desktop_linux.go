//go:build linux

package backend

import (
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
	"github.com/IsraelAraujo70/whisky-game-engine/internal/platform/sdl3"
)

// NewDesktop returns the current Linux desktop backend used by whisky.Run.
// SDL3 remains the default implementation during the backend transition.
func NewDesktop(title string, width, height int, keyMap map[string]string) (platformapi.Backend, error) {
	return sdl3.New(title, width, height, keyMap)
}
