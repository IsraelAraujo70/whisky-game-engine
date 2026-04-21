package whisky

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/IsraelAraujo70/whisky-game-engine/assets"
	"github.com/IsraelAraujo70/whisky-game-engine/audio"
	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/input"
	backendapi "github.com/IsraelAraujo70/whisky-game-engine/internal/backend"
	platformapi "github.com/IsraelAraujo70/whisky-game-engine/internal/platform"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
	"github.com/IsraelAraujo70/whisky-game-engine/scene"
)

var ErrQuit = errors.New("whisky: quit requested")

type Game interface {
	Load(ctx *Context) error
	Update(ctx *Context, dt float64) error
	Shutdown(ctx *Context) error
}

// KeyMap maps key names (e.g. "space", "w", "up") to semantic control names
// used by the input system (e.g. "jump", "move_up"). If nil, a default set of
// semantic controls is used: w/up → "move_up", s/down → "move_down",
// a/left → "move_left", d/right → "move_right", space → "action",
// lshift → "sprint", enter → "confirm".
//
// Supported key names: letter keys ("a"–"z"), digit keys ("0"–"9"), arrow
// keys ("up", "down", "left", "right"), and named keys ("space", "enter",
// "escape", "lshift", "rshift", "lctrl", "rctrl", "tab", "backspace").
type KeyMap map[string]string

type Config struct {
	Title         string
	WindowWidth   int
	WindowHeight  int
	VirtualWidth  int
	VirtualHeight int
	PixelPerfect  bool
	VSync         bool
	TargetFPS     int
	MaxFrames     int
	ClearColor    geom.Color
	StartScene    *scene.Scene
	Headless      bool
	// KeyMap defines the scancode-to-control mapping for the platform layer.
	// Nil means use the built-in defaults.
	KeyMap KeyMap
	// GravityY is the downward acceleration applied per second (px/s²).
	// Zero means no gravity. Games read this via ctx.Config.GravityY.
	GravityY float64
	// Audio configures the audio engine. If Audio.Enabled is false (or left
	// at the zero value), audio is initialised with sensible defaults
	// (enabled, 32 channels, 48 kHz).
	Audio audio.Config
	// AssetsRoot is the directory used to resolve relative asset paths.
	// Empty means "assets" in the working directory.
	AssetsRoot string
	// HotReload starts an fsnotify watcher on AssetsRoot during development.
	// When a file changes the engine invalidates the cache entry and re-uploads
	// GPU textures automatically on the main thread.
	HotReload bool
	// CacheMaxSize is the LRU limit for the asset cache. Zero or negative
	// means unlimited.
	CacheMaxSize int
}

type Context struct {
	Config Config
	Input  *input.State
	Scene  *scene.Scene
	Camera *render.Camera2D
	Delta  float64
	Frames int
	Assets *assets.Cache
	logger *log.Logger
	quit   bool

	platform      platformapi.Platform
	renderer      platformapi.Renderer
	backend       platformapi.Backend
	audioEngine   *audio.Engine
	assetWatcher  *assets.Watcher
	reloadQueue   chan string
	debugLines    []string
	drawCmds      []render.DrawCmd
	texSeq        render.TextureID
}

// Audio returns the audio engine, or nil if audio is disabled.
func (c *Context) Audio() *audio.Engine {
	return c.audioEngine
}

func (c *Context) Quit() {
	c.quit = true
}

func (c *Context) ShouldQuit() bool {
	return c.quit
}

func (c *Context) Logf(format string, args ...any) {
	c.logger.Printf(format, args...)
}

func (c *Context) SetDebugText(lines ...string) {
	c.debugLines = append(c.debugLines[:0], lines...)
}

// Mouse returns the mouse input state. Never nil.
func (c *Context) Mouse() *input.MouseState {
	return c.Input.Mouse()
}

// Gamepad returns the gamepad input state for the given slot
// (0 to input.MaxGamepads-1). Never nil, but may be disconnected.
func (c *Context) Gamepad(index int) *input.GamepadState {
	return c.Input.Gamepad(index)
}

