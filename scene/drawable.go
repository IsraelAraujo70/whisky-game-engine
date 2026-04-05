package scene

import "github.com/IsraelAraujo70/whisky-game-engine/render"

// Drawable is implemented by components that emit draw commands.
type Drawable interface {
	Draw(node *Node, ctx render.DrawContext)
}

// Draw walks the scene tree and invokes Draw on every Drawable component.
func (s *Scene) Draw(ctx render.DrawContext) {
	if s == nil || s.Root == nil {
		return
	}
	s.Root.draw(ctx)
}

func (n *Node) draw(ctx render.DrawContext) {
	for _, component := range n.Components {
		if drawable, ok := component.(Drawable); ok {
			drawable.Draw(n, ctx)
		}
	}

	for _, child := range n.Children {
		child.draw(ctx)
	}
}
