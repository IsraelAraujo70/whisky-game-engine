package game

import (
	"fmt"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/IsraelAraujo70/whisky-game-engine/gameplay"
	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/input"
	"github.com/IsraelAraujo70/whisky-game-engine/physics"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
	"github.com/IsraelAraujo70/whisky-game-engine/scene"
	"github.com/IsraelAraujo70/whisky-game-engine/tilemap"
	whisky "github.com/IsraelAraujo70/whisky-game-engine/whisky"
)

const (
	playerW               = 8.0
	playerH               = 16.0
	moveSpeed             = 80.0   // pixels per second
	sprintSpeed           = 160.0  // pixels per second while sprinting
	jumpVel               = -200.0 // pixels per second upward (negative Y = up)
	playerMaxHP           = 5
	playerInvulnerableFor = 0.6
	playerAttackDamage    = 1
	playerAttackReach     = 12.0
	playerAttackInsetY    = 2.0
	playerAttackDuration  = 0.12
	enemyW                = 12.0
	enemyH                = 12.0
	enemyTouchDamage      = 1
	enemySightRangeX      = 56.0
	enemySightRangeY      = 24.0
)

type enemy struct {
	id     string
	node   *scene.Node
	patrol *gameplay.PatrolComponent
	target *gameplay.TargetComponent
	health *gameplay.Health
	drops  *gameplay.DropComponent
}

type pixelQuest struct {
	player         *scene.Node
	playerSprite   *scene.SpriteComponent
	playerHealth   *gameplay.Health
	playerFacing   float64
	attackTimer    float64
	enemies        []*enemy
	playerDefeated bool
	world          *physics.World
	rng            *rand.Rand
	tileMap        *tilemap.TileMap
	tileSheet      *render.Spritesheet
	playerSheet    *render.Spritesheet
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
			"space":  "jump", "up": "jump", "w": "jump",
			"j": "attack", "k": "attack",
		},
	})
}

func (g *pixelQuest) Load(ctx *whisky.Context) error {
	g.world = physics.NewWorld()
	g.rng = rand.New(rand.NewSource(7))

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
	if err := g.loadSprites(ctx); err != nil {
		return err
	}

	// Attach tilemap to scene via component.
	levelNode := scene.NewNode("level")
	levelNode.AddComponent(&tilemap.TileMapComponent{
		Map:   m,
		World: g.world,
		Sheet: g.tileSheet,
	})
	ctx.Scene.Root.AddChild(levelNode)

	// --- Player setup ---
	g.player = scene.NewNode("player")
	g.player.Position = geom.Vec2{X: 24, Y: 144}
	g.playerFacing = 1
	g.playerSprite = &scene.SpriteComponent{
		Sheet: g.playerSheet,
		W:     playerW,
		H:     playerH,
	}
	g.player.AddComponent(g.playerSprite)
	g.playerHealth = gameplay.NewHealth(playerMaxHP)
	g.playerHealth.InvulnerableFor = playerInvulnerableFor
	g.player.AddComponent(g.playerHealth)
	ctx.Scene.Root.AddChild(g.player)

	g.spawnEnemy(ctx.Scene.Root, "enemy:slime:1", 88, 164, 80, 136, 2, 22, []gameplay.DropEntry{
		{Kind: "xp", MinAmount: 1, MaxAmount: 2, Chance: 1},
		{Kind: "health", MinAmount: 1, MaxAmount: 1, Chance: 0.35},
	})
	g.spawnEnemy(ctx.Scene.Root, "enemy:slime:2", 212, 164, 196, 260, 3, 28, []gameplay.DropEntry{
		{Kind: "xp", MinAmount: 2, MaxAmount: 4, Chance: 1},
		{Kind: "item", ID: "slime_gel", MinAmount: 1, MaxAmount: 1, Chance: 0.5},
	})

	// --- Input bindings ---
	ctx.Input.Bind("move_left", "move_left")
	ctx.Input.Bind("move_right", "move_right")
	ctx.Input.Bind("sprint", "sprint")
	ctx.Input.Bind("jump", "jump")
	ctx.Input.Bind("attack", "attack")

	ctx.Logf("pixel-quest booted with tilemap (%dx%d tiles)", m.Width, m.Height)
	return nil
}

