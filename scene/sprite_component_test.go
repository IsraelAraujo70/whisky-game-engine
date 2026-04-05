package scene

import (
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
)

type spriteCall struct {
	texture render.TextureID
	src     geom.Rect
	dst     geom.Rect
	flipH   bool
	flipV   bool
}

type spriteContextStub struct {
	sprite spriteCall
	called bool
}

func (s *spriteContextStub) DrawRect(worldRect geom.Rect, color geom.Color) {
}

func (s *spriteContextStub) DrawSprite(texture render.TextureID, src, dst geom.Rect, flipH, flipV bool) {
	s.called = true
	s.sprite = spriteCall{
		texture: texture,
		src:     src,
		dst:     dst,
		flipH:   flipH,
		flipV:   flipV,
	}
}

func (s *spriteContextStub) VirtualSize() (w, h float64) {
	return 320, 180
}

func (s *spriteContextStub) ViewportRect() geom.Rect {
	return geom.Rect{W: 320, H: 180}
}

func TestSpriteComponentDraw(t *testing.T) {
	parent := NewNode("parent")
	parent.Position = geom.Vec2{X: 10, Y: 20}
	node := NewNode("player")
	node.Position = geom.Vec2{X: 3, Y: 4}
	parent.AddChild(node)

	component := &SpriteComponent{
		Sheet: &render.Spritesheet{
			Texture:     9,
			FrameWidth:  8,
			FrameHeight: 16,
			Columns:     2,
			Rows:        1,
		},
		Frame:   1,
		FlipH:   true,
		OffsetX: 1,
		OffsetY: 2,
	}

	ctx := &spriteContextStub{}
	component.Draw(node, ctx)

	if !ctx.called {
		t.Fatal("expected DrawSprite to be called")
	}

	want := spriteCall{
		texture: 9,
		src:     geom.Rect{X: 8, Y: 0, W: 8, H: 16},
		dst:     geom.Rect{X: 14, Y: 26, W: 8, H: 16},
		flipH:   true,
	}
	if ctx.sprite != want {
		t.Fatalf("unexpected sprite draw: got %#v want %#v", ctx.sprite, want)
	}
}
