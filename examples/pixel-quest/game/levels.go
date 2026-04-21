package game

import (
	"fmt"

	"github.com/IsraelAraujo70/whisky-game-engine/gameplay"
	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/physics"
	"github.com/IsraelAraujo70/whisky-game-engine/scene"
	"github.com/IsraelAraujo70/whisky-game-engine/tilemap"
	whisky "github.com/IsraelAraujo70/whisky-game-engine/whisky"
)

// levelDesc describes a playable stage.
type levelDesc struct {
	Name        string
	Width       int // tiles
	Height      int // tiles
	TileSet     *tilemap.TileSet
	Build       func(m *tilemap.TileMap)
	PlayerStart geom.Vec2
	Enemies     []enemySpawn
	TriggerPos  geom.Vec2 // tile coordinates
}

// enemySpawn describes one enemy placement.
type enemySpawn struct {
	ID       string
	X, Y     float64
	Typ      enemyType
	HP       int
	Speed    float64
	Drops    []gameplay.DropEntry
}

// allLevels defines the campaign.
var allLevels = []levelDesc{
	{
		Name:        "Tutorial",
		Width:       20,
		Height:      12,
		PlayerStart: geom.Vec2{X: 24, Y: 144},
		TriggerPos:  geom.Vec2{X: 18, Y: 10},
		Enemies: []enemySpawn{
			{ID: "slime:a", X: 88, Y: 164, Typ: enemyPatrol, HP: 2, Speed: 22, Drops: defaultSlimeDrops()},
		},
		Build: buildLevel1,
	},
	{
		Name:        "Caverns",
		Width:       24,
		Height:      14,
		PlayerStart: geom.Vec2{X: 24, Y: 192},
		TriggerPos:  geom.Vec2{X: 22, Y: 12},
		Enemies: []enemySpawn{
			{ID: "slime:b", X: 100, Y: 192, Typ: enemyPatrol, HP: 2, Speed: 24, Drops: defaultSlimeDrops()},
			{ID: "jumper:a", X: 180, Y: 160, Typ: enemyJumper, HP: 3, Speed: 18, Drops: jumperDrops()},
		},
		Build: buildLevel2,
	},
	{
		Name:        "Fortress",
		Width:       28,
		Height:      16,
		PlayerStart: geom.Vec2{X: 24, Y: 224},
		TriggerPos:  geom.Vec2{X: 26, Y: 14},
		Enemies: []enemySpawn{
			{ID: "shooter:a", X: 120, Y: 224, Typ: enemyShooter, HP: 3, Speed: 15, Drops: shooterDrops()},
			{ID: "slime:c", X: 240, Y: 224, Typ: enemyPatrol, HP: 3, Speed: 26, Drops: defaultSlimeDrops()},
			{ID: "boss:king", X: 360, Y: 200, Typ: enemyBoss, HP: 10, Speed: 12, Drops: bossDrops()},
		},
		Build: buildLevel3,
	},
}

func defaultSlimeDrops() []gameplay.DropEntry {
	return []gameplay.DropEntry{
		{Kind: "xp", MinAmount: 1, MaxAmount: 2, Chance: 1},
		{Kind: "health", MinAmount: 1, MaxAmount: 1, Chance: 0.35},
	}
}

func jumperDrops() []gameplay.DropEntry {
	return []gameplay.DropEntry{
		{Kind: "xp", MinAmount: 2, MaxAmount: 3, Chance: 1},
		{Kind: "coin", MinAmount: 1, MaxAmount: 3, Chance: 0.6},
	}
}

func shooterDrops() []gameplay.DropEntry {
	return []gameplay.DropEntry{
		{Kind: "xp", MinAmount: 2, MaxAmount: 4, Chance: 1},
		{Kind: "item", ID: "crystal", MinAmount: 1, MaxAmount: 1, Chance: 0.4},
	}
}

func bossDrops() []gameplay.DropEntry {
	return []gameplay.DropEntry{
		{Kind: "xp", MinAmount: 10, MaxAmount: 15, Chance: 1},
		{Kind: "coin", MinAmount: 5, MaxAmount: 10, Chance: 1},
		{Kind: "item", ID: "crown", MinAmount: 1, MaxAmount: 1, Chance: 1},
	}
}

func buildLevel1(m *tilemap.TileMap) {
	m.AddLayer("terrain")
	m.FillRow("terrain", 0, 11, 20, 1)
	m.BuildPlatform("terrain", 3, 8, 5, 2)
	m.BuildPlatform("terrain", 12, 6, 4, 1)
	m.FillCol("terrain", 0, 0, 11, 1)
	m.FillCol("terrain", 19, 0, 11, 1)
	m.SetTile("terrain", 18, 10, 3)
}

