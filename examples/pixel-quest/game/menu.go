package game

import (
	"fmt"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	whisky "github.com/IsraelAraujo70/whisky-game-engine/whisky"
)

// gameState represents the current high-level screen.
type gameState int

const (
	stateTitle gameState = iota
	statePlaying
	statePaused
	stateOptions
	stateControls
	stateGameOver
	stateVictory
	stateLevelSelect
	stateSaveSlots
)

// screenTitle handles the title screen.
type screenTitle struct {
	menu *uiMenu
}

func newScreenTitle(g *pixelQuest) *screenTitle {
	s := &screenTitle{menu: newUIMenu()}
	s.menu.AddButton("New Game", func() {
		g.saveSlotsScreen = newScreenSaveSlots(g, "new")
		g.changeState(stateSaveSlots)
	})
	s.menu.AddButton("Continue", func() {
		if g.saveData.LastUsedSlot >= 0 && g.saveData.LastUsedSlot < len(g.saveData.Slots) {
			g.applySnapshot(g.saveData.Slots[g.saveData.LastUsedSlot])
			g.loadLevel(g.ctx, g.currentLevel)
			g.changeState(statePlaying)
		}
	})
	s.menu.AddButton("Level Select", func() {
		g.levelSelectScreen = newScreenLevelSelect(g)
		g.changeState(stateLevelSelect)
	})
	s.menu.AddButton("Options", func() { g.changeState(stateOptions) })
	s.menu.AddButton("Quit", func() { g.quitRequested = true })
	return s
}

func (s *screenTitle) Update(g *pixelQuest, ctx *whisky.Context, dt float64) {
	s.menu.Update(ctx, dt)
}

func (s *screenTitle) Draw(g *pixelQuest, ctx *whisky.Context) {
	vw, vh := ctx.VirtualSize()
	// Background pattern (simple grid)
	for x := 0.0; x < vw; x += 20 {
		ctx.DrawRect(geom.Rect{X: x, Y: 0, W: 1, H: vh}, geom.RGBA(0.06, 0.06, 0.08, 1))
	}
	for y := 0.0; y < vh; y += 20 {
		ctx.DrawRect(geom.Rect{X: 0, Y: y, W: vw, H: 1}, geom.RGBA(0.06, 0.06, 0.08, 1))
	}

	// Title banner
	bannerH := 24.0
	uiTitle(ctx, 30, 16, vw-60, bannerH, "Pixel Quest")

	// Menu
	s.menu.LayoutCentered(50, 80, 14, 4, vw, vh)
	s.menu.Draw(ctx)

	// Version footer
	ctx.DrawRect(geom.Rect{X: 0, Y: vh - 10, W: vw, H: 10}, geom.RGBA(0.06, 0.06, 0.08, 0.8))
}

// screenPause handles the in-game pause menu.
type screenPause struct {
	menu *uiMenu
}

func newScreenPause(g *pixelQuest) *screenPause {
	s := &screenPause{menu: newUIMenu()}
	s.menu.AddButton("Resume", func() { g.changeState(statePlaying) })
	s.menu.AddButton("Options", func() { g.changeState(stateOptions) })
	s.menu.AddButton("Quit to Title", func() { g.changeState(stateTitle) })
	return s
}

func (s *screenPause) Update(g *pixelQuest, ctx *whisky.Context, dt float64) {
	s.menu.Update(ctx, dt)
}

func (s *screenPause) Draw(g *pixelQuest, ctx *whisky.Context) {
	vw, vh := ctx.VirtualSize()
	// Dim overlay
	ctx.DrawRect(geom.Rect{X: 0, Y: 0, W: vw, H: vh}, geom.RGBA(0, 0, 0, 0.55))
	// Panel
	panelW, panelH := 140.0, 90.0
	px := (vw - panelW) / 2
	py := (vh - panelH) / 2
	uiPanel(ctx, px, py, panelW, panelH)
	uiTitle(ctx, px, py, panelW, 20, "PAUSED")

	s.menu.LayoutCentered(py+28, 100, 16, 5, vw, vh)
	s.menu.Draw(ctx)
}

