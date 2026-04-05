package game

import (
	"fmt"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/physics"
	"github.com/IsraelAraujo70/whisky-game-engine/scene"
	"github.com/IsraelAraujo70/whisky-game-engine/tilemap"
	whisky "github.com/IsraelAraujo70/whisky-game-engine/whisky"
)

const (
	playerW     = 8.0
	playerH     = 16.0
	moveSpeed   = 80.0   // pixels per second
	sprintSpeed = 160.0  // pixels per second while sprinting
	jumpVel     = -200.0 // pixels per second upward (negative Y = up)
)

type pixelQuest struct {
	player         *scene.Node
	world          *physics.World
	tileMap        *tilemap.TileMap
	triggerReached bool
	velocity       geom.Vec2 // accumulates gravity and jump impulse
	grounded       bool      // true when standing on solid or one-way surface
	jumpsLeft      int       // remaining jumps (reset to 2 on landing)
}

func Run() error {
	return whisky.Run(&pixelQuest{}, whisky.Config{
		Title:         "Pixel Quest",
		VirtualWidth:  320,
		VirtualHeight: 180,
		PixelPerfect:  true,
		TargetFPS:     60,
		GravityY:      400.0, // px/s² downward
		StartScene:    scene.New("pixel-quest"),
		// KeyMap maps physical keys to semantic control names.
		// Space and Up arrow both trigger jump.
		KeyMap: whisky.KeyMap{
			"a": "move_left", "left": "move_left",
			"d": "move_right", "right": "move_right",
			"lshift": "sprint",
			"space": "jump", "up": "jump", "w": "jump",
		},
	})
}

func (g *pixelQuest) Load(ctx *whisky.Context) error {
	g.world = physics.NewWorld()

	// --- Tilemap setup ---
	ts := tilemap.NewTileSet("quest", 16, 16, 4)
	ts.SetProperties(1, tilemap.TileProperties{Solid: true})
	ts.SetProperties(2, tilemap.TileProperties{Solid: true, OneWay: true})
	ts.SetProperties(3, tilemap.TileProperties{
		Trigger: true,
		Tags:    map[string]string{"type": "door"},
	})

	// 20x12 tiles = 320x192 pixels (covers 320x180 virtual resolution).
	m := tilemap.New(ts, 20, 12)
	m.AddLayer("terrain")

	// Ground floor.
	m.FillRow("terrain", 0, 11, 20, 1)
	// Platforms.
	m.BuildPlatform("terrain", 3, 8, 5, 2)  // one-way platform
	m.BuildPlatform("terrain", 12, 6, 4, 1) // solid platform
	// Walls.
	m.FillCol("terrain", 0, 0, 11, 1)  // left wall
	m.FillCol("terrain", 19, 0, 11, 1) // right wall
	// Door trigger.
	m.SetTile("terrain", 18, 10, 3)

	g.tileMap = m

	// Attach tilemap to scene via component.
	levelNode := scene.NewNode("level")
	levelNode.AddComponent(&tilemap.TileMapComponent{
		Map:   m,
		World: g.world,
	})
	ctx.Scene.Root.AddChild(levelNode)

	// --- Player setup ---
	g.player = scene.NewNode("player")
	g.player.Position = geom.Vec2{X: 24, Y: 144}
	ctx.Scene.Root.AddChild(g.player)

	// --- Input bindings ---
	ctx.Input.Bind("move_left", "move_left")
	ctx.Input.Bind("move_right", "move_right")
	ctx.Input.Bind("sprint", "sprint")
	ctx.Input.Bind("jump", "jump")

	ctx.Logf("pixel-quest booted with tilemap (%dx%d tiles)", m.Width, m.Height)
	return nil
}

