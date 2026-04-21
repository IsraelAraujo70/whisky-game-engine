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

// TextCmd draws a string of text using the engine's built-in bitmap font.
// Pos is in the same coordinate space as other draw commands (screen/virtual).
type TextCmd struct {
	Text  string
	Pos   geom.Vec2
	Color geom.Color
	Scale float64
}

func (TextCmd) drawCmd() {}

// DrawContext is used by Drawable components to enqueue draw commands.
type DrawContext interface {
	DrawRect(worldRect geom.Rect, color geom.Color)
	DrawSprite(texture TextureID, src, dst geom.Rect, flipH, flipV bool)
	DrawText(text string, worldPos geom.Vec2, color geom.Color, scale float64)
	VirtualSize() (w, h float64)
	ViewportRect() geom.Rect
}
