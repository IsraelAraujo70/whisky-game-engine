package scene

import (
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
)

type stubComponent struct {
	starts   int
	updates  int
	destroys int
}

func (s *stubComponent) Start(node *Node) error {
	s.starts++
	return nil
}

func (s *stubComponent) Update(node *Node, dt float64) error {
	s.updates++
	return nil
}

func (s *stubComponent) Destroy(node *Node) error {
	s.destroys++
	return nil
}

func TestWorldPosition(t *testing.T) {
	root := NewNode("root")
	root.Position = geom.Vec2{X: 2, Y: 1}
	child := NewNode("child")
	child.Position = geom.Vec2{X: 4, Y: 3}
	root.AddChild(child)

	if got := child.WorldPosition(); got != (geom.Vec2{X: 6, Y: 4}) {
		t.Fatalf("unexpected world position: %#v", got)
	}
}

func TestSceneLifecycle(t *testing.T) {
	s := New("test")
	component := &stubComponent{}
	node := NewNode("player")
	node.AddComponent(component)
	s.Root.AddChild(node)

	if err := s.Update(1.0 / 60.0); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}

	if component.starts != 1 || component.updates != 1 {
		t.Fatalf("unexpected lifecycle counters: %+v", component)
	}

	if err := s.Root.Destroy(); err != nil {
		t.Fatalf("unexpected destroy error: %v", err)
	}

	if component.destroys != 1 {
		t.Fatalf("expected destroy to run once, got %d", component.destroys)
	}
}