// screenOptions handles the settings screen.
type screenOptions struct {
	menu       *uiMenu
	sliderVol  *uiSlider
	changingKB bool
}

func newScreenOptions(g *pixelQuest) *screenOptions {
	s := &screenOptions{menu: newUIMenu()}
	s.sliderVol = &uiSlider{
		Label:    "Volume",
		Min:      0,
		Max:      1,
		Step:     0.05,
		Value:    g.config.Volume,
		OnChange: func(v float64) { g.config.Volume = v },
	}
	s.menu.AddButton("Difficulty: "+formatDifficulty(g.config.Difficulty), func() {
		g.config.Difficulty = (g.config.Difficulty + 1) % 3
		s.refreshLabels(g)
	})

	// Resolution cycling button.
	resW, resH := g.config.WindowWidth, g.config.WindowHeight
	if resW == 0 {
		resW = 1280
	}
	if resH == 0 {
		resH = 720
	}
	s.menu.AddButton(fmt.Sprintf("Resolution: %s", formatResolution(resW, resH)), func() {
		idx := findResolutionIndex(g.config.WindowWidth, g.config.WindowHeight)
		idx = (idx + 1) % len(commonResolutions)
		g.config.WindowWidth = commonResolutions[idx][0]
		g.config.WindowHeight = commonResolutions[idx][1]
		s.refreshLabels(g)
	})

	// Window mode cycling button.
	s.menu.AddButton(fmt.Sprintf("Display: %s", formatWindowMode(g.config.WindowMode)), func() {
		g.config.WindowMode = (g.config.WindowMode + 1) % 2
		s.refreshLabels(g)
	})

	s.menu.AddButton("Controls", func() { g.changeState(stateControls) })
	s.menu.AddButton("Save & Back", func() {
		_ = saveGameConfig(g.config)
		applyDisplayConfigForce(g.ctx, g.config)
		g.popState()
	})
	return s
}

func (s *screenOptions) refreshLabels(g *pixelQuest) {
	if len(s.menu.Buttons) > 0 {
		s.menu.Buttons[0].Text = "Difficulty: " + formatDifficulty(g.config.Difficulty)
	}
	if len(s.menu.Buttons) > 1 {
		resW, resH := g.config.WindowWidth, g.config.WindowHeight
		if resW == 0 {
			resW = 1280
		}
		if resH == 0 {
			resH = 720
		}
		s.menu.Buttons[1].Text = fmt.Sprintf("Resolution: %s", formatResolution(resW, resH))
	}
	if len(s.menu.Buttons) > 2 {
		s.menu.Buttons[2].Text = fmt.Sprintf("Display: %s", formatWindowMode(g.config.WindowMode))
	}
}

func (s *screenOptions) Update(g *pixelQuest, ctx *whisky.Context, dt float64) {
	s.sliderVol.Update(ctx)
	s.menu.Update(ctx, dt)
	if ctx.Input.JustPressed("menu_back") {
		_ = saveGameConfig(g.config)
		applyDisplayConfigForce(ctx, g.config)
		g.popState()
	}
}

func (s *screenOptions) Draw(g *pixelQuest, ctx *whisky.Context) {
	vw, vh := ctx.VirtualSize()
	ctx.DrawRect(geom.Rect{X: 0, Y: 0, W: vw, H: vh}, geom.RGBA(0.02, 0.02, 0.04, 0.85))
	panelW, panelH := 180.0, 140.0
	px := (vw - panelW) / 2
	py := (vh - panelH) / 2
	uiPanel(ctx, px, py, panelW, panelH)
	uiTitle(ctx, px, py, panelW, 20, "OPTIONS")

	// Volume slider
	s.sliderVol.Rect = geom.Rect{X: px + 60, Y: py + 28, W: 100, H: 8}
	s.sliderVol.Draw(ctx)

	s.menu.LayoutCentered(py+44, 140, 14, 4, vw, vh)
	s.menu.Draw(ctx)
}