func (c *Context) LoadTexture(path string) (render.TextureID, int, int, error) {
	if c.renderer == nil {
		c.texSeq++
		return c.texSeq, 0, 0, nil
	}

	rel, abs, err := c.resolveAssetPath(path)
	if err != nil {
		return 0, 0, 0, err
	}

	ct, err := assets.Get(c.Assets, rel, func(_ string) (cachedTexture, error) {
		id, w, h, err := c.renderer.LoadTexture(abs)
		if err != nil {
			return cachedTexture{}, err
		}
		return cachedTexture{id: id, width: w, height: h}, nil
	})
	if err != nil {
		return 0, 0, 0, err
	}
	return ct.id, ct.width, ct.height, nil
}

type cachedTexture struct {
	id     render.TextureID
	width  int
	height int
}

func (c *Context) resolveAssetPath(path string) (rel string, abs string, err error) {
	if filepath.IsAbs(path) {
		abs = path
		rel, err = filepath.Rel(c.Assets.AssetsRoot, abs)
		if err != nil {
			return "", "", err
		}
		rel = filepath.ToSlash(rel)
		return rel, abs, nil
	}
	rel = filepath.ToSlash(path)
	abs = filepath.Join(c.Assets.AssetsRoot, rel)
	return rel, abs, nil
}

func (c *Context) VirtualSize() (w, h float64) {
	return float64(c.Config.VirtualWidth), float64(c.Config.VirtualHeight)
}

func (c *Context) ViewportRect() geom.Rect {
	vw, vh := c.VirtualSize()
	if c.Camera == nil {
		return geom.Rect{W: vw, H: vh}
	}
	return c.Camera.ViewportRect(vw, vh)
}

// DrawRect queues a filled rectangle in world coordinates. The camera
// transform is applied automatically so callers work in world space.
// If Camera is nil the rectangle is drawn as-is (screen space).
func (c *Context) DrawRect(worldRect geom.Rect, color geom.Color) {
	r := worldRect
	if c.Camera != nil {
		vw, vh := c.VirtualSize()
		screenPos := c.Camera.WorldToScreen(
			geom.Vec2{X: worldRect.X, Y: worldRect.Y}, vw, vh,
		)
		r = geom.Rect{X: screenPos.X, Y: screenPos.Y, W: worldRect.W, H: worldRect.H}
	}
	c.drawCmds = append(c.drawCmds, render.FillRect{
		Rect:  r,
		Color: color,
	})
}

// DrawText queues a text string in world coordinates. The camera transform is
// applied automatically, just like DrawRect. Scale multiplies the base glyph
// dimensions (1.0 = native font size).
func (c *Context) DrawText(text string, worldPos geom.Vec2, color geom.Color, scale float64) {
	pos := worldPos
	if c.Camera != nil {
		vw, vh := c.VirtualSize()
		pos = c.Camera.WorldToScreen(worldPos, vw, vh)
	}
	if scale <= 0 {
		scale = 1
	}
	c.drawCmds = append(c.drawCmds, render.TextCmd{
		Text:  text,
		Pos:   pos,
		Color: color,
		Scale: scale,
	})
}

func (c *Context) DrawSprite(texture render.TextureID, src, dst geom.Rect, flipH, flipV bool) {
	drawDst := dst
	if c.Camera != nil {
		vw, vh := c.VirtualSize()
		screenPos := c.Camera.WorldToScreen(geom.Vec2{X: dst.X, Y: dst.Y}, vw, vh)
		drawDst = geom.Rect{
			X: screenPos.X,
			Y: screenPos.Y,
			W: dst.W,
			H: dst.H,
		}
	}

	c.drawCmds = append(c.drawCmds, render.SpriteCmd{
		Texture: texture,
		Src:     src,
		Dst:     drawDst,
		FlipH:   flipH,
		FlipV:   flipV,
	})
}

// DisplayController is a re-export of the platform interface for game-level access.
type WindowMode = platformapi.WindowMode

const (
	WindowModeWindowed   = platformapi.WindowModeWindowed
	WindowModeBorderless = platformapi.WindowModeBorderless
	WindowModeFullscreen = platformapi.WindowModeFullscreen
)

