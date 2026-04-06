package gameplay

import (
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/scene"
)

func TestTargetComponentChasesVisibleTarget(t *testing.T) {
	enemy := scene.NewNode("enemy")
	target := scene.NewNode("player")
	target.Position = geom.Vec2{X: 30, Y: 0}

	patrol := &PatrolComponent{MinX: 0, MaxX: 40, Speed: 10}
	chase := &TargetComponent{
		TargetBox: Box{
			Node: target,
			W:    8,
			H:    16,
		},
		Sight: Box{
			Node: enemy,
			Offset: geom.Vec2{
				X: -16,
				Y: -8,
			},
			W: 48,
			H: 32,
		},
		Speed:          20,
		StopDistance:   0,
		HorizontalOnly: true,
		Patrol:         patrol,
	}

	if err := chase.Update(enemy, 1); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if !chase.Chasing {
		t.Fatal("expected enemy to chase visible target")
	}
	if !patrol.Disabled {
		t.Fatal("expected patrol to be disabled while chasing")
	}
	if enemy.Position.X <= 0 {
		t.Fatalf("expected enemy to move toward target, got X=%.2f", enemy.Position.X)
	}
}

func TestTargetComponentStopsChasingOutsideSight(t *testing.T) {
	enemy := scene.NewNode("enemy")
	target := scene.NewNode("player")
	target.Position = geom.Vec2{X: 100, Y: 0}

	patrol := &PatrolComponent{MinX: 0, MaxX: 40, Speed: 10, Disabled: true}
	chase := &TargetComponent{
		TargetBox: Box{
			Node: target,
			W:    8,
			H:    16,
		},
		Sight: Box{
			Node: enemy,
			W:    32,
			H:    24,
		},
		Speed:          20,
		HorizontalOnly: true,
		Patrol:         patrol,
	}

	if err := chase.Update(enemy, 1); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if chase.Chasing {
		t.Fatal("expected enemy not to chase target outside sight")
	}
	if patrol.Disabled {
		t.Fatal("expected patrol to re-enable when target is lost")
	}
}
