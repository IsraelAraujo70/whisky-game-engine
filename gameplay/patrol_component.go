package gameplay

import "github.com/IsraelAraujo70/whisky-game-engine/scene"

// PatrolComponent moves a node back and forth between MinX and MaxX.
type PatrolComponent struct {
	MinX      float64
	MaxX      float64
	Speed     float64
	Direction float64
	Disabled  bool
}

func (p *PatrolComponent) Start(node *scene.Node) error {
	if p == nil {
		return nil
	}
	if p.MaxX < p.MinX {
		p.MinX, p.MaxX = p.MaxX, p.MinX
	}
	if p.Direction == 0 {
		p.Direction = 1
	}
	if node.Position.X < p.MinX {
		node.Position.X = p.MinX
	}
	if node.Position.X > p.MaxX {
		node.Position.X = p.MaxX
	}
	return nil
}

func (p *PatrolComponent) Update(node *scene.Node, dt float64) error {
	if p == nil || p.Disabled || p.Speed == 0 {
		return nil
	}
	if p.Direction == 0 {
		p.Direction = 1
	}
	step := p.Speed * dt

	// If chase or another system moved the node outside patrol bounds, walk it
	// back to the route instead of snapping directly to MinX/MaxX.
	if node.Position.X < p.MinX {
		node.Position.X += step
		if node.Position.X >= p.MinX {
			node.Position.X = p.MinX
		}
		p.Direction = 1
		return nil
	}
	if node.Position.X > p.MaxX {
		node.Position.X -= step
		if node.Position.X <= p.MaxX {
			node.Position.X = p.MaxX
		}
		p.Direction = -1
		return nil
	}

	node.Position.X += p.Direction * step
	if node.Position.X <= p.MinX {
		node.Position.X = p.MinX
		p.Direction = 1
	}
	if node.Position.X >= p.MaxX {
		node.Position.X = p.MaxX
		p.Direction = -1
	}
	return nil
}

func (p *PatrolComponent) Destroy(node *scene.Node) error {
	return nil
}

func (p *PatrolComponent) FacingRight() bool {
	return p == nil || p.Direction >= 0
}

var _ scene.Component = (*PatrolComponent)(nil)
