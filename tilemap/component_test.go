package tilemap

import (
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/physics"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
	"github.com/IsraelAraujo70/whisky-game-engine/scene"
)

type tileDrawContextStub struct {
	sprites []tileSpriteCall
}

type tileSpriteCall struct {
	texture render.TextureID
	src     geom.Rect
	dst     geom.Rect
}

func (d *tileDrawContextStub) DrawRect(worldRect geom.Rect, color geom.Color) {
}

func (d *tileDrawContextStub) DrawSprite(texture render.TextureID, src, dst geom.Rect, flipH, flipV bool) {
	d.sprites = append(d.sprites, tileSpriteCall{
		texture: texture,
		src:     src,
		dst:     dst,
	})
}

func (d *tileDrawContextStub) VirtualSize() (w, h float64) {
	return 32, 32
}

func (d *tileDrawContextStub) ViewportRect() geom.Rect {
	return geom.Rect{X: 0, Y: 0, W: 32, H: 32}
}

func TestTileMapComponentStart(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true})

	m := New(ts, 5, 5)
	m.AddLayer("terrain")
	m.FillRow("terrain", 0, 4, 5, 1)

	world := physics.NewWorld()
	comp := &TileMapComponent{
		Map:   m,
		World: world,
	}

	node := scene.NewNode("level")
	if err := comp.Start(node); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Should have added colliders to the world.
	hits := world.QueryRect(geom.Rect{X: 0, Y: 64, W: 80, H: 16}, physics.LayerWorld)
	if len(hits) == 0 {
		t.Fatal("expected colliders in world after Start")
	}
}

func TestTileMapComponentStartWithOffset(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true})

	m := New(ts, 3, 3)
	m.AddLayer("terrain")
	m.SetTile("terrain", 0, 0, 1)

	world := physics.NewWorld()
	comp := &TileMapComponent{
		Map:   m,
		World: world,
	}

	// Parent node shifts the level.
	parent := scene.NewNode("world")
	parent.Position = geom.Vec2{X: 100, Y: 50}
	child := scene.NewNode("level")
	parent.AddChild(child)

	if err := comp.Start(child); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Collider should be offset by parent position.
	hits := world.QueryPoint(geom.Vec2{X: 108, Y: 58}, physics.LayerWorld)
	if len(hits) != 1 {
		t.Fatalf("expected 1 collider at offset position, got %d", len(hits))
	}
}

func TestTileMapComponentMarkDirty(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true})

	m := New(ts, 5, 5)
	m.AddLayer("terrain")
	m.FillRow("terrain", 0, 4, 5, 1)

	world := physics.NewWorld()
	comp := &TileMapComponent{
		Map:   m,
		World: world,
	}

	node := scene.NewNode("level")
	if err := comp.Start(node); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Modify the map and mark dirty.
	m.FillRow("terrain", 0, 3, 5, 1)
	comp.MarkDirty()

	if err := comp.Update(node, 0.016); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Should now have colliders for both rows.
	hits := world.QueryRect(geom.Rect{X: 0, Y: 48, W: 80, H: 32}, physics.LayerWorld)
	if len(hits) == 0 {
		t.Fatal("expected colliders after dirty rebuild")
	}
}

func TestTileMapComponentDestroy(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true})

	m := New(ts, 5, 5)
	m.AddLayer("terrain")
	m.FillRow("terrain", 0, 4, 5, 1)

	world := physics.NewWorld()
	// Add a non-tile collider that should survive.
	world.Add(physics.Collider{
		ID:     "player",
		Bounds: geom.Rect{X: 0, Y: 0, W: 8, H: 8},
		Layer:  physics.LayerPlayer,
		Mask:   physics.LayerWorld,
	})

	comp := &TileMapComponent{
		Map:   m,
		World: world,
	}

	node := scene.NewNode("level")
	if err := comp.Start(node); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := comp.Destroy(node); err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	// Tile colliders should be gone.
	hits := world.QueryRect(geom.Rect{X: 0, Y: 64, W: 80, H: 16}, physics.LayerWorld)
	if len(hits) != 0 {
		t.Fatalf("expected 0 tile colliders after Destroy, got %d", len(hits))
	}

	// Player collider should survive.
	hits = world.QueryPoint(geom.Vec2{X: 4, Y: 4}, physics.LayerPlayer)
	if len(hits) != 1 {
		t.Fatalf("expected player collider to survive, got %d", len(hits))
	}
}

