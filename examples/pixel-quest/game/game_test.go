package game

import (
	"math"
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/physics"
	"github.com/IsraelAraujo70/whisky-game-engine/scene"
	"github.com/IsraelAraujo70/whisky-game-engine/tilemap"
)

const epsilon = 1e-9

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

// newGame builds a minimal pixelQuest with a player node and an empty world.
// Tests add colliders to g.world as needed.
func newGame(startX, startY float64) *pixelQuest {
	world := physics.NewWorld()
	player := scene.NewNode("player")
	player.Position = geom.Vec2{X: startX, Y: startY}
	return &pixelQuest{
		world:  world,
		player: player,
	}
}

// addSolid adds a solid collider to the world.
func addSolid(g *pixelQuest, x, y, w, h float64) {
	g.world.Add(physics.Collider{
		ID:     "solid",
		Bounds: geom.Rect{X: x, Y: y, W: w, H: h},
		Layer:  physics.LayerWorld,
		Mask:   physics.LayerPlayer,
	})
}

// addOneWay adds a one-way collider to the world.
func addOneWay(g *pixelQuest, x, y, w, h float64) {
	g.world.Add(physics.Collider{
		ID:     "oneway",
		Bounds: geom.Rect{X: x, Y: y, W: w, H: h},
		Layer:  physics.LayerWorld,
		Mask:   physics.LayerPlayer,
		OneWay: true,
	})
}

// --- Gravity accumulation ---

func TestGravityAccumulatesVelocity(t *testing.T) {
	g := newGame(0, 0)
	const gravityY = 400.0
	const dt = 1.0 / 60.0

	for i := 0; i < 10; i++ {
		g.velocity.Y += gravityY * dt
	}

	expected := gravityY * dt * 10
	if !approxEqual(g.velocity.Y, expected) {
		t.Fatalf("expected velocity.Y≈%.6f after 10 frames, got %.6f", expected, g.velocity.Y)
	}
}

func TestGravityIsPositive(t *testing.T) {
	// Gravity must increase Y (downward in screen space).
	g := newGame(0, 0)
	g.velocity.Y += 400.0 * (1.0 / 60.0)
	if g.velocity.Y <= 0 {
		t.Fatalf("expected positive velocity.Y (downward), got %.4f", g.velocity.Y)
	}
}

// --- resolveY: solid floor ---

func TestResolveY_LandingOnSolidZerosVelocityAndSetsGrounded(t *testing.T) {
	// Player at Y=80 (bottom=96), falling onto floor at Y=96.
	g := newGame(0, 80)
	addSolid(g, 0, 96, 64, 16)

	g.velocity.Y = 100.0 // falling
	prevBottom := g.player.Position.Y + playerH
	dy := g.velocity.Y * (1.0 / 60.0)
	g.player.Position.Y += dy

	g.resolveY(dy, prevBottom)

	if !g.grounded {
		t.Fatal("expected grounded=true after landing on solid")
	}
	if g.velocity.Y != 0 {
		t.Fatalf("expected velocity.Y=0 after landing, got %.4f", g.velocity.Y)
	}
	if g.player.Position.Y+playerH > 96+0.001 {
		t.Fatalf("player bottom (%.2f) penetrates floor at Y=96", g.player.Position.Y+playerH)
	}
}

func TestResolveY_HittingCeilingZerosVelocity(t *testing.T) {
	// Ceiling at Y=0, H=16 (bottom=16).
	// Player at Y=14 moves up by ~3.33px → Y≈10.67, overlaps ceiling → blocked.
	g := newGame(0, 14)
	addSolid(g, 0, 0, 64, 16)

	g.velocity.Y = -200.0 // jumping up
	prevBottom := g.player.Position.Y + playerH
	dy := g.velocity.Y * (1.0 / 60.0)
	g.player.Position.Y += dy

	g.resolveY(dy, prevBottom)

	if g.velocity.Y != 0 {
		t.Fatalf("expected velocity.Y=0 after hitting ceiling, got %.4f", g.velocity.Y)
	}
	if g.grounded {
		t.Fatal("expected grounded=false after ceiling hit")
	}
	// Player top should be pushed to ceiling bottom (16).
	if g.player.Position.Y != 16 {
		t.Fatalf("expected player.Y=16 after ceiling push, got %.2f", g.player.Position.Y)
	}
}

