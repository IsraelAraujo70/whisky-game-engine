package game

import (
	"fmt"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/physics"
	"github.com/IsraelAraujo70/whisky-game-engine/scene"
	whisky "github.com/IsraelAraujo70/whisky-game-engine/whisky"
)

type pixelQuest struct {
	player         *scene.Node
	world          *physics.World
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
	g.player = scene.NewNode("player")
	g.player.Position = geom.Vec2{X: 8, Y: 8}
	ctx.Scene.Root.AddChild(g.player)

	g.world = physics.NewWorld()
	g.world.Add(physics.Collider{
		ID:      "door",
		Bounds:  geom.Rect{X: 16, Y: 8, W: 8, H: 8},
		Layer:   physics.LayerTrigger,
		Mask:    physics.LayerPlayer,
		Trigger: true,
	})

	ctx.Logf("pixel-quest booted")
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

	ctx.SetDebugText(
		"Whisky SDL3 bootstrap is live.",
		fmt.Sprintf("player=(%.1f, %.1f)", g.player.WorldPosition().X, g.player.WorldPosition().Y),
		fmt.Sprintf("state=%s", status),
		"Close with Esc or the window close button.",
	)
	return nil
}

func (g *pixelQuest) Shutdown(ctx *whisky.Context) error {
	ctx.Logf("pixel-quest shutdown")
	return nil
}
