package scene

import "github.com/IsraelAraujo70/whisky-game-engine/geom"

type Component interface {
	Start(node *Node) error
	Update(node *Node, dt float64) error
	Destroy(node *Node) error
}

type Scene struct {
	Name    string
	Root    *Node
	started bool
}

func New(name string) *Scene {
	return &Scene{
		Name: name,
		Root: NewNode("root"),
	}
}

func (s *Scene) Update(dt float64) error {
	if !s.started {
		if err := s.Root.start(); err != nil {
			return err
		}
		s.started = true
	}

	return s.Root.update(dt)
}

type Node struct {
	Name       string
	Position   geom.Vec2
	Parent     *Node
	Children   []*Node
	Components []Component
	started    bool
	destroyed  bool
}

func NewNode(name string) *Node {
	return &Node{Name: name}
}

func (n *Node) AddChild(child *Node) {
	child.Parent = n
	n.Children = append(n.Children, child)
}

func (n *Node) AddComponent(component Component) {
	n.Components = append(n.Components, component)
}

func (n *Node) WorldPosition() geom.Vec2 {
	pos := n.Position
	current := n.Parent

	for current != nil {
		pos = pos.Add(current.Position)
		current = current.Parent
	}

	return pos
}

func (n *Node) start() error {
	if n.started {
		return nil
	}

	for _, component := range n.Components {
		if err := component.Start(n); err != nil {
			return err
		}
	}

	for _, child := range n.Children {
		if err := child.start(); err != nil {
			return err
		}
	}

	n.started = true
	return nil
}

func (n *Node) update(dt float64) error {
	for _, component := range n.Components {
		if err := component.Update(n, dt); err != nil {
			return err
		}
	}

	for _, child := range n.Children {
		if !child.started {
			if err := child.start(); err != nil {
				return err
			}
		}
		if err := child.update(dt); err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) Destroy() error {
	if n.destroyed {
		return nil
	}

	for _, child := range n.Children {
		if err := child.Destroy(); err != nil {
			return err
		}
	}

	for _, component := range n.Components {
		if err := component.Destroy(n); err != nil {
			return err
		}
	}

	n.destroyed = true
	return nil
}