func TestTileMapComponentUpdateNoop(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	m := New(ts, 3, 3)
	m.AddLayer("terrain")

	world := physics.NewWorld()
	comp := &TileMapComponent{Map: m, World: world}

	node := scene.NewNode("level")
	if err := comp.Start(node); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Update without marking dirty should be a no-op.
	if err := comp.Update(node, 0.016); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
}

// [C1] Destroy before Start must be safe (no-op).
func TestTileMapComponentDestroyBeforeStart(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	m := New(ts, 3, 3)
	m.AddLayer("terrain")

	world := physics.NewWorld()
	// Add an unrelated collider.
	world.Add(physics.Collider{
		ID:     "player",
		Bounds: geom.Rect{X: 0, Y: 0, W: 8, H: 8},
		Layer:  physics.LayerPlayer,
		Mask:   physics.LayerWorld,
	})

	comp := &TileMapComponent{Map: m, World: world}
	node := scene.NewNode("level")

	// Destroy before Start should be a no-op, NOT wipe the world.
	if err := comp.Destroy(node); err != nil {
		t.Fatalf("Destroy failed: %v", err)
	}

	hits := world.QueryPoint(geom.Vec2{X: 4, Y: 4}, physics.LayerPlayer)
	if len(hits) != 1 {
		t.Fatalf("expected player collider to survive Destroy-before-Start, got %d", len(hits))
	}
}

// [C3] Two TileMapComponents with different node names must not interfere.
func TestTwoTileMapComponentsIndependent(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true})

	m1 := New(ts, 3, 3)
	m1.AddLayer("terrain")
	m1.SetTile("terrain", 0, 0, 1)

	m2 := New(ts, 3, 3)
	m2.AddLayer("terrain")
	m2.SetTile("terrain", 1, 1, 1)

	world := physics.NewWorld()
	comp1 := &TileMapComponent{Map: m1, World: world}
	comp2 := &TileMapComponent{Map: m2, World: world}

	node1 := scene.NewNode("level-a")
	node2 := scene.NewNode("level-b")

	if err := comp1.Start(node1); err != nil {
		t.Fatalf("comp1.Start failed: %v", err)
	}
	if err := comp2.Start(node2); err != nil {
		t.Fatalf("comp2.Start failed: %v", err)
	}

	// Both should have colliders.
	all := world.QueryRect(geom.Rect{X: 0, Y: 0, W: 48, H: 48}, physics.LayerWorld)
	if len(all) != 2 {
		t.Fatalf("expected 2 colliders from two components, got %d", len(all))
	}

	// Destroying comp1 should leave comp2's colliders intact.
	if err := comp1.Destroy(node1); err != nil {
		t.Fatalf("comp1.Destroy failed: %v", err)
	}

	remaining := world.QueryRect(geom.Rect{X: 0, Y: 0, W: 48, H: 48}, physics.LayerWorld)
	if len(remaining) != 1 {
		t.Fatalf("expected 1 collider after destroying comp1, got %d", len(remaining))
	}
}

func TestTileMapComponentDraw(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	m := New(ts, 3, 3)
	m.AddLayer("terrain")
	m.SetTile("terrain", 0, 0, 1)
	m.SetTile("terrain", 1, 0, 2)

	comp := &TileMapComponent{
		Map: m,
		Sheet: &render.Spritesheet{
			Texture:     11,
			FrameWidth:  16,
			FrameHeight: 16,
			Columns:     2,
			Rows:        2,
		},
	}

	node := scene.NewNode("level")
	ctx := &tileDrawContextStub{}
	comp.Draw(node, ctx)

	if len(ctx.sprites) != 2 {
		t.Fatalf("expected 2 sprites, got %d", len(ctx.sprites))
	}

	if ctx.sprites[0] != (tileSpriteCall{
		texture: 11,
		src:     geom.Rect{X: 0, Y: 0, W: 16, H: 16},
		dst:     geom.Rect{X: 0, Y: 0, W: 16, H: 16},
	}) {
		t.Fatalf("unexpected first sprite: %#v", ctx.sprites[0])
	}

	if ctx.sprites[1] != (tileSpriteCall{
		texture: 11,
		src:     geom.Rect{X: 16, Y: 0, W: 16, H: 16},
		dst:     geom.Rect{X: 16, Y: 0, W: 16, H: 16},
	}) {
		t.Fatalf("unexpected second sprite: %#v", ctx.sprites[1])
	}
}
