package physics

import (
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
)

func TestOverlapsRespectsLayersAndMasks(t *testing.T) {
	player := Collider{
		ID:     "player",
		Bounds: geom.Rect{X: 0, Y: 0, W: 8, H: 8},
		Layer:  LayerPlayer,
		Mask:   LayerWorld | LayerTrigger,
	}
	wall := Collider{
		ID:     "wall",
		Bounds: geom.Rect{X: 4, Y: 4, W: 8, H: 8},
		Layer:  LayerWorld,
		Mask:   LayerPlayer,
	}

	if !Overlaps(player, wall) {
		t.Fatal("expected colliders to overlap")
	}
}

func TestWorldQueries(t *testing.T) {
	world := NewWorld()
	world.Add(Collider{
		ID:      "pickup",
		Bounds:  geom.Rect{X: 2, Y: 2, W: 4, H: 4},
		Layer:   LayerTrigger,
		Mask:    LayerPlayer,
		Trigger: true,
	})

	if got := len(world.QueryPoint(geom.Vec2{X: 3, Y: 3}, LayerTrigger)); got != 1 {
		t.Fatalf("expected one collider from point query, got %d", got)
	}

	if got := len(world.QueryRect(geom.Rect{X: 0, Y: 0, W: 10, H: 10}, LayerTrigger)); got != 1 {
		t.Fatalf("expected one collider from rect query, got %d", got)
	}
}