// DisplayMode describes a supported resolution and refresh rate.
type DisplayMode = platformapi.DisplayMode

// MonitorInfo describes a connected display.
type MonitorInfo = platformapi.MonitorInfo

// displayController is the interface used by Context to control window display.
type displayController interface {
	SetWindowSize(width, height int) error
	SetWindowMode(mode platformapi.WindowMode) error
	Monitors() ([]platformapi.MonitorInfo, error)
	MoveToMonitor(index int) error
}

// SetWindowSize resizes the OS window to the given dimensions.
func (c *Context) SetWindowSize(width, height int) error {
	dc := c.getDisplayController()
	if dc == nil {
		return platformapi.ErrNotSupported
	}
	return dc.SetWindowSize(width, height)
}

// SetWindowMode changes the window mode (windowed, borderless fullscreen, etc.).
func (c *Context) SetWindowMode(mode WindowMode) error {
	dc := c.getDisplayController()
	if dc == nil {
		return platformapi.ErrNotSupported
	}
	return dc.SetWindowMode(mode)
}

// Monitors returns the list of connected monitors with their supported modes.
func (c *Context) Monitors() ([]MonitorInfo, error) {
	dc := c.getDisplayController()
	if dc == nil {
		return nil, platformapi.ErrNotSupported
	}
	return dc.Monitors()
}

// MoveToMonitor moves the window to the specified monitor by index.
func (c *Context) MoveToMonitor(index int) error {
	dc := c.getDisplayController()
	if dc == nil {
		return platformapi.ErrNotSupported
	}
	return dc.MoveToMonitor(index)
}

func (c *Context) getDisplayController() displayController {
	if c.backend == nil {
		return nil
	}
	dc, ok := c.backend.(displayController)
	if !ok {
		return nil
	}
	return dc
}

