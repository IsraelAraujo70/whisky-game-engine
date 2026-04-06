//go:build linux

package wayland

import (
	"errors"

	"github.com/IsraelAraujo70/whisky-game-engine/input"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

var ErrNotImplemented = errors.New("wayland platform backend is not implemented yet")

type Window struct {
	width  int
	height int
}

func New(title string, width, height int, keyMap map[string]string) (*Window, error) {
	return nil, ErrNotImplemented
}

func (w *Window) UpdateInput(state *input.State) {}

func (w *Window) PumpEvents() bool { return false }

func (w *Window) Size() (width, height int) {
	if w == nil {
		return 0, 0
	}
	return w.width, w.height
}

func (w *Window) NativeHandle() platformapi.NativeWindowHandle {
	return platformapi.NativeWindowHandle{
		Kind: platformapi.NativeWindowKindWayland,
	}
}

func (w *Window) Destroy() error { return nil }
