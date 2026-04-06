package platform

import (
	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/input"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
)

// Platform owns native event pumping and input collection.
type Platform interface {
	UpdateInput(state *input.State)
	PumpEvents() bool
}

// Renderer owns texture lifetime and frame presentation.
type Renderer interface {
	LoadTexture(path string) (render.TextureID, int, int, error)
	SetLogicalSize(w, h int, pixelPerfect bool) error
	DrawFrame(clearColor geom.Color, cmds []render.DrawCmd, lines []string) error
}

// Backend is the current integration point used by whisky.Run.
// The long-term direction is to keep platform and rendering separable while
// allowing transitional backends, such as SDL3, to implement both.
type Backend interface {
	Platform
	Renderer
	Destroy() error
}