// --- resolveY: one-way platform ---

func TestResolveY_OneWayBlocksFallFromAbove(t *testing.T) {
	// Player at Y=70 (bottom=86), falling onto one-way platform at Y=88.
	g := newGame(0, 70)
	addOneWay(g, 0, 88, 64, 8)

	g.velocity.Y = 150.0
	prevBottom := g.player.Position.Y + playerH // 86 — above platform top (88)
	dy := g.velocity.Y * (1.0 / 60.0)
	g.player.Position.Y += dy

	g.resolveY(dy, prevBottom)

	if !g.grounded {
		t.Fatal("expected grounded=true when falling onto one-way from above")
	}
	if g.velocity.Y != 0 {
		t.Fatalf("expected velocity.Y=0, got %.4f", g.velocity.Y)
	}
}

func TestResolveY_OneWayDoesNotBlockJumpingUp(t *testing.T) {
	// Player below a one-way platform, jumping upward through it.
	g := newGame(0, 100)
	addOneWay(g, 0, 88, 64, 8)

	g.velocity.Y = -200.0 // moving up
	prevBottom := g.player.Position.Y + playerH
	dy := g.velocity.Y * (1.0 / 60.0)
	g.player.Position.Y += dy

	g.resolveY(dy, prevBottom)

	// Should pass through — not grounded, velocity unchanged.
	if g.grounded {
		t.Fatal("expected grounded=false: player should pass through one-way when moving up")
	}
	if g.velocity.Y != -200.0 {
		t.Fatalf("expected velocity.Y=-200, got %.4f", g.velocity.Y)
	}
}

func TestResolveY_OneWayDoesNotBlockWhenAlreadyBelow(t *testing.T) {
	// Player whose bottom was already below platform top last frame (overlapping from below).
	// prevBottom > platform.Y → should pass through even when falling.
	g := newGame(0, 85)
	addOneWay(g, 0, 88, 64, 8)

	g.velocity.Y = 50.0
	prevBottom := g.player.Position.Y + playerH // 101 — below platform top (88)
	dy := g.velocity.Y * (1.0 / 60.0)
	g.player.Position.Y += dy

	g.resolveY(dy, prevBottom)

	if g.grounded {
		t.Fatal("expected grounded=false: player was already below platform, should pass through")
	}
}

// --- Grounded resets jumpsLeft ---

func TestGroundedResetsJumpsLeft(t *testing.T) {
	g := newGame(0, 80)
	addSolid(g, 0, 96, 64, 16)
	g.jumpsLeft = 0 // used both jumps in the air

	g.velocity.Y = 100.0
	prevBottom := g.player.Position.Y + playerH
	dy := g.velocity.Y * (1.0 / 60.0)
	g.player.Position.Y += dy
	g.resolveY(dy, prevBottom)

	// Simulate the post-resolveY reset that Update does.
	if g.grounded {
		g.jumpsLeft = 2
	}

	if g.jumpsLeft != 2 {
		t.Fatalf("expected jumpsLeft=2 after landing, got %d", g.jumpsLeft)
	}
}

// --- Integration tests: tilemap → GenerateColliders → World → resolveY ---
// These tests exercise the full pipeline so that changes in geom.Rect.Intersects,
// physics.World.QueryRect, or tilemap.GenerateColliders are caught here.

func newGameFromTilemap(ts *tilemap.TileSet, m *tilemap.TileMap, startX, startY float64) *pixelQuest {
	world := physics.NewWorld()
	tilemap.AddToWorld(m, world, geom.Vec2{}, tilemap.DefaultColliderConfig())
	player := scene.NewNode("player")
	player.Position = geom.Vec2{X: startX, Y: startY}
	return &pixelQuest{world: world, player: player}
}

