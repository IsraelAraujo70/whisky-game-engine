//go:build !linux || (!amd64 && !arm64)

package x11

import (
	"errors"

	"github.com/IsraelAraujo70/whisky-game-engine/input"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
)

var ErrUnsupported = errors.New("x11 platform backend is only supported on Linux amd64/arm64")

type Window struct{}

func New(title string, width, height int, keyMap map[string]string) (*Window, error) {
	return nil, ErrUnsupported
}

func (w *Window) UpdateInput(state *input.State) {}

func (w *Window) PumpEvents() bool { return false }

func (w *Window) Size() (width, height int) { return 0, 0 }

func (w *Window) NativeHandle() platformapi.NativeWindowHandle {
	return platformapi.NativeWindowHandle{}
}

func (w *Window) Destroy() error { return nil }
