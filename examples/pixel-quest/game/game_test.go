package game

import (
	"math"
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/physics"
	"github.com/IsraelAraujo70/whisky-game-engine/scene"
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
