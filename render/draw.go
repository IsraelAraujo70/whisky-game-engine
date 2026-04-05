package render

import "github.com/IsraelAraujo70/whisky-game-engine/geom"

// TextureID is an opaque handle for a loaded texture. Zero is invalid.
type TextureID uint32

// DrawCmd is implemented by all draw command types.
// The unexported marker keeps the set of implementations closed.
type DrawCmd interface {
	drawCmd()
}

// SpriteCmd draws a source region from a texture into a destination rectangle.
type SpriteCmd struct {
	Texture TextureID
	Src     geom.Rect
	Dst     geom.Rect
	FlipH   bool
	FlipV   bool
}

func (SpriteCmd) drawCmd() {}

// DrawContext is used by Drawable components to enqueue draw commands.
type DrawContext interface {
	DrawRect(worldRect geom.Rect, color geom.Color)
	DrawSprite(texture TextureID, src, dst geom.Rect, flipH, flipV bool)
	VirtualSize() (w, h float64)
	ViewportRect() geom.Rect
}