func Run(game Game, cfg Config) (err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cfg = withDefaults(cfg)
	ctx := &Context{
		Config: cfg,
		Input:  input.NewState(),
		Scene:  cfg.StartScene,
		Camera: &render.Camera2D{
			Position: geom.Vec2{
				X: float64(cfg.VirtualWidth) / 2,
				Y: float64(cfg.VirtualHeight) / 2,
			},
		},
		Delta:  1.0 / float64(cfg.TargetFPS),
		logger: log.New(os.Stdout, "[whisky] ", 0),
	}

	if ctx.Scene == nil {
		ctx.Scene = scene.New(cfg.Title)
	}

	// --- Asset cache & hot-reload watcher ---
	ctx.Assets = assets.NewCache(cfg.CacheMaxSize)
	absRoot, err := filepath.Abs(cfg.AssetsRoot)
	if err != nil {
		return err
	}
	ctx.Assets.AssetsRoot = absRoot
	ctx.reloadQueue = make(chan string, 64)

	if cfg.HotReload {
		watcher, werr := assets.NewWatcher(absRoot, ctx.Assets, ctx.logger)
		if werr != nil {
			return werr
		}
		if watcher != nil {
			ctx.assetWatcher = watcher
			watcher.SetOnReload(func(relPath string) {
				select {
				case ctx.reloadQueue <- relPath:
				default:
					ctx.logger.Printf("[assets] reload queue full, dropping %s", relPath)
				}
			})
			defer func() { _ = watcher.Close() }()
		}
	}

	var backend platformapi.Backend
	if !cfg.Headless && os.Getenv("WHISKY_HEADLESS") != "1" {
		backend, err = backendapi.NewDesktop(cfg.Title, cfg.WindowWidth, cfg.WindowHeight, map[string]string(cfg.KeyMap))
		if err != nil {
			return err
		}
		if err = backend.SetLogicalSize(cfg.VirtualWidth, cfg.VirtualHeight, cfg.PixelPerfect); err != nil {
			_ = backend.Destroy()
			return err
		}
		defer func() {
			destroyErr := backend.Destroy()
			if err == nil {
				err = destroyErr
			}
		}()
	}
	ctx.platform = backend
	ctx.renderer = backend
	ctx.backend = backend

	// --- Audio engine ---
	audioEngine, audioErr := audio.Init(cfg.Audio)
	if audioErr != nil {
		ctx.logger.Printf("audio init failed (continuing without audio): %v", audioErr)
	} else {
		ctx.audioEngine = audioEngine
		defer func() {
			if shutErr := audioEngine.Shutdown(); shutErr != nil && err == nil {
				err = shutErr
			}
		}()
	}

	if err := game.Load(ctx); err != nil {
		return err
	}

	defer func() {
		shutdownErr := game.Shutdown(ctx)
		if err == nil {
			err = shutdownErr
		}
	}()

	ticker := time.NewTicker(time.Second / time.Duration(cfg.TargetFPS))
	defer ticker.Stop()

	for {
		if ctx.platform != nil {
			ctx.platform.UpdateInput(ctx.Input)
			if ctx.platform.PumpEvents() {
				ctx.Quit()
			}
		}

		if ctx.ShouldQuit() {
			return nil
		}

		if cfg.MaxFrames > 0 && ctx.Frames >= cfg.MaxFrames {
			return nil
		}

		// Drain hot-reload queue on the main thread
	drain:
		for {
			select {
			case relPath := <-ctx.reloadQueue:
				ctx.Assets.Invalidate(relPath)
				if filepath.Ext(relPath) == ".png" {
					if _, _, _, err := ctx.LoadTexture(relPath); err != nil {
						ctx.logger.Printf("[assets] hot-reload failed for %s: %v", relPath, err)
					} else {
						ctx.logger.Printf("[assets] hot-reloaded %s", relPath)
					}
				}
			default:
				break drain
			}
		}

		if err := ctx.Scene.Update(ctx.Delta); err != nil {
			return err
		}

		if err := game.Update(ctx, ctx.Delta); err != nil {
			if errors.Is(err, ErrQuit) {
				ctx.Quit()
				continue
			}
			return err
		}

		ctx.Scene.Draw(ctx)

		if ctx.renderer != nil {
			if err := ctx.renderer.DrawFrame(ctx.Config.ClearColor, ctx.drawCmds, ctx.overlayLines()); err != nil {
				return err
			}
		}
		ctx.drawCmds = ctx.drawCmds[:0]

		ctx.Input.NextFrame()
		ctx.Frames++
		<-ticker.C
	}
}

func withDefaults(cfg Config) Config {
	if cfg.Title == "" {
		cfg.Title = "whisky game"
	}
	if cfg.WindowWidth == 0 {
		cfg.WindowWidth = 1280
	}
	if cfg.WindowHeight == 0 {
		cfg.WindowHeight = 720
	}
	if cfg.VirtualWidth == 0 {
		cfg.VirtualWidth = 320
	}
	if cfg.VirtualHeight == 0 {
		cfg.VirtualHeight = 180
	}
	if cfg.TargetFPS == 0 {
		cfg.TargetFPS = 60
	}
	if cfg.ClearColor == (geom.Color{}) {
		cfg.ClearColor = geom.RGBA(0.08, 0.08, 0.1, 1)
	}

	// Audio defaults: enabled with 32 channels at 48 kHz.
	if !cfg.Audio.Enabled && cfg.Audio.Channels == 0 && cfg.Audio.SampleRate == 0 {
		cfg.Audio.Enabled = true
	}

	if cfg.AssetsRoot == "" {
		cfg.AssetsRoot = "assets"
	}

	if cfg.KeyMap == nil {
		cfg.KeyMap = KeyMap{
			"w":      "move_up",
			"up":     "move_up",
			"s":      "move_down",
			"down":   "move_down",
			"a":      "move_left",
			"left":   "move_left",
			"d":      "move_right",
			"right":  "move_right",
			"space":  "action",
			"lshift": "sprint",
			"enter":  "confirm",
		}
	}

	return cfg
}

func (c *Context) overlayLines() []string {
	lines := []string{
		c.Config.Title,
		"Esc closes the window",
		"Frames: " + strconv.Itoa(c.Frames),
	}

	if len(c.debugLines) > 0 {
		lines = append(lines, c.debugLines...)
	}

	return lines
}
