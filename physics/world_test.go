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

func TestWorldClear(t *testing.T) {
	world := NewWorld()
	world.Add(Collider{ID: "a", Bounds: geom.Rect{X: 0, Y: 0, W: 4, H: 4}, Layer: LayerWorld, Mask: LayerPlayer})
	world.Add(Collider{ID: "b", Bounds: geom.Rect{X: 8, Y: 8, W: 4, H: 4}, Layer: LayerWorld, Mask: LayerPlayer})

	world.Clear()

	if got := len(world.QueryRect(geom.Rect{X: 0, Y: 0, W: 100, H: 100}, LayerWorld)); got != 0 {
		t.Fatalf("expected 0 colliders after Clear, got %d", got)
	}
}

func TestWorldRemoveByPrefix(t *testing.T) {
	world := NewWorld()
	world.Add(Collider{ID: "tile:terrain:0,0:1x1", Bounds: geom.Rect{X: 0, Y: 0, W: 16, H: 16}, Layer: LayerWorld, Mask: LayerPlayer})
	world.Add(Collider{ID: "tile:terrain:1,0:1x1", Bounds: geom.Rect{X: 16, Y: 0, W: 16, H: 16}, Layer: LayerWorld, Mask: LayerPlayer})
	world.Add(Collider{ID: "player", Bounds: geom.Rect{X: 32, Y: 0, W: 8, H: 8}, Layer: LayerPlayer, Mask: LayerWorld})

	world.RemoveByPrefix("tile:")

	all := world.QueryRect(geom.Rect{X: 0, Y: 0, W: 100, H: 100}, LayerWorld|LayerPlayer)
	if got := len(all); got != 1 {
		t.Fatalf("expected 1 collider after RemoveByPrefix, got %d", got)
	}
	if all[0].ID != "player" {
		t.Fatalf("expected remaining collider to be 'player', got %q", all[0].ID)
	}
}

func TestColliderOneWayField(t *testing.T) {
	world := NewWorld()
	world.Add(Collider{
		ID:     "platform:oneway",
		Bounds: geom.Rect{X: 0, Y: 32, W: 64, H: 8},
		Layer:  LayerWorld,
		Mask:   LayerPlayer,
		OneWay: true,
	})
	hits := world.QueryRect(geom.Rect{X: 0, Y: 30, W: 8, H: 10}, LayerWorld)
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if !hits[0].OneWay {
		t.Fatal("expected OneWay=true on collider returned from QueryRect")
	}
}

// [M1] RemoveByPrefix with empty string must be a no-op.
func TestWorldRemoveByPrefixEmptyStringIsNoop(t *testing.T) {
	world := NewWorld()
	world.Add(Collider{ID: "a", Bounds: geom.Rect{X: 0, Y: 0, W: 4, H: 4}, Layer: LayerWorld, Mask: LayerPlayer})
	world.Add(Collider{ID: "b", Bounds: geom.Rect{X: 8, Y: 8, W: 4, H: 4}, Layer: LayerWorld, Mask: LayerPlayer})

	world.RemoveByPrefix("")

	if got := len(world.QueryRect(geom.Rect{X: 0, Y: 0, W: 100, H: 100}, LayerWorld)); got != 2 {
		t.Fatalf("expected 2 colliders after RemoveByPrefix(''), got %d", got)
	}
}
