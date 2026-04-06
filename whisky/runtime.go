package whisky

import (
	"errors"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

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

	platform   platformapi.Platform
	renderer   platformapi.Renderer
	debugLines []string
	drawCmds   []render.DrawCmd
	texSeq     render.TextureID
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

func (c *Context) LoadTexture(path string) (render.TextureID, int, int, error) {
	if c.renderer == nil {
		c.texSeq++
		return c.texSeq, 0, 0, nil
	}
	return c.renderer.LoadTexture(path)
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