func buildLevel2(m *tilemap.TileMap) {
	m.AddLayer("terrain")
	m.FillRow("terrain", 0, 13, 24, 1)
	m.BuildPlatform("terrain", 4, 10, 4, 2)
	m.BuildPlatform("terrain", 10, 8, 3, 1)
	m.BuildPlatform("terrain", 16, 6, 4, 2)
	m.FillCol("terrain", 0, 0, 13, 1)
	m.FillCol("terrain", 23, 0, 13, 1)
	m.SetTile("terrain", 22, 12, 3)
}

func buildLevel3(m *tilemap.TileMap) {
	m.AddLayer("terrain")
	m.FillRow("terrain", 0, 15, 28, 1)
	m.BuildPlatform("terrain", 5, 12, 4, 2)
	m.BuildPlatform("terrain", 12, 10, 3, 1)
	m.BuildPlatform("terrain", 18, 8, 4, 2)
	m.BuildPlatform("terrain", 24, 11, 3, 1)
	m.FillCol("terrain", 0, 0, 15, 1)
	m.FillCol("terrain", 27, 0, 15, 1)
	m.SetTile("terrain", 26, 14, 3)
}

// loadLevel rebuilds the world and scene for the given level index.
func (g *pixelQuest) loadLevel(ctx *whisky.Context, levelIdx int) {
	if levelIdx < 0 || levelIdx >= len(allLevels) {
		return
	}
	g.currentLevel = levelIdx
	lvl := allLevels[levelIdx]

	g.world = physics.NewWorld()
	g.enemies = nil
	g.enemyStates = make(map[string]*enemyState)
	g.playerDefeated = false
	g.triggerReached = false
	g.velocity = geom.Vec2{}
	g.grounded = false
	g.jumpsLeft = 2
	g.attackTimer = 0
	g.projectiles = nil

	ts := tilemap.NewTileSet("quest", 16, 16, 4)
	ts.SetProperties(1, tilemap.TileProperties{Solid: true})
	ts.SetProperties(2, tilemap.TileProperties{Solid: true, OneWay: true})
	ts.SetProperties(3, tilemap.TileProperties{Trigger: true, Tags: map[string]string{"type": "door"}})

	m := tilemap.New(ts, lvl.Width, lvl.Height)
	lvl.Build(m)
	g.tileMap = m

	_ = g.loadSprites(ctx)

	levelNode := scene.NewNode("level")
	levelNode.AddComponent(&tilemap.TileMapComponent{Map: m, World: g.world, Sheet: g.tileSheet})
	ctx.Scene.Root.AddChild(levelNode)

	g.player = scene.NewNode("player")
	g.player.Position = lvl.PlayerStart
	g.playerFacing = 1
	g.playerSprite = &scene.SpriteComponent{Sheet: g.playerSheet, W: playerW, H: playerH}
	g.player.AddComponent(g.playerSprite)
	g.playerHealth = gameplay.NewHealth(playerMaxHP)
	g.playerHealth.InvulnerableFor = playerInvulnerableFor
	g.player.AddComponent(g.playerHealth)
	ctx.Scene.Root.AddChild(g.player)

	for _, es := range lvl.Enemies {
		g.spawnEnemyByType(ctx.Scene.Root, es.ID, es.X, es.Y, es.Typ, es.HP, es.Speed, es.Drops)
	}
}

// screenLevelSelect is the level picker menu.
type screenLevelSelect struct {
	menu *uiMenu
}

func newScreenLevelSelect(g *pixelQuest) *screenLevelSelect {
	s := &screenLevelSelect{menu: newUIMenu()}
	for i, lvl := range allLevels {
		idx := i
		locked := i > 0 && !g.saveData.UnlockedLevels[i]
		label := fmt.Sprintf("%d. %s", i+1, lvl.Name)
		if locked {
			label += " [LOCKED]"
		}
		btn := s.menu.AddButton(label, func() {
			if !locked {
				g.loadLevel(nil, idx)
				g.changeState(statePlaying)
			}
		})
		btn.Enabled = !locked
	}
	s.menu.AddButton("Back", func() { g.popState() })
	return s
}

func (s *screenLevelSelect) Update(g *pixelQuest, ctx *whisky.Context, dt float64) {
	s.menu.Update(ctx, dt)
}

func (s *screenLevelSelect) Draw(g *pixelQuest, ctx *whisky.Context) {
	vw, vh := ctx.VirtualSize()
	ctx.DrawRect(geom.Rect{X: 0, Y: 0, W: vw, H: vh}, geom.RGBA(0.02, 0.02, 0.04, 0.9))
	panelW, panelH := 160.0, float64(len(allLevels)+2)*22+20
	px := (vw - panelW) / 2
	py := (vh - panelH) / 2
	uiPanel(ctx, px, py, panelW, panelH)
	uiTitle(ctx, px, py, panelW, 20, "LEVEL SELECT")

	s.menu.LayoutCentered(py+26, 130, 16, 5, vw, vh)
	s.menu.Draw(ctx)
}
