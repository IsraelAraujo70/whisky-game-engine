package render

import "github.com/IsraelAraujo70/whisky-game-engine/geom"

// FillRect represents a colored filled rectangle to draw on screen.
type FillRect struct {
	Rect  geom.Rect
	Color geom.Color
}

func (FillRect) drawCmd() {}

// Camera2D provides world-to-screen coordinate transformation for 2D rendering.
// Position represents the center of the camera in world coordinates.
type Camera2D struct {
	Position geom.Vec2
}

// ViewportRect returns the camera's visible area in world coordinates.
func (c *Camera2D) ViewportRect(virtualW, virtualH float64) geom.Rect {
	return geom.Rect{
		X: c.Position.X - virtualW/2,
		Y: c.Position.Y - virtualH/2,
		W: virtualW,
		H: virtualH,
	}
}

// WorldToScreen converts a world position to screen (virtual) coordinates.
func (c *Camera2D) WorldToScreen(worldPos geom.Vec2, virtualW, virtualH float64) geom.Vec2 {
	return geom.Vec2{
		X: worldPos.X - c.Position.X + virtualW/2,
		Y: worldPos.Y - c.Position.Y + virtualH/2,
	}
}
