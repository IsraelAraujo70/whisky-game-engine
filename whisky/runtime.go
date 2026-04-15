package whisky

import (
	"errors"
	"image"
	_ "image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/IsraelAraujo70/whisky-game-engine/assets"
	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/input"
	"github.com/IsraelAraujo70/whisky-game-engine/internal/platform/sdl3"
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

	// HotReload enables the file-system watcher that automatically invalidates
	// cached assets and reuploads GPU textures when files under AssetsRoot
	// change on disk. Default is true.
	HotReload *bool

	// AssetsRoot is the root directory for game assets. When empty, hot-reload
	// is disabled but the asset cache is still available.
	AssetsRoot string

	// CacheMaxSize sets the maximum number of entries in the asset cache.
	// Zero or negative means unlimited. Default is 256.
	CacheMaxSize int
}

// hotReloadEnabled returns true if hot-reload should be enabled.
func (c Config) hotReloadEnabled() bool {
	if c.HotReload != nil {
		return *c.HotReload
	}
	return true // default on
}

type Context struct {
	Config Config
	Input  *input.State
	Scene  *scene.Scene
	Camera *render.Camera2D
	Delta  float64
	Frames int
	logger *log.Logger
	quit   bool

	platform   *sdl3.Runtime
	debugLines []string
	drawCmds   []render.DrawCmd
	texSeq     render.TextureID
	cache      *assets.Cache
	watcher    *assets.Watcher
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

// Assets returns the engine's asset cache. It is always non-nil after Run
// begins, regardless of whether hot-reload is enabled.
func (c *Context) Assets() *assets.Cache {
	return c.cache
}

// textureResult is the cached representation of a loaded texture.
type textureResult struct {
	ID     render.TextureID
	Width  int
	Height int
}

// LoadTexture loads a PNG (or JPEG) image from path into a GPU texture. Results
// are cached by the asset cache so repeated calls with the same path are free.
func (c *Context) LoadTexture(path string) (render.TextureID, int, int, error) {
	if c.platform == nil {
		c.texSeq++
		return c.texSeq, 0, 0, nil
	}

	// Compute cache key: relative to AssetsRoot when possible, otherwise absolute.
	cacheKey := path
	if c.cache != nil && c.cache.AssetsRoot != "" {
		if absPath, err := filepath.Abs(path); err == nil {
			if rel, err := filepath.Rel(c.cache.AssetsRoot, absPath); err == nil && !strings.HasPrefix(rel, "..") {
				cacheKey = filepath.ToSlash(rel)
			}
		}
	}

	if c.cache != nil {
		result, err := assets.Get(c.cache, cacheKey, func(_ string) (textureResult, error) {
			id, w, h, loadErr := c.platform.LoadTexture(path)
			if loadErr != nil {
				return textureResult{}, loadErr
			}
			return textureResult{ID: id, Width: w, Height: h}, nil
		})
		if err != nil {
			return 0, 0, 0, err
		}
		return result.ID, result.Width, result.Height, nil
	}

	return c.platform.LoadTexture(path)
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

func Run(game Game, cfg Config) (err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cfg = withDefaults(cfg)
	logger := log.New(os.Stdout, "[whisky] ", 0)

	// --- Asset cache ---
	cache := assets.NewCache(cfg.CacheMaxSize)
	if cfg.AssetsRoot != "" {
		if abs, absErr := filepath.Abs(cfg.AssetsRoot); absErr == nil {
			cache.AssetsRoot = abs
		}
	}

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
		logger: logger,
		cache:  cache,
	}

	if ctx.Scene == nil {
		ctx.Scene = scene.New(cfg.Title)
	}

	var platform *sdl3.Runtime
	if !cfg.Headless && os.Getenv("WHISKY_HEADLESS") != "1" {
		platform, err = sdl3.New(cfg.Title, cfg.WindowWidth, cfg.WindowHeight, map[string]string(cfg.KeyMap))
		if err != nil {
			return err
		}
		if err = platform.SetLogicalSize(cfg.VirtualWidth, cfg.VirtualHeight, cfg.PixelPerfect); err != nil {
			_ = platform.Destroy()
			return err
		}
		defer func() {
			destroyErr := platform.Destroy()
			if err == nil {
				err = destroyErr
			}
		}()
	}
	ctx.platform = platform

	// --- Hot-reload watcher ---
	if cfg.hotReloadEnabled() && cfg.AssetsRoot != "" {
		w, wErr := assets.NewWatcher(cfg.AssetsRoot, cache, logger)
		if wErr != nil {
			logger.Printf("[assets] warning: could not start watcher: %v", wErr)
		} else if w != nil {
			ctx.watcher = w
			defer func() { _ = w.Close() }()

			// Register texture reupload handler.
			if platform != nil {
				w.SetOnReload(func(relPath string) {
					lower := strings.ToLower(relPath)
					if !strings.HasSuffix(lower, ".png") && !strings.HasSuffix(lower, ".jpg") && !strings.HasSuffix(lower, ".jpeg") {
						return
					}
					absPath := filepath.Join(cache.AssetsRoot, filepath.FromSlash(relPath))
					img, decodeErr := decodeImage(absPath)
					if decodeErr != nil {
						logger.Printf("[assets] hot-reload decode error %s: %v", relPath, decodeErr)
						return
					}
					if reupErr := platform.ReuploadTexture(absPath, img); reupErr != nil {
						logger.Printf("[assets] hot-reload reupload error %s: %v", relPath, reupErr)
						return
					}
					logger.Printf("[assets] hot-reloaded %s", relPath)
				})
			}
		}
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
		if platform != nil {
			platform.UpdateInput(ctx.Input)
			if platform.PumpEvents() {
				ctx.Quit()
			}
		}

		if ctx.ShouldQuit() {
			return nil
		}

		if cfg.MaxFrames > 0 && ctx.Frames >= cfg.MaxFrames {
			return nil
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

		if platform != nil {
			if err := platform.DrawFrame(ctx.Config.ClearColor, ctx.drawCmds, ctx.overlayLines()); err != nil {
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
	if cfg.CacheMaxSize == 0 {
		cfg.CacheMaxSize = 256
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

// decodeImage opens and decodes a PNG or JPEG file.
func decodeImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".png") {
		return png.Decode(f)
	}
	// JPEG and other formats handled by image.Decode (jpeg registered via blank import).
	img, _, err := image.Decode(f)
	return img, err
}
