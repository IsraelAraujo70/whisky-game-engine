package render

import "github.com/IsraelAraujo70/whisky-game-engine/geom"

type Spritesheet struct {
	Texture     TextureID
	FrameWidth  int
	FrameHeight int
	Columns     int
	Rows        int
}

func (s *Spritesheet) FrameCount() int {
	if s == nil || s.Columns <= 0 || s.Rows <= 0 {
		return 0
	}
	return s.Columns * s.Rows
}

// FrameRect returns the source rectangle in pixels for frame i.
// Indices are clamped into the valid frame range.
func (s *Spritesheet) FrameRect(i int) geom.Rect {
	count := s.FrameCount()
	if count == 0 || s.FrameWidth <= 0 || s.FrameHeight <= 0 {
		return geom.Rect{}
	}
	if i < 0 {
		i = 0
	}
	if i >= count {
		i = count - 1
	}

	col := i % s.Columns
	row := i / s.Columns

	return geom.Rect{
		X: float64(col * s.FrameWidth),
		Y: float64(row * s.FrameHeight),
		W: float64(s.FrameWidth),
		H: float64(s.FrameHeight),
	}
}
