package game

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/IsraelAraujo70/whisky-game-engine/audio"
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

	// Audio sound effects. Nil when audio engine is not available.
	jumpSound   *audio.Sound
	attackSound *audio.Sound

	// Game state machine.
	state         gameState
	stateStack    []gameState
	config        GameConfig
	quitRequested bool

	// Screens (lazily created in Load).
	titleScreen      *screenTitle
	pauseScreen      *screenPause
	optionsScreen    *screenOptions
	controlsScreen   *screenControls
	gameOverScreen   *screenGameOver
	victoryScreen    *screenVictory
	levelSelectScreen *screenLevelSelect
	saveSlotsScreen  *screenSaveSlots

	// Subsystems.
	particles   *particleSystem
	dropManager *dropManager
	score       *scoreState

	// Extended enemy state.
	enemyStates map[string]*enemyState
	projectiles []*projectile

	// Progression.
	currentLevel int
	saveData     *saveData

	// ctx holds the current frame's whisky.Context so button closures can
	// safely access it without storing stale references on screen structs.
	ctx *whisky.Context
}

func Run() error {
	// Build an identity KeyMap so the platform layer emits raw key names
	// (e.g. "a", "left") as control names. The game-level binding system
	// then maps actions to those same raw names via applyKeyMap.
	km := make(whisky.KeyMap, len(controlNames))
	for _, name := range controlNames {
		km[name] = name
	}

	return whisky.Run(&pixelQuest{}, whisky.Config{
		Title:         "Pixel Quest",
		VirtualWidth:  320,
		VirtualHeight: 180,
		PixelPerfect:  true,
		TargetFPS:     60,
		GravityY:      400.0, // px/s² downward
		StartScene:    scene.New("pixel-quest"),
		AssetsRoot:    "assets",
		HotReload:     true,
		KeyMap:        km,
	})
}

func (g *pixelQuest) Load(ctx *whisky.Context) error {
	// --- Config ---
	cfg, err := loadGameConfig()
	if err != nil {
		cfg = defaultGameConfig()
	}
	g.config = cfg
	applyKeyMap(ctx, g.config.KeyMap)
	applyDisplayConfig(ctx, g.config)

	// --- Save data ---
	sd, err := loadSaveData()
	if err != nil {
		sd = newSaveData()
	}
	g.saveData = sd

	// --- Subsystems ---
	g.particles = newParticleSystem()
	g.dropManager = newDropManager()
	g.score = newScoreState()
	g.enemyStates = make(map[string]*enemyState)

	// --- State machine screens ---
	g.titleScreen = newScreenTitle(g)
	g.pauseScreen = newScreenPause(g)
	g.optionsScreen = newScreenOptions(g)
	g.controlsScreen = newScreenControls(g)
	g.gameOverScreen = newScreenGameOver(g)
	g.victoryScreen = newScreenVictory(g)
	g.changeState(stateTitle)

	g.initLevel(ctx)

	// --- Audio ---
	g.jumpSound = audio.NewSoundFromSamples(
		audio.GenerateSineWave(523.25, 0.08, 48000), // C5 note, 80ms
		48000,
	)
	g.attackSound = audio.NewSoundFromSamples(
		audio.GenerateSineWave(220.0, 0.06, 48000), // A3 note, 60ms
		48000,
	)

	ctx.Logf("pixel-quest booted with tilemap (%dx%d tiles)", g.tileMap.Width, g.tileMap.Height)
	return nil
}

