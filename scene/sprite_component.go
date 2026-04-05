package scene

import (
	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
)

type SpriteComponent struct {
	Sheet   *render.Spritesheet
	Frame   int
	FlipH   bool
	FlipV   bool
	OffsetX float64
	OffsetY float64
	W       float64
	H       float64
}

func (sc *SpriteComponent) Start(node *Node) error {
	return nil
}

func (sc *SpriteComponent) Update(node *Node, dt float64) error {
	return nil
}

func (sc *SpriteComponent) Destroy(node *Node) error {
	return nil
}

func (sc *SpriteComponent) Draw(node *Node, ctx render.DrawContext) {
	if sc == nil || sc.Sheet == nil {
		return
	}

	pos := node.WorldPosition()
	w := float64(sc.Sheet.FrameWidth)
	h := float64(sc.Sheet.FrameHeight)
	if sc.W != 0 {
		w = sc.W
	}
	if sc.H != 0 {
		h = sc.H
	}

	ctx.DrawSprite(
		sc.Sheet.Texture,
		sc.Sheet.FrameRect(sc.Frame),
		geom.Rect{
			X: pos.X + sc.OffsetX,
			Y: pos.Y + sc.OffsetY,
			W: w,
			H: h,
		},
		sc.FlipH,
		sc.FlipV,
	)
}

var _ Component = (*SpriteComponent)(nil)
var _ Drawable = (*SpriteComponent)(nil)
