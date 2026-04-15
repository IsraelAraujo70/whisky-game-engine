//go:build !linux && !windows && !darwin

package nativewindow

import (
	"errors"

	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

var ErrUnsupportedPlatform = errors.New("native desktop window is not supported on this platform yet")

// NewDesktop returns an explicit unsupported error on platforms without a
// native window implementation yet.
func NewDesktop(title string, width, height int, keyMap map[string]string) (platformapi.NativeWindow, error) {
	return nil, ErrUnsupportedPlatform
}