func (g *pixelQuest) Update(ctx *whisky.Context, dt float64) error {
	playerAlive := g.playerHealth == nil || g.playerHealth.Alive()
	if g.attackTimer > 0 {
		g.attackTimer -= dt
		if g.attackTimer < 0 {
			g.attackTimer = 0
		}
	}

	// Reset grounded each frame — resolveY sets it back to true on landing.
	g.grounded = false

	// --- Horizontal movement ---
	speed := 0.0
	if playerAlive {
		speed = moveSpeed
		if ctx.Input.Pressed("sprint") {
			speed = sprintSpeed
		}
	}
	dx := ctx.Input.Axis("move_left", "move_right") * speed * dt
	if dx > 0 {
		g.playerFacing = 1
	} else if dx < 0 {
		g.playerFacing = -1
	}
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
	if playerAlive && g.jumpsLeft > 0 && ctx.Input.JustPressed("jump") {
		g.velocity.Y = jumpVel
		g.jumpsLeft--
	}

	if playerAlive && ctx.Input.JustPressed("attack") {
		g.attackTimer = playerAttackDuration
	}

	for _, event := range gameplay.ResolveDamage(g.damageSources(), g.damageTargets()) {
		switch event.TargetID {
		case "player":
			ctx.Logf("player took %d damage (%d/%d hp)", event.Amount, g.playerHealth.Current, g.playerHealth.Max)
			if !g.playerHealth.Alive() && !g.playerDefeated {
				g.playerDefeated = true
				ctx.Logf("player defeated on frame %d", ctx.Frames)
			}
		default:
			enemy := g.enemyByID(event.TargetID)
			if enemy == nil {
				continue
			}
			ctx.Logf("%s took %d damage (%d/%d hp)", enemy.id, event.Amount, enemy.health.Current, enemy.health.Max)
			if !enemy.health.Alive() {
				enemy.patrol.Disabled = true
				if enemy.target != nil {
					enemy.target.Disabled = true
				}
				if drops := enemy.drops.Roll(g.rng); len(drops) > 0 {
					ctx.Logf("%s dropped %s", enemy.id, formatDrops(drops))
				}
				ctx.Logf("%s defeated", enemy.id)
			}
		}
	}

	// --- Trigger detection ---
	pp := g.player.WorldPosition()
	playerRect := geom.Rect{X: pp.X, Y: pp.Y, W: playerW, H: playerH}
	triggers := g.world.QueryRect(playerRect, physics.LayerTrigger)
	if len(triggers) > 0 && !g.triggerReached {
		g.triggerReached = true
		ctx.Logf("player reached trigger %s on frame %d", triggers[0].ID, ctx.Frames)
	}

	// Fallback rendering path when sprite loading is unavailable.
	if g.tileSheet == nil {
		visible := tilemap.VisibleTiles(g.tileMap, ctx.ViewportRect(), geom.Vec2{})
		tw, th := g.tileMap.TileSize()
		for _, t := range visible {
			ctx.DrawRect(geom.Rect{
				X: t.WorldPos.X,
				Y: t.WorldPos.Y,
				W: float64(tw),
				H: float64(th),
			}, tileColor(t.ID))
		}
	}
	if g.playerSheet == nil {
		color := geom.RGBA(0.2, 0.8, 0.3, 1)
		if !playerAlive {
			color = geom.RGBA(0.5, 0.1, 0.1, 1)
		} else if g.playerHealth != nil && g.playerHealth.Invulnerable() {
			color = geom.RGBA(0.9, 0.9, 0.3, 1)
		}
		ctx.DrawRect(geom.Rect{X: pp.X, Y: pp.Y, W: playerW, H: playerH}, color)
	}
	if g.playerSprite != nil {
		g.playerSprite.FlipH = g.playerFacing < 0
	}
	for _, enemy := range g.enemies {
		if enemy.health == nil || !enemy.health.Alive() {
			continue
		}
		pos := enemy.node.WorldPosition()
		color := geom.RGBA(0.9, 0.3, 0.25, 1)
		if enemy.target != nil && enemy.target.Chasing {
			color = geom.RGBA(1, 0.1, 0.1, 1)
		}
		if enemy.health.Invulnerable() {
			color = geom.RGBA(1, 0.7, 0.55, 1)
		}
		ctx.DrawRect(geom.Rect{X: pos.X, Y: pos.Y, W: enemyW, H: enemyH}, color)
		g.drawHealthBar(ctx, pos.X, pos.Y-4, enemyW, enemy.health)
	}
	if g.playerHealth != nil {
		g.drawHealthBar(ctx, pp.X, pp.Y-5, playerW, g.playerHealth)
	}
	if g.attackTimer > 0 && playerAlive {
		ctx.DrawRect(g.playerAttackBox().Rect(), geom.RGBA(1.0, 0.85, 0.2, 0.55))
	}

	// --- Mouse crosshair ---
	// Draw a small crosshair at the mouse position as a visual indicator.
	mx, my := ctx.Mouse().Position()
	crosshairColor := geom.RGBA(1, 1, 0, 0.7)
	ctx.DrawRect(geom.Rect{X: mx - 2, Y: my, W: 5, H: 1}, crosshairColor)
	ctx.DrawRect(geom.Rect{X: mx, Y: my - 2, W: 1, H: 5}, crosshairColor)

	// --- Gamepad movement (left stick) as alternative to keyboard ---
	if playerAlive {
		pad := ctx.Gamepad(0)
		if pad.Connected() {
			lx := pad.Axis(input.GamepadAxisLX)
			if lx > 0.2 || lx < -0.2 { // deadzone
				g.player.Position.X += lx * speed * dt
				if lx > 0 {
					g.playerFacing = 1
				} else {
					g.playerFacing = -1
				}
			}
			// Gamepad A button = jump
			if g.jumpsLeft > 0 && pad.JustPressed(input.GamepadButtonA) {
				g.velocity.Y = jumpVel
				g.jumpsLeft--
			}
			// Gamepad X button = attack
			if pad.JustPressed(input.GamepadButtonX) {
				g.attackTimer = playerAttackDuration
			}
		}
	}

	// --- Debug overlay ---
	status := "airborne"
	if !playerAlive {
		status = "defeated"
	} else if g.grounded {
		status = "grounded"
	}
	if playerAlive && ctx.Input.Pressed("sprint") && g.grounded {
		status = "sprinting"
	}
	if g.triggerReached {
		status = "trigger reached!"
	}

	mouseInfo := fmt.Sprintf("mouse=(%.0f,%.0f)", mx, my)
	padInfo := "gamepad=none"
	if ctx.Gamepad(0).Connected() {
		padInfo = fmt.Sprintf("gamepad=connected lx=%.2f", ctx.Gamepad(0).Axis(input.GamepadAxisLX))
	}

	ctx.SetDebugText(
		"A/D move   Space/W/Up jump   J/K attack   LShift sprint",
		fmt.Sprintf("player=(%.0f,%.0f) vel=(%.0f,%.0f)", pp.X, pp.Y, g.velocity.X, g.velocity.Y),
		fmt.Sprintf("hp=%d/%d  enemies=%d  chasing=%d  grounded=%v  state=%s", g.playerHealth.Current, g.playerHealth.Max, g.aliveEnemies(), g.chasingEnemies(), g.grounded, status),
		mouseInfo+"  "+padInfo,
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

func (g *pixelQuest) playerAttackBox() gameplay.Box {
	offsetX := playerW
	if g.playerFacing < 0 {
		offsetX = -playerAttackReach
	}
	return gameplay.Box{
		Node:   g.player,
		Offset: geom.Vec2{X: offsetX, Y: playerAttackInsetY},
		W:      playerAttackReach,
		H:      playerH - (playerAttackInsetY * 2),
	}
}

func (g *pixelQuest) damageSources() []gameplay.DamageSource {
	sources := make([]gameplay.DamageSource, 0, len(g.enemies)+1)
	if g.playerHealth != nil && g.playerHealth.Alive() && g.attackTimer > 0 {
		sources = append(sources, gameplay.DamageSource{
			ID:     "player:attack",
			Team:   gameplay.TeamPlayer,
			Damage: playerAttackDamage,
			Box:    g.playerAttackBox(),
		})
	}

	for _, enemy := range g.enemies {
		if enemy.health == nil || !enemy.health.Alive() {
			continue
		}
		sources = append(sources, gameplay.DamageSource{
			ID:     enemy.id + ":touch",
			Team:   gameplay.TeamEnemy,
			Damage: enemyTouchDamage,
			Box: gameplay.Box{
				Node: enemy.node,
				W:    enemyW,
				H:    enemyH,
			},
		})
	}

	return sources
}

func (g *pixelQuest) damageTargets() []gameplay.DamageTarget {
	targets := make([]gameplay.DamageTarget, 0, len(g.enemies)+1)
	if g.playerHealth != nil {
		targets = append(targets, gameplay.DamageTarget{
			ID:       "player",
			Team:     gameplay.TeamPlayer,
			Health:   g.playerHealth,
			Disabled: !g.playerHealth.Alive(),
			Box: gameplay.Box{
				Node: g.player,
				W:    playerW,
				H:    playerH,
			},
		})
	}

	for _, enemy := range g.enemies {
		if enemy.health == nil {
			continue
		}
		targets = append(targets, gameplay.DamageTarget{
			ID:       enemy.id,
			Team:     gameplay.TeamEnemy,
			Health:   enemy.health,
			Disabled: !enemy.health.Alive(),
			Box: gameplay.Box{
				Node: enemy.node,
				W:    enemyW,
				H:    enemyH,
			},
		})
	}

	return targets
}

func (g *pixelQuest) drawHealthBar(ctx *whisky.Context, x, y, width float64, health *gameplay.Health) {
	if health == nil {
		return
	}

	ctx.DrawRect(geom.Rect{X: x, Y: y, W: width, H: 2}, geom.RGBA(0.15, 0.15, 0.18, 1))
	fill := width * health.Fraction()
	if fill > 0 {
		ctx.DrawRect(geom.Rect{X: x, Y: y, W: fill, H: 2}, geom.RGBA(0.2, 0.85, 0.3, 1))
	}
}

func (g *pixelQuest) spawnEnemy(root *scene.Node, id string, x, y, minX, maxX float64, hp int, speed float64, dropEntries []gameplay.DropEntry) {
	node := scene.NewNode(id)
	node.Position = geom.Vec2{X: x, Y: y}

	patrol := &gameplay.PatrolComponent{
		MinX:      minX,
		MaxX:      maxX,
		Speed:     speed,
		Direction: 1,
	}
	health := gameplay.NewHealth(hp)
	health.InvulnerableFor = 0.15
	drops := &gameplay.DropComponent{Entries: dropEntries}
	target := &gameplay.TargetComponent{
		TargetBox: gameplay.Box{
			Node: g.player,
			W:    playerW,
			H:    playerH,
		},
		Sight: gameplay.Box{
			Node:   node,
			Offset: geom.Vec2{X: -enemySightRangeX, Y: -6},
			W:      enemySightRangeX * 2,
			H:      enemySightRangeY,
		},
		Speed:          speed + 10,
		StopDistance:   6,
		HorizontalOnly: true,
		Patrol:         patrol,
	}

	node.AddComponent(patrol)
	node.AddComponent(target)
	node.AddComponent(health)
	node.AddComponent(drops)
	root.AddChild(node)

	g.enemies = append(g.enemies, &enemy{
		id:     id,
		node:   node,
		patrol: patrol,
		target: target,
		health: health,
		drops:  drops,
	})
}

func (g *pixelQuest) enemyByID(id string) *enemy {
	for _, enemy := range g.enemies {
		if enemy.id == id {
			return enemy
		}
	}
	return nil
}

func (g *pixelQuest) aliveEnemies() int {
	count := 0
	for _, enemy := range g.enemies {
		if enemy.health != nil && enemy.health.Alive() {
			count++
		}
	}
	return count
}

func (g *pixelQuest) chasingEnemies() int {
	count := 0
	for _, enemy := range g.enemies {
		if enemy.target != nil && enemy.target.Chasing && enemy.health != nil && enemy.health.Alive() {
			count++
		}
	}
	return count
}

func formatDrops(drops []gameplay.DropResult) string {
	parts := make([]string, 0, len(drops))
	for _, drop := range drops {
		parts = append(parts, drop.String())
	}
	return strings.Join(parts, ", ")
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

func (g *pixelQuest) loadSprites(ctx *whisky.Context) error {
	assetsDir, err := pixelQuestAssetsDir()
	if err != nil {
		return err
	}

	tilesPath := filepath.Join(assetsDir, "tiles.png")
	playerPath := filepath.Join(assetsDir, "player.png")

	tileTexture, _, _, err := ctx.LoadTexture(tilesPath)
	if err != nil {
		return err
	}
	playerTexture, _, _, err := ctx.LoadTexture(playerPath)
	if err != nil {
		return err
	}
	playerWpx, playerHpx, err := pngSize(playerPath)
	if err != nil {
		return err
	}

	g.tileSheet = &render.Spritesheet{
		Texture:     tileTexture,
		FrameWidth:  16,
		FrameHeight: 16,
		Columns:     4,
		Rows:        1,
	}
	g.playerSheet = &render.Spritesheet{
		Texture:     playerTexture,
		FrameWidth:  playerWpx,
		FrameHeight: playerHpx,
		Columns:     1,
		Rows:        1,
	}

	return nil
}

func pixelQuestAssetsDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", os.ErrNotExist
	}
	return filepath.Join(filepath.Dir(filename), "..", "assets"), nil
}

func pngSize(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	cfg, err := png.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}
