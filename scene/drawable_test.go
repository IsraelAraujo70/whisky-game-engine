package scene

import (
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
)

type drawContextStub struct {
	rects []geom.Rect
}

func (d *drawContextStub) DrawRect(worldRect geom.Rect, color geom.Color) {
	d.rects = append(d.rects, worldRect)
}

func (d *drawContextStub) DrawSprite(texture render.TextureID, src, dst geom.Rect, flipH, flipV bool) {
}

func (d *drawContextStub) DrawText(text string, worldPos geom.Vec2, color geom.Color, scale float64) {
}

func (d *drawContextStub) VirtualSize() (w, h float64) {
	return 320, 180
}

func (d *drawContextStub) ViewportRect() geom.Rect {
	return geom.Rect{W: 320, H: 180}
}

type drawableStub struct {
	stubComponent
	draws []*Node
}

func (d *drawableStub) Draw(node *Node, ctx render.DrawContext) {
	d.draws = append(d.draws, node)
	ctx.DrawRect(geom.Rect{X: node.Position.X, Y: node.Position.Y, W: 1, H: 1}, geom.RGBA(1, 1, 1, 1))
}

func TestSceneDrawCallsDrawableComponents(t *testing.T) {
	s := New("test")
	node := NewNode("player")
	node.Position = geom.Vec2{X: 10, Y: 20}
	drawable := &drawableStub{}
	node.AddComponent(drawable)
	s.Root.AddChild(node)

	ctx := &drawContextStub{}
	s.Draw(ctx)

	if len(drawable.draws) != 1 || drawable.draws[0] != node {
		t.Fatalf("Draw called with wrong node: %+v", drawable.draws)
	}
	if len(ctx.rects) != 1 || ctx.rects[0] != (geom.Rect{X: 10, Y: 20, W: 1, H: 1}) {
		t.Fatalf("unexpected draw commands: %+v", ctx.rects)
	}
}
