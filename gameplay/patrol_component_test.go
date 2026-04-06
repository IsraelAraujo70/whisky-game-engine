package gameplay

import (
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/scene"
)

func TestPatrolComponentReversesAtBounds(t *testing.T) {
	node := scene.NewNode("enemy")
	node.Position.X = 9

	patrol := &PatrolComponent{
		MinX:      4,
		MaxX:      10,
		Speed:     4,
		Direction: 1,
	}

	if err := patrol.Start(node); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}
	if err := patrol.Update(node, 1); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}

	if node.Position.X != 10 {
		t.Fatalf("expected clamped X=10, got %.2f", node.Position.X)
	}
	if patrol.Direction != -1 {
		t.Fatalf("expected direction=-1 after hitting max, got %.2f", patrol.Direction)
	}
}

func TestPatrolComponentDisabledDoesNotMove(t *testing.T) {
	node := scene.NewNode("enemy")
	node.Position.X = 6

	patrol := &PatrolComponent{
		MinX:      4,
		MaxX:      10,
		Speed:     4,
		Direction: 1,
		Disabled:  true,
	}

	if err := patrol.Update(node, 1); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if node.Position.X != 6 {
		t.Fatalf("expected X to stay at 6, got %.2f", node.Position.X)
	}
}

func TestPatrolComponentReturnsSmoothlyFromOutsideBounds(t *testing.T) {
	node := scene.NewNode("enemy")
	node.Position.X = 14

	patrol := &PatrolComponent{
		MinX:      4,
		MaxX:      10,
		Speed:     4,
		Direction: 1,
	}

	if err := patrol.Update(node, 0.25); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if node.Position.X != 13 {
		t.Fatalf("expected smooth return from out-of-bounds position, got %.2f", node.Position.X)
	}
	if patrol.Direction != -1 {
		t.Fatalf("expected direction=-1 while returning from right side, got %.2f", patrol.Direction)
	}
}
