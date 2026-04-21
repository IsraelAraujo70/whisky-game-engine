//go:build !linux || (!amd64 && !arm64)

package linux

import "github.com/IsraelAraujo70/whisky-game-engine/input"

// GamepadPoller is a no-op stub on non-Linux platforms.
type GamepadPoller struct{}

// NewGamepadPoller returns a no-op stub.
func NewGamepadPoller() *GamepadPoller { return &GamepadPoller{} }

// Poll is a no-op on unsupported platforms.
func (p *GamepadPoller) Poll(state *input.State) {}
