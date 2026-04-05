package game

import (
	"fmt"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/physics"
	"github.com/IsraelAraujo70/whisky-game-engine/scene"
	"github.com/IsraelAraujo70/whisky-game-engine/tilemap"
	whisky "github.com/IsraelAraujo70/whisky-game-engine/whisky"
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

	ctx.Logf("pixel-quest booted with tilemap (%dx%d tiles)", m.Width, m.Height)
	ctx.SetDebugText(
		"Whisky SDL3 bootstrap is live.",
		"Player walks to the right automatically.",
		"Wait for the trigger, then close with Esc.",
	)
	return nil
}

func (g *pixelQuest) Update(ctx *whisky.Context, dt float64) error {
	if !g.triggerReached {
		g.player.Position = g.player.Position.Add(geom.Vec2{X: 0.5, Y: 0})
	}

	hits := g.world.QueryPoint(g.player.WorldPosition(), physics.LayerTrigger)
	if len(hits) > 0 && !g.triggerReached {
		g.triggerReached = true
		ctx.Logf("player reached trigger %s on frame %d", hits[0].ID, ctx.Frames)
	}

	status := "walking"
	if g.triggerReached {
		status = "trigger reached"
	}

	// Count solid colliders to show merge efficiency.
	solidCount := len(g.world.QueryRect(g.tileMap.WorldBounds(), physics.LayerWorld))

	ctx.SetDebugText(
		"Whisky SDL3 bootstrap is live.",
		fmt.Sprintf("player=(%.1f, %.1f)", g.player.WorldPosition().X, g.player.WorldPosition().Y),
		fmt.Sprintf("state=%s", status),
		fmt.Sprintf("map=%dx%d tiles, %d merged colliders", g.tileMap.Width, g.tileMap.Height, solidCount),
		"Close with Esc or the window close button.",
	)
	return nil
}

func (g *pixelQuest) Shutdown(ctx *whisky.Context) error {
	ctx.Logf("pixel-quest shutdown")
	return nil
}
