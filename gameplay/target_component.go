package gameplay

import (
	"math"

	"github.com/IsraelAraujo70/whisky-game-engine/scene"
)

// TargetComponent acquires a target inside Sight and moves toward it.
// It can optionally disable a PatrolComponent while chasing.
type TargetComponent struct {
	TargetBox      Box
	Sight          Box
	Speed          float64
	StopDistance   float64
	HorizontalOnly bool
	Disabled       bool
	Patrol         *PatrolComponent
	Chasing        bool
}

func (t *TargetComponent) Start(node *scene.Node) error {
	return nil
}

func (t *TargetComponent) Update(node *scene.Node, dt float64) error {
	if t == nil || t.Disabled || t.TargetBox.Node == nil || t.Speed == 0 {
		t.Chasing = false
		if t.Patrol != nil {
			t.Patrol.Disabled = false
		}
		return nil
	}

	t.Chasing = t.Sight.Rect().Intersects(t.TargetBox.Rect())
	if t.Patrol != nil {
		t.Patrol.Disabled = t.Chasing
	}
	if !t.Chasing {
		return nil
	}

	sightRect := t.Sight.Rect()
	targetRect := t.TargetBox.Rect()
	targetCenterX := targetRect.X + (targetRect.W / 2)
	targetCenterY := targetRect.Y + (targetRect.H / 2)
	nodeCenterX := sightRect.X + (sightRect.W / 2)
	nodeCenterY := sightRect.Y + (sightRect.H / 2)

	dx := targetCenterX - nodeCenterX
	if math.Abs(dx) > t.StopDistance {
		if dx > 0 {
			node.Position.X += t.Speed * dt
		} else {
			node.Position.X -= t.Speed * dt
		}
	}

	if t.HorizontalOnly {
		return nil
	}

	dy := targetCenterY - nodeCenterY
	if math.Abs(dy) > t.StopDistance {
		if dy > 0 {
			node.Position.Y += t.Speed * dt
		} else {
			node.Position.Y -= t.Speed * dt
		}
	}

	return nil
}

func (t *TargetComponent) Destroy(node *scene.Node) error {
	return nil
}

var _ scene.Component = (*TargetComponent)(nil)