func (g *pixelQuest) initLevel(ctx *whisky.Context) {
	g.world = physics.NewWorld()
	g.rng = rand.New(rand.NewSource(7))
	g.enemies = nil
	g.playerDefeated = false
	g.triggerReached = false
	g.velocity = geom.Vec2{}
	g.grounded = false
	g.jumpsLeft = 2
	g.attackTimer = 0

	// --- Tilemap setup ---
	ts := tilemap.NewTileSet("quest", 16, 16, 4)
	ts.SetProperties(1, tilemap.TileProperties{Solid: true})
	ts.SetProperties(2, tilemap.TileProperties{Solid: true, OneWay: true})
	ts.SetProperties(3, tilemap.TileProperties{
		Trigger: true,
		Tags:    map[string]string{"type": "door"},
	})

	m := tilemap.New(ts, 20, 12)
	m.AddLayer("terrain")
	m.FillRow("terrain", 0, 11, 20, 1)
	m.BuildPlatform("terrain", 3, 8, 5, 2)
	m.BuildPlatform("terrain", 12, 6, 4, 1)
	m.FillCol("terrain", 0, 0, 11, 1)
	m.FillCol("terrain", 19, 0, 11, 1)
	m.SetTile("terrain", 18, 10, 3)
	g.tileMap = m

	_ = g.loadSprites(ctx)

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

	// --- Enemies ---
	g.spawnEnemy(ctx.Scene.Root, "enemy:slime:1", 88, 164, 80, 136, 2, 22, []gameplay.DropEntry{
		{Kind: "xp", MinAmount: 1, MaxAmount: 2, Chance: 1},
		{Kind: "health", MinAmount: 1, MaxAmount: 1, Chance: 0.35},
	})
	g.spawnEnemy(ctx.Scene.Root, "enemy:slime:2", 212, 164, 196, 260, 3, 28, []gameplay.DropEntry{
		{Kind: "xp", MinAmount: 2, MaxAmount: 4, Chance: 1},
		{Kind: "item", ID: "slime_gel", MinAmount: 1, MaxAmount: 1, Chance: 0.5},
	})

	// --- Camera init ---
	g.updateCamera(ctx)
}

func (g *pixelQuest) restartLevel() {
	g.score.Reset()
	g.loadLevel(g.ctx, g.currentLevel)
	g.changeState(statePlaying)
}

func (g *pixelQuest) changeState(newState gameState) {
	g.stateStack = append(g.stateStack, g.state)
	g.state = newState
	// Reset menu cooldowns when entering a menu.
	switch newState {
	case stateTitle:
		if g.titleScreen != nil {
			g.titleScreen.menu.ConfirmDelay = 0.15
		}
	case statePaused:
		if g.pauseScreen != nil {
			g.pauseScreen.menu.ConfirmDelay = 0.15
		}
	case stateOptions:
		if g.optionsScreen != nil {
			g.optionsScreen.menu.ConfirmDelay = 0.15
		}
	case stateControls:
		if g.controlsScreen != nil {
			g.controlsScreen.menu.ConfirmDelay = 0.15
		}
	case stateGameOver:
		if g.gameOverScreen != nil {
			g.gameOverScreen.menu.ConfirmDelay = 0.15
		}
	case stateVictory:
		if g.victoryScreen != nil {
			g.victoryScreen.menu.ConfirmDelay = 0.15
		}
	}
}

func (g *pixelQuest) popState() {
	if len(g.stateStack) > 0 {
		g.state = g.stateStack[len(g.stateStack)-1]
		g.stateStack = g.stateStack[:len(g.stateStack)-1]
	}
}

func (g *pixelQuest) Update(ctx *whisky.Context, dt float64) error {
	g.ctx = ctx
	if g.quitRequested {
		ctx.Quit()
		return nil
	}

	// Reset camera to screen center in menus so UI draws in screen space.
	if g.state != statePlaying && ctx.Camera != nil {
		vw, vh := ctx.VirtualSize()
		ctx.Camera.Position.X = vw / 2
		ctx.Camera.Position.Y = vh / 2
	}

	switch g.state {
	case stateTitle:
		g.titleScreen.Update(g, ctx, dt)
		g.titleScreen.Draw(g, ctx)
		ctx.SetDebugText("Pixel Quest v0.2", "Arrow keys / WASD to navigate, Enter to confirm, Mouse click OK")
	case statePlaying:
		return g.updatePlaying(ctx, dt)
	case statePaused:
		g.pauseScreen.Update(g, ctx, dt)
		g.renderWorld(ctx)
		g.pauseScreen.Draw(g, ctx)
		ctx.SetDebugText("PAUSED", "Esc to resume")
	case stateOptions:
		g.optionsScreen.Update(g, ctx, dt)
		g.optionsScreen.Draw(g, ctx)
		ctx.SetDebugText("OPTIONS")
	case stateControls:
		g.controlsScreen.Update(g, ctx, dt)
		g.controlsScreen.Draw(g, ctx)
		if g.controlsScreen.awaitingAction != "" {
			ctx.SetDebugText("Press any key to bind to "+g.controlsScreen.awaitingAction, "Esc to cancel")
		} else {
			ctx.SetDebugText("CONTROLS", "Select an action and press a key to rebind")
		}
	case stateGameOver:
		g.gameOverScreen.Update(g, ctx, dt)
		g.renderWorld(ctx)
		g.gameOverScreen.Draw(g, ctx)
		ctx.SetDebugText("GAME OVER", formatScoreLine(g.score))
	case stateVictory:
		g.victoryScreen.Update(g, ctx, dt)
		g.renderWorld(ctx)
		g.victoryScreen.Draw(g, ctx)
		ctx.SetDebugText("VICTORY!", formatScoreLine(g.score))
	case stateLevelSelect:
		g.levelSelectScreen.Update(g, ctx, dt)
		g.levelSelectScreen.Draw(g, ctx)
		ctx.SetDebugText("LEVEL SELECT")
	case stateSaveSlots:
		g.saveSlotsScreen.Update(g, ctx, dt)
		g.saveSlotsScreen.Draw(g, ctx)
		ctx.SetDebugText("SAVE SLOTS")
	}
	// Update particles even in menus for visual flair.
	if g.particles != nil {
		g.particles.Update(dt)
		g.particles.Draw(ctx)
	}
	return nil
}

