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

	debugLines []string
	drawCmds   []render.FillRect
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

// DrawRect queues a filled rectangle in world coordinates. The camera
// transform is applied automatically so callers work in world space.
// If Camera is nil the rectangle is drawn as-is (screen space).
func (c *Context) DrawRect(worldRect geom.Rect, color geom.Color) {
	r := worldRect
	if c.Camera != nil {
		vw := float64(c.Config.VirtualWidth)
		vh := float64(c.Config.VirtualHeight)
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

	var platform *sdl3.Runtime
	if !cfg.Headless && os.Getenv("WHISKY_HEADLESS") != "1" {
		platform, err = sdl3.New(cfg.Title, cfg.WindowWidth, cfg.WindowHeight)
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