// findResolutionIndex returns the index in commonResolutions matching the given dimensions.
func findResolutionIndex(w, h int) int {
	if w == 0 {
		w = 1280
	}
	if h == 0 {
		h = 720
	}
	for i, r := range commonResolutions {
		if r[0] == w && r[1] == h {
			return i
		}
	}
	return 0
}

// applyDisplayConfig applies the display settings from the game config to the engine.
// It is safe to call during Load (skips no-op operations when config is at defaults).
func applyDisplayConfig(ctx *whisky.Context, cfg GameConfig) {
	if ctx == nil {
		return
	}

	// Apply window mode only if non-default.
	if cfg.WindowMode == 1 {
		_ = ctx.SetWindowMode(whisky.WindowModeBorderless)
	}

	// Apply resolution (only in windowed mode and if explicitly configured).
	if cfg.WindowMode == 0 && cfg.WindowWidth > 0 && cfg.WindowHeight > 0 {
		_ = ctx.SetWindowSize(cfg.WindowWidth, cfg.WindowHeight)
	}

	// Apply monitor selection.
	if cfg.MonitorIndex > 0 {
		_ = ctx.MoveToMonitor(cfg.MonitorIndex)
	}
}

// applyDisplayConfigForce always applies the display settings, including restoring
// windowed mode. Used from the Options screen where the user may be toggling modes.
func applyDisplayConfigForce(ctx *whisky.Context, cfg GameConfig) {
	if ctx == nil {
		return
	}

	// Apply window mode.
	if cfg.WindowMode == 1 {
		if err := ctx.SetWindowMode(whisky.WindowModeBorderless); err != nil {
			ctx.Logf("display: SetWindowMode(borderless) failed: %v", err)
		}
	} else {
		if err := ctx.SetWindowMode(whisky.WindowModeWindowed); err != nil {
			ctx.Logf("display: SetWindowMode(windowed) failed: %v", err)
		}
	}

	// Apply resolution (only in windowed mode).
	if cfg.WindowMode == 0 {
		w, h := cfg.WindowWidth, cfg.WindowHeight
		if w == 0 {
			w = 1280
		}
		if h == 0 {
			h = 720
		}
		if err := ctx.SetWindowSize(w, h); err != nil {
			ctx.Logf("display: SetWindowSize(%d, %d) failed: %v", w, h, err)
		}
	}

	// Apply monitor selection.
	if cfg.MonitorIndex > 0 {
		if err := ctx.MoveToMonitor(cfg.MonitorIndex); err != nil {
			ctx.Logf("display: MoveToMonitor(%d) failed: %v", cfg.MonitorIndex, err)
		}
	}
}

// screenControls handles the key-bindings screen.
type screenControls struct {
	menu           *uiMenu
	actions        []string
	awaitingAction string // which action is waiting for a key press
}

func newScreenControls(g *pixelQuest) *screenControls {
	s := &screenControls{menu: newUIMenu()}
	s.actions = []string{"move_left", "move_right", "jump", "attack", "sprint"}
	for _, action := range s.actions {
		a := action // capture
		s.menu.AddButton(fmt.Sprintf("%s: %s", action, formatControls(g.config.KeyMap[a])), func() {
			s.awaitingAction = a
		})
	}
	s.menu.AddButton("Reset to Defaults", func() {
		g.config.KeyMap = defaultGameConfig().KeyMap
		s.refreshLabels(g)
		applyKeyMap(ctxRef, g.config.KeyMap)
	})
	s.menu.AddButton("Back", func() { g.popState() })
	return s
}

var ctxRef *whisky.Context // temporary reference for closures

func (s *screenControls) refreshLabels(g *pixelQuest) {
	for i, action := range s.actions {
		if i < len(s.menu.Buttons) {
			s.menu.Buttons[i].Text = fmt.Sprintf("%s: %s", action, formatControls(g.config.KeyMap[action]))
		}
	}
}