func (g *pixelQuest) Update(ctx *whisky.Context, dt float64) error {
	// Reset grounded each frame — resolveY sets it back to true on landing.
	g.grounded = false

	// --- Horizontal movement ---
	speed := moveSpeed
	if ctx.Input.Pressed("sprint") {
		speed = sprintSpeed
	}
	dx := ctx.Input.Axis("move_left", "move_right") * speed * dt
	g.player.Position.X += dx
	g.resolveX(dx)

	// --- Apply gravity ---
	g.velocity.Y += ctx.Config.GravityY * dt

	// Store bottom edge before moving Y (needed for one-way platform detection).
	prevBottom := g.player.Position.Y + playerH

	// --- Move Y and resolve ---
	// resolveY sets g.grounded=true if landing on something this frame.
	dy := g.velocity.Y * dt
	g.player.Position.Y += dy
	g.resolveY(dy, prevBottom)

	// Reset jumpsLeft when landing.
	if g.grounded {
		g.jumpsLeft = 2
	}

	// --- Jump (checked after resolveY so g.grounded is accurate) ---
	if g.jumpsLeft > 0 && ctx.Input.JustPressed("jump") {
		g.velocity.Y = jumpVel
		g.jumpsLeft--
	}

	// --- Trigger detection ---
	pp := g.player.WorldPosition()
	playerRect := geom.Rect{X: pp.X, Y: pp.Y, W: playerW, H: playerH}
	triggers := g.world.QueryRect(playerRect, physics.LayerTrigger)
	if len(triggers) > 0 && !g.triggerReached {
		g.triggerReached = true
		ctx.Logf("player reached trigger %s on frame %d", triggers[0].ID, ctx.Frames)
	}

	// --- Rendering ---
	vw := float64(ctx.Config.VirtualWidth)
	vh := float64(ctx.Config.VirtualHeight)
	cameraRect := ctx.Camera.ViewportRect(vw, vh)

	// Draw tiles as colored rectangles.
	visible := tilemap.VisibleTiles(g.tileMap, cameraRect, geom.Vec2{})
	tw, th := g.tileMap.TileSize()
	for _, t := range visible {
		ctx.DrawRect(geom.Rect{
			X: t.WorldPos.X,
			Y: t.WorldPos.Y,
			W: float64(tw),
			H: float64(th),
		}, tileColor(t.ID))
	}

	// Draw player on top of tiles.
	ctx.DrawRect(geom.Rect{X: pp.X, Y: pp.Y, W: playerW, H: playerH}, geom.RGBA(0.2, 0.8, 0.3, 1))

	// --- Debug overlay ---
	status := "airborne"
	if g.grounded {
		status = "grounded"
	}
	if ctx.Input.Pressed("sprint") && g.grounded {
		status = "sprinting"
	}
	if g.triggerReached {
		status = "trigger reached!"
	}

	ctx.SetDebugText(
		"A/D to move   Space/W/Up to jump   LShift to sprint",
		fmt.Sprintf("player=(%.0f,%.0f) vel=(%.0f,%.0f)", pp.X, pp.Y, g.velocity.X, g.velocity.Y),
		fmt.Sprintf("grounded=%v  state=%s", g.grounded, status),
	)
	return nil
}

// resolveX pushes the player out of solid colliders on the X axis.
// One-way platforms are skipped — they never block horizontal movement.
func (g *pixelQuest) resolveX(dx float64) {
	if dx == 0 {
		return
	}
	bounds := g.playerBounds()
	for _, h := range g.world.QueryRect(bounds, physics.LayerWorld) {
		if h.OneWay {
			continue
		}
		if dx > 0 {
			g.player.Position.X = h.Bounds.X - playerW
		} else {
			g.player.Position.X = h.Bounds.X + h.Bounds.W
		}
	}
}

// resolveY pushes the player out of colliders on the Y axis.
// prevBottom is the player's bottom edge before the Y move — used to detect
// whether the player was above a one-way platform before landing on it.
func (g *pixelQuest) resolveY(dy float64, prevBottom float64) {
	if dy == 0 {
		return
	}
	bounds := g.playerBounds()
	for _, h := range g.world.QueryRect(bounds, physics.LayerWorld) {
		if h.OneWay {
			// One-way: only block when falling AND player bottom was above
			// the platform top last frame.
			if dy > 0 && prevBottom <= h.Bounds.Y {
				g.player.Position.Y = h.Bounds.Y - playerH
				g.velocity.Y = 0
				g.grounded = true
			}
			// Moving up or already overlapping from below: pass through.
		} else {
			// Solid: block from both directions.
			if dy > 0 {
				g.player.Position.Y = h.Bounds.Y - playerH
				g.velocity.Y = 0
				g.grounded = true
			} else {
				g.player.Position.Y = h.Bounds.Y + h.Bounds.H
				g.velocity.Y = 0
			}
		}
	}
}

func (g *pixelQuest) playerBounds() geom.Rect {
	p := g.player.WorldPosition()
	return geom.Rect{X: p.X, Y: p.Y, W: playerW, H: playerH}
}

func tileColor(id tilemap.TileID) geom.Color {
	switch id {
	case 1:
		return geom.RGBA(0.4, 0.4, 0.45, 1) // solid — gray
	case 2:
		return geom.RGBA(0.3, 0.5, 0.7, 1) // one-way — blue
	case 3:
		return geom.RGBA(0.9, 0.7, 0.2, 1) // trigger/door — gold
	default:
		return geom.RGBA(1, 0, 1, 1) // fallback — magenta
	}
}

func (g *pixelQuest) Shutdown(ctx *whisky.Context) error {
	ctx.Logf("pixel-quest shutdown")
	return nil
}
