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
	playerW   = 8.0
	playerH   = 16.0
	moveSpeed = 80.0 // pixels per second
)

type pixelQuest struct {
	player         *scene.Node
	world          *physics.World
	tileMap        *tilemap.TileMap
	triggerReached bool
}

func Run() error {
	return whisky.Run(&pixelQuest{}, whisky.Config{
		Title:         "Pixel Quest",
		VirtualWidth:  320,
		VirtualHeight: 180,
		PixelPerfect:  true,
		TargetFPS:     60,
		StartScene:    scene.New("pixel-quest"),
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
	g.player.Position = geom.Vec2{X: 24, Y: 160}
	ctx.Scene.Root.AddChild(g.player)

	// --- Input bindings ---
	ctx.Input.Bind("move_left", "a", "left")
	ctx.Input.Bind("move_right", "d", "right")
	ctx.Input.Bind("move_up", "w", "up")
	ctx.Input.Bind("move_down", "s", "down")

	ctx.Logf("pixel-quest booted with tilemap (%dx%d tiles)", m.Width, m.Height)
	return nil
}

func (g *pixelQuest) Update(ctx *whisky.Context, dt float64) error {
	// --- Movement + collision ---
	dx := ctx.Input.Axis("move_left", "move_right") * moveSpeed * dt
	dy := ctx.Input.Axis("move_up", "move_down") * moveSpeed * dt

	// Move X axis, then resolve collisions.
	g.player.Position.X += dx
	g.resolveX(dx)

	// Move Y axis, then resolve collisions.
	g.player.Position.Y += dy
	g.resolveY(dy)

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
	status := "walking"
	if g.triggerReached {
		status = "trigger reached!"
	}

	ctx.SetDebugText(
		"WASD to move",
		fmt.Sprintf("player=(%.0f, %.0f)", pp.X, pp.Y),
		fmt.Sprintf("state=%s", status),
	)
	return nil
}

// resolveX pushes the player out of solid colliders on the X axis.
func (g *pixelQuest) resolveX(dx float64) {
	if dx == 0 {
		return
	}
	bounds := g.playerBounds()
	for _, h := range g.world.QueryRect(bounds, physics.LayerWorld) {
		if dx > 0 {
			g.player.Position.X = h.Bounds.X - playerW
		} else {
			g.player.Position.X = h.Bounds.X + h.Bounds.W
		}
	}
}

// resolveY pushes the player out of solid colliders on the Y axis.
func (g *pixelQuest) resolveY(dy float64) {
	if dy == 0 {
		return
	}
	bounds := g.playerBounds()
	for _, h := range g.world.QueryRect(bounds, physics.LayerWorld) {
		if dy > 0 {
			g.player.Position.Y = h.Bounds.Y - playerH
		} else {
			g.player.Position.Y = h.Bounds.Y + h.Bounds.H
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