func TestIntegration_LandOnSolidTile(t *testing.T) {
	ts := tilemap.NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, tilemap.TileProperties{Solid: true})
	m := tilemap.New(ts, 10, 10)
	m.AddLayer("terrain")
	m.FillRow("terrain", 0, 9, 10, 1) // solid floor at row 9 → Y=144

	// Player just above the floor, falling.
	// bottom=143, floor top=144. dy=200*(1/60)≈3.33 → bottom crosses 144.
	g := newGameFromTilemap(ts, m, 8, 127) // bottom=143, floor top=144
	g.velocity.Y = 200.0
	prevBottom := g.player.Position.Y + playerH
	dy := g.velocity.Y * (1.0 / 60.0)
	g.player.Position.Y += dy
	g.resolveY(dy, prevBottom)

	if !g.grounded {
		t.Fatal("expected grounded=true after landing on solid tile floor")
	}
	if g.velocity.Y != 0 {
		t.Fatalf("expected velocity.Y=0, got %.4f", g.velocity.Y)
	}
	if g.player.Position.Y+playerH > 144+0.001 {
		t.Fatalf("player bottom (%.2f) penetrates tile floor at Y=144", g.player.Position.Y+playerH)
	}
}

func TestIntegration_OneWayTileBlocksFallFromAbove(t *testing.T) {
	ts := tilemap.NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, tilemap.TileProperties{Solid: true, OneWay: true})
	m := tilemap.New(ts, 10, 10)
	m.AddLayer("terrain")
	m.FillRow("terrain", 0, 5, 5, 1) // one-way row at Y=80

	// Player just above the platform, falling.
	// bottom=79, platform top=80. dy≈3.33 → bottom crosses 80.
	g := newGameFromTilemap(ts, m, 8, 63) // bottom=79, platform top=80
	g.velocity.Y = 200.0
	prevBottom := g.player.Position.Y + playerH
	dy := g.velocity.Y * (1.0 / 60.0)
	g.player.Position.Y += dy
	g.resolveY(dy, prevBottom)

	if !g.grounded {
		t.Fatal("expected grounded=true after landing on one-way tile from above")
	}
}

func TestIntegration_OneWayTilePassThroughFromBelow(t *testing.T) {
	ts := tilemap.NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, tilemap.TileProperties{Solid: true, OneWay: true})
	m := tilemap.New(ts, 10, 10)
	m.AddLayer("terrain")
	m.FillRow("terrain", 0, 5, 5, 1) // one-way row at Y=80

	// Player below the platform, jumping up through it.
	g := newGameFromTilemap(ts, m, 8, 90)
	g.velocity.Y = -200.0
	prevBottom := g.player.Position.Y + playerH // 106 > 80 — already below
	dy := g.velocity.Y * (1.0 / 60.0)
	g.player.Position.Y += dy
	g.resolveY(dy, prevBottom)

	if g.grounded {
		t.Fatal("expected grounded=false: player jumping through one-way from below")
	}
	if g.velocity.Y != -200.0 {
		t.Fatalf("expected velocity unchanged, got %.4f", g.velocity.Y)
	}
}

func TestIntegration_SolidWallBlocksHorizontal(t *testing.T) {
	ts := tilemap.NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, tilemap.TileProperties{Solid: true})
	m := tilemap.New(ts, 10, 10)
	m.AddLayer("terrain")
	m.FillCol("terrain", 9, 0, 10, 1) // right wall at X=144

	// Player moving right into wall.
	g := newGameFromTilemap(ts, m, 132, 0) // right edge=140, wall at X=144
	dx := 8.0
	g.player.Position.X += dx
	g.resolveX(dx)

	if g.player.Position.X+playerW > 144+0.001 {
		t.Fatalf("player right edge (%.2f) penetrates wall at X=144", g.player.Position.X+playerW)
	}
}

func TestIntegration_TriggerTileDetected(t *testing.T) {
	ts := tilemap.NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, tilemap.TileProperties{Trigger: true})
	m := tilemap.New(ts, 10, 10)
	m.AddLayer("terrain")
	m.SetTile("terrain", 5, 5, 1) // trigger at tile (5,5) → world X=80,Y=80

	g := newGameFromTilemap(ts, m, 80, 80)
	pp := g.player.WorldPosition()
	playerRect := geom.Rect{X: pp.X, Y: pp.Y, W: playerW, H: playerH}
	hits := g.world.QueryRect(playerRect, physics.LayerTrigger)

	if len(hits) != 1 {
		t.Fatalf("expected 1 trigger hit, got %d", len(hits))
	}
	if !hits[0].Trigger {
		t.Fatal("expected Trigger=true on hit collider")
	}
}