func (s *screenControls) Update(g *pixelQuest, ctx *whisky.Context, dt float64) {
	ctxRef = ctx
	if s.awaitingAction != "" {
		// Wait for any control press.
		control, ok := ctx.Input.AnyControlJustPressed()
		if ok && control != "" {
			g.config.KeyMap[s.awaitingAction] = []string{control}
			s.awaitingAction = ""
			s.refreshLabels(g)
			applyKeyMap(ctx, g.config.KeyMap)
		}
		return // block other input while waiting
	}
	s.menu.Update(ctx, dt)
	if ctx.Input.JustPressed("menu_back") {
		g.popState()
	}
}

func (s *screenControls) Draw(g *pixelQuest, ctx *whisky.Context) {
	vw, vh := ctx.VirtualSize()
	ctx.DrawRect(geom.Rect{X: 0, Y: 0, W: vw, H: vh}, geom.RGBA(0.02, 0.02, 0.04, 0.9))
	panelW, panelH := 200.0, 140.0
	px := (vw - panelW) / 2
	py := (vh - panelH) / 2
	uiPanel(ctx, px, py, panelW, panelH)
	uiTitle(ctx, px, py, panelW, 20, "CONTROLS")

	s.menu.LayoutCentered(py+26, 170, 14, 4, vw, vh)
	s.menu.Draw(ctx)

	if s.awaitingAction != "" {
		// Modal prompt
		promptW, promptH := 160.0, 30.0
		ppx := (vw - promptW) / 2
		ppy := (vh - promptH) / 2
		uiPanel(ctx, ppx, ppy, promptW, promptH)
		uiTitle(ctx, ppx, ppy, promptW, 14, "Press a key...")
	}
}

// screenGameOver handles the death screen.
type screenGameOver struct {
	menu *uiMenu
}

func newScreenGameOver(g *pixelQuest) *screenGameOver {
	s := &screenGameOver{menu: newUIMenu()}
	s.menu.AddButton("Retry", func() { g.restartLevel() })
	s.menu.AddButton("Quit to Title", func() { g.changeState(stateTitle) })
	return s
}

func (s *screenGameOver) Update(g *pixelQuest, ctx *whisky.Context, dt float64) {
	s.menu.Update(ctx, dt)
}

func (s *screenGameOver) Draw(g *pixelQuest, ctx *whisky.Context) {
	vw, vh := ctx.VirtualSize()
	ctx.DrawRect(geom.Rect{X: 0, Y: 0, W: vw, H: vh}, geom.RGBA(0.15, 0.02, 0.02, 0.75))
	panelW, panelH := 140.0, 70.0
	px := (vw - panelW) / 2
	py := (vh - panelH) / 2
	uiPanel(ctx, px, py, panelW, panelH)
	uiTitle(ctx, px, py, panelW, 20, "GAME OVER")

	s.menu.LayoutCentered(py+28, 100, 16, 5, vw, vh)
	s.menu.Draw(ctx)
}

// screenVictory handles the win screen.
type screenVictory struct {
	menu *uiMenu
}

func newScreenVictory(g *pixelQuest) *screenVictory {
	s := &screenVictory{menu: newUIMenu()}
	s.menu.AddButton("Play Again", func() { g.restartLevel() })
	s.menu.AddButton("Quit to Title", func() { g.changeState(stateTitle) })
	return s
}

func (s *screenVictory) Update(g *pixelQuest, ctx *whisky.Context, dt float64) {
	s.menu.Update(ctx, dt)
}

func (s *screenVictory) Draw(g *pixelQuest, ctx *whisky.Context) {
	vw, vh := ctx.VirtualSize()
	ctx.DrawRect(geom.Rect{X: 0, Y: 0, W: vw, H: vh}, geom.RGBA(0.02, 0.10, 0.02, 0.75))
	panelW, panelH := 140.0, 70.0
	px := (vw - panelW) / 2
	py := (vh - panelH) / 2
	uiPanel(ctx, px, py, panelW, panelH)
	uiTitle(ctx, px, py, panelW, 20, "VICTORY!")

	s.menu.LayoutCentered(py+28, 100, 16, 5, vw, vh)
	s.menu.Draw(ctx)
}
