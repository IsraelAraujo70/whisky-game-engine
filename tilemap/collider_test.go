package tilemap

import (
	"strings"
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/physics"
)

func TestGreedyMergeFloor(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true})

	m := New(ts, 20, 12)
	m.AddLayer("terrain")
	// Fill bottom row (20 solid tiles).
	m.FillRow("terrain", 0, 11, 20, 1)

	colliders := GenerateColliders(m, geom.Vec2{}, DefaultColliderConfig())

	// All 20 tiles should merge into a single collider.
	if len(colliders) != 1 {
		t.Fatalf("expected 1 merged collider, got %d", len(colliders))
	}
	c := colliders[0]
	if c.Bounds.W != 320 || c.Bounds.H != 16 {
		t.Fatalf("expected collider 320x16, got %.0fx%.0f", c.Bounds.W, c.Bounds.H)
	}
	if c.Bounds.Y != 176 {
		t.Fatalf("expected Y=176, got %.0f", c.Bounds.Y)
	}
}

func TestGreedyMergeBlock(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true})

	m := New(ts, 10, 10)
	m.AddLayer("terrain")
	m.FillRect("terrain", 2, 3, 4, 3, 1)

	colliders := GenerateColliders(m, geom.Vec2{}, DefaultColliderConfig())

	if len(colliders) != 1 {
		t.Fatalf("expected 1 merged collider for 4x3 block, got %d", len(colliders))
	}
	c := colliders[0]
	if c.Bounds.W != 64 || c.Bounds.H != 48 {
		t.Fatalf("expected 64x48, got %.0fx%.0f", c.Bounds.W, c.Bounds.H)
	}
}

func TestGreedyMergeSeparateBlocks(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true})

	m := New(ts, 10, 10)
	m.AddLayer("terrain")
	// Two separated platforms.
	m.FillRow("terrain", 0, 5, 3, 1)
	m.FillRow("terrain", 6, 5, 3, 1)

	colliders := GenerateColliders(m, geom.Vec2{}, DefaultColliderConfig())

	if len(colliders) != 2 {
		t.Fatalf("expected 2 colliders for separated blocks, got %d", len(colliders))
	}
}

func TestOneWayTilesNotMerged(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true, OneWay: true})

	m := New(ts, 10, 10)
	m.AddLayer("terrain")
	m.FillRow("terrain", 0, 5, 5, 1)

	colliders := GenerateColliders(m, geom.Vec2{}, DefaultColliderConfig())

	// Each one-way tile should be its own collider.
	if len(colliders) != 5 {
		t.Fatalf("expected 5 individual one-way colliders, got %d", len(colliders))
	}
	for _, c := range colliders {
		if !strings.HasSuffix(c.ID, ":oneway") {
			t.Fatalf("expected :oneway suffix, got %q", c.ID)
		}
	}
}

func TestTriggerTilesNotMerged(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Trigger: true})

	m := New(ts, 5, 5)
	m.AddLayer("triggers")
	m.SetTile("triggers", 1, 1, 1)
	m.SetTile("triggers", 2, 1, 1)

	colliders := GenerateColliders(m, geom.Vec2{}, DefaultColliderConfig())

	if len(colliders) != 2 {
		t.Fatalf("expected 2 trigger colliders, got %d", len(colliders))
	}
	for _, c := range colliders {
		if !c.Trigger {
			t.Fatal("trigger collider should have Trigger=true")
		}
		if !strings.HasSuffix(c.ID, ":trigger") {
			t.Fatalf("expected :trigger suffix, got %q", c.ID)
		}
	}
}

func TestColliderOffset(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true})

	m := New(ts, 5, 5)
	m.AddLayer("terrain")
	m.SetTile("terrain", 0, 0, 1)

	offset := geom.Vec2{X: 100, Y: 200}
	colliders := GenerateColliders(m, offset, DefaultColliderConfig())

	if len(colliders) != 1 {
		t.Fatalf("expected 1 collider, got %d", len(colliders))
	}
	if colliders[0].Bounds.X != 100 || colliders[0].Bounds.Y != 200 {
		t.Fatalf("expected offset (100,200), got (%.0f,%.0f)", colliders[0].Bounds.X, colliders[0].Bounds.Y)
	}
}

func TestColliderLayerAndMask(t *testing.T) {
	cfg := DefaultColliderConfig()

	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true})
	ts.SetProperties(2, TileProperties{Trigger: true})

	m := New(ts, 5, 5)
	m.AddLayer("terrain")
	m.SetTile("terrain", 0, 0, 1) // solid
	m.SetTile("terrain", 1, 0, 2) // trigger

	colliders := GenerateColliders(m, geom.Vec2{}, cfg)

	if len(colliders) != 2 {
		t.Fatalf("expected 2 colliders, got %d", len(colliders))
	}

	// Find solid and trigger.
	var solid, trigger *physics.Collider
	for i := range colliders {
		if colliders[i].Trigger {
			trigger = &colliders[i]
		} else {
			solid = &colliders[i]
		}
	}

	if solid == nil || trigger == nil {
		t.Fatal("expected one solid and one trigger collider")
	}
	if solid.Layer != physics.LayerWorld || solid.Mask != physics.LayerPlayer {
		t.Fatal("solid collider has wrong layer/mask")
	}
	if trigger.Layer != physics.LayerTrigger || trigger.Mask != physics.LayerPlayer {
		t.Fatal("trigger collider has wrong layer/mask")
	}
}

func TestAddToWorld(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true})

	m := New(ts, 10, 10)
	m.AddLayer("terrain")
	m.FillRow("terrain", 0, 9, 10, 1)

	world := physics.NewWorld()
	AddToWorld(m, world, geom.Vec2{}, DefaultColliderConfig())

	// Should be queryable.
	hits := world.QueryRect(geom.Rect{X: 0, Y: 144, W: 160, H: 16}, physics.LayerWorld)
	if len(hits) != 1 {
		t.Fatalf("expected 1 collider in world query, got %d", len(hits))
	}
}

func TestEmptyMapProducesNoColliders(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	m := New(ts, 5, 5)
	m.AddLayer("terrain")

	colliders := GenerateColliders(m, geom.Vec2{}, DefaultColliderConfig())
	if len(colliders) != 0 {
		t.Fatalf("expected 0 colliders for empty map, got %d", len(colliders))
	}
}

func TestMixedSolidAndOneWay(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true})
	ts.SetProperties(2, TileProperties{Solid: true, OneWay: true})

	m := New(ts, 6, 1)
	m.AddLayer("terrain")
	// [solid, solid, oneway, solid, solid, solid]
	m.SetTile("terrain", 0, 0, 1)
	m.SetTile("terrain", 1, 0, 1)
	m.SetTile("terrain", 2, 0, 2) // one-way breaks the merge
	m.SetTile("terrain", 3, 0, 1)
	m.SetTile("terrain", 4, 0, 1)
	m.SetTile("terrain", 5, 0, 1)

	colliders := GenerateColliders(m, geom.Vec2{}, DefaultColliderConfig())

	// Should be: 1 merged (2 tiles) + 1 one-way + 1 merged (3 tiles) = 3 colliders.
	if len(colliders) != 3 {
		t.Fatalf("expected 3 colliders (2 merged + 1 oneway), got %d", len(colliders))
	}
}