func (g *pixelQuest) updateCamera(ctx *whisky.Context) {
	if ctx.Camera == nil {
		return
	}
	pp := g.player.WorldPosition()
	vw, vh := ctx.VirtualSize()
	targetX := pp.X + playerW/2
	targetY := pp.Y + playerH/2

	// Clamp camera so the viewport stays within the tilemap bounds.
	mapW := float64(g.tileMap.Width * g.tileMap.TileSet.TileWidth)
	mapH := float64(g.tileMap.Height * g.tileMap.TileSet.TileHeight)

	minX := vw / 2
	maxX := mapW - vw/2
	minY := vh / 2
	maxY := mapH - vh/2

	if mapW <= vw {
		// Map is narrower than screen: center horizontally.
		targetX = mapW / 2
	} else {
		if targetX < minX {
			targetX = minX
		}
		if targetX > maxX {
			targetX = maxX
		}
	}

	if mapH <= vh {
		// Map is shorter than screen: center vertically.
		targetY = mapH / 2
	} else {
		if targetY < minY {
			targetY = minY
		}
		if targetY > maxY {
			targetY = maxY
		}
	}

	ctx.Camera.Position.X = targetX
	ctx.Camera.Position.Y = targetY
}

func (g *pixelQuest) updatePlaying(ctx *whisky.Context, dt float64) error {
	g.updateCamera(ctx)
	playerAlive := g.playerHealth == nil || g.playerHealth.Alive()

	// Pause toggle.
	if ctx.Input.JustPressed("menu_back") {
		g.changeState(statePaused)
		return nil
	}

	if g.attackTimer > 0 {
		g.attackTimer -= dt
		if g.attackTimer < 0 {
			g.attackTimer = 0
		}
	}

	// Reset grounded each frame — resolveY sets it back to true on landing.
	wasGrounded := g.grounded
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
	dy := g.velocity.Y * dt
	g.player.Position.Y += dy
	g.resolveY(dy, prevBottom)

	// Landing particles.
	if !wasGrounded && g.grounded && playerAlive {
		g.particles.emitLand(g.player.Position.X, g.player.Position.Y, g.rng)
	}

	// Reset jumpsLeft when landing.
	if g.grounded {
		g.jumpsLeft = 2
	}

	// --- Jump ---
	if playerAlive && g.jumpsLeft > 0 && ctx.Input.JustPressed("jump") {
		g.velocity.Y = jumpVel
		g.jumpsLeft--
		g.particles.emitJump(g.player.Position.X, g.player.Position.Y, g.rng)
		if eng := ctx.Audio(); eng != nil && g.jumpSound != nil {
			eng.Play(g.jumpSound, audio.PlayOpts{Volume: g.config.Volume})
		}
	}

	// Attack with keyboard or mouse left click.
	if playerAlive && (ctx.Input.JustPressed("attack") || ctx.Input.Mouse().JustPressed(input.MouseButtonLeft)) {
		g.attackTimer = playerAttackDuration
		if eng := ctx.Audio(); eng != nil && g.attackSound != nil {
			eng.Play(g.attackSound, audio.PlayOpts{Volume: g.config.Volume})
		}
	}

	// --- Update enemies (AI + projectiles) ---
	g.updateEnemies(ctx, dt, g.particles, g.dropManager, g.score)

	// --- Damage resolution ---
	for _, event := range gameplay.ResolveDamage(g.damageSources(), g.damageTargets()) {
		switch event.TargetID {
		case "player":
			ctx.Logf("player took %d damage (%d/%d hp)", event.Amount, g.playerHealth.Current, g.playerHealth.Max)
			g.particles.emitPlayerHit(g.player.Position.X+playerW/2, g.player.Position.Y+playerH/2, g.rng)
			if !g.playerHealth.Alive() && !g.playerDefeated {
				g.playerDefeated = true
				ctx.Logf("player defeated on frame %d", ctx.Frames)
				g.changeState(stateGameOver)
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
				g.score.AddKill()
				g.particles.emitEnemyDeath(enemy.node.Position.X, enemy.node.Position.Y, g.rng)
				if drops := enemy.drops.Roll(g.rng); len(drops) > 0 {
					ctx.Logf("%s dropped %s", enemy.id, formatDrops(drops))
					g.dropManager.Spawn(enemy.node.Position.X+enemyW/2, enemy.node.Position.Y, drops, g.rng)
				}
				ctx.Logf("%s defeated", enemy.id)
			} else {
				g.particles.emitAttackHit(enemy.node.Position.X+enemyW/2, enemy.node.Position.Y+enemyH/2, g.rng)
			}
		}
	}

	// --- Projectile hits ---
	g.checkProjectileHits(g.score, g.particles, g.dropManager)

	// --- Drops update ---
	pp := g.player.WorldPosition()
	g.dropManager.Update(dt, pp, playerW, playerH, g.score, g.playerHealth, g.particles, g.rng)

	// --- Particles update ---
	g.particles.Update(dt)

	// --- Trigger / victory detection ---
	playerRect := geom.Rect{X: pp.X, Y: pp.Y, W: playerW, H: playerH}
	triggers := g.world.QueryRect(playerRect, physics.LayerTrigger)
	if len(triggers) > 0 && !g.triggerReached {
		g.triggerReached = true
		ctx.Logf("player reached trigger %s on frame %d", triggers[0].ID, ctx.Frames)
		// Unlock next level.
		if g.currentLevel+1 < len(allLevels) {
			g.saveData.UnlockedLevels[g.currentLevel+1] = true
		}
		_ = saveSaveData(g.saveData)
		g.changeState(stateVictory)
	}

	// --- Render world ---
	g.renderWorld(ctx)
	g.dropManager.Draw(ctx)
	g.drawProjectiles(ctx)
	g.particles.Draw(ctx)

	// --- HUD ---
	vw, _ := ctx.VirtualSize()
	if g.playerHealth != nil {
		g.drawHealthBar(ctx, 4, 4, 40, g.playerHealth)
	}
	// Score line background.
	scoreText := formatScoreLine(g.score)
	ctx.DrawRect(geom.Rect{X: 2, Y: 2, W: 44, H: 6}, geom.RGBA(0.1, 0.1, 0.12, 0.7))
	ctx.DrawRect(geom.Rect{X: vw - 90, Y: 2, W: 88, H: 6}, geom.RGBA(0.1, 0.1, 0.12, 0.7))

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

	ctx.SetDebugText(
		"A/D move   Space/W/Up jump   J/K/LMB attack   LShift sprint   Esc=menu",
		fmt.Sprintf("player=(%.0f,%.0f) vel=(%.0f,%.0f)  %s", pp.X, pp.Y, g.velocity.X, g.velocity.Y, status),
		fmt.Sprintf("%s  hp=%d/%d  enemies=%d  parts=%d", scoreText, g.playerHealth.Current, g.playerHealth.Max, g.aliveEnemies(), g.particles.Count()),
	)
	return nil
}

func (g *pixelQuest) renderWorld(ctx *whisky.Context) {
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
	pp := g.player.WorldPosition()
	if g.playerSheet == nil {
		color := geom.RGBA(0.2, 0.8, 0.3, 1)
		if g.playerDefeated {
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
		g.drawEnemyByType(ctx, enemy)
	}
	if g.attackTimer > 0 && (g.playerHealth == nil || g.playerHealth.Alive()) {
		ctx.DrawRect(g.playerAttackBox().Rect(), geom.RGBA(1.0, 0.85, 0.2, 0.55))
	}
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
	_ = saveGameConfig(g.config)
	ctx.Logf("pixel-quest shutdown")
	return nil
}

func (g *pixelQuest) loadSprites(ctx *whisky.Context) error {
	tileTexture, _, _, err := ctx.LoadTexture("tiles.png")
	if err != nil {
		return err
	}
	playerTexture, playerWpx, playerHpx, err := ctx.LoadTexture("player.png")
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
