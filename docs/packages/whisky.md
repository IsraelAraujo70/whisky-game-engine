# Package: `whisky`

The `whisky` package is the runtime entrypoint for games. It owns the high-level configuration contract and the execution loop that calls game lifecycle hooks.

## Files

| File | Purpose |
|------|---------|
| `runtime.go` | `Config`, `Context`, `Game`, and `Run()` |
| `internal/platform/sdl3/runtime.go` | SDL3 window bootstrap, event pump, input polling, and 2D rendering |

## Responsibilities

- normalize runtime configuration
- lock the main goroutine to the OS thread
- create a `Context` with Camera2D, Input, and Scene
- create a native SDL3 window with virtual resolution scaling
- poll keyboard state and feed it into the input system each frame
- execute `Load`, `Update`, and `Shutdown` in a predictable order
- advance the active scene each frame
- collect draw commands (`DrawRect`) and present them via SDL3
- provide a simple shutdown mechanism through `Context.Quit()`

## Current Loop

```text
Run()
  -> defaults
  -> Context (with Camera2D at virtual center)
  -> SDL3 window + renderer + SetLogicalPresentation
  -> Load()
  -> repeat until quit or max frames:
       UpdateInput (keyboard state -> input.State)
       Pump native events
       Scene.Update(dt)
       Game.Update(dt)     // game calls ctx.DrawRect() here
       DrawFrame (clear + filled rects + debug text)
       Reset draw queue
  -> Shutdown()
```

## Virtual Resolution

SDL3's `SetLogicalPresentation` handles the mapping from virtual coordinates (e.g. 320x180) to window pixels (e.g. 1280x720). When `Config.PixelPerfect` is true, integer scaling is used; otherwise letterboxing is applied. All draw calls and debug text operate in virtual coordinate space.

## Input Pipeline

The SDL3 platform reads the keyboard state every frame via `sdl.GetKeyboardState()` and maps scancodes to engine control names. Games bind actions to control names via `ctx.Input.Bind("move_left", "a", "left")`.

The scancode-to-control mapping is defined by `Config.KeyMap` (`whisky.KeyMap`, a `map[string]string`). Keys are human-readable names (e.g. `"w"`, `"space"`, `"f1"`); values are the control names fed into the input system. If `KeyMap` is nil, a built-in default set is used (w/a/s/d, arrows, space, lshift, enter).

```go
whisky.Run(&game{}, whisky.Config{
    KeyMap: whisky.KeyMap{
        "space": "jump",
        "z":     "attack",
        "lctrl": "crouch",
    },
})
```

Supported key names: letters (`"a"`–`"z"`), digits (`"0"`–`"9"`), arrow keys (`"up"`, `"down"`, `"left"`, `"right"`), named keys (`"space"`, `"enter"`, `"escape"`, `"backspace"`, `"tab"`, `"lshift"`, `"rshift"`, `"lctrl"`, `"rctrl"`, `"lalt"`, `"ralt"`), and function keys (`"f1"`–`"f12"`). Unknown key names are silently ignored.

## Rendering

Games queue filled rectangles via `ctx.DrawRect(worldRect, color)`. The Camera2D automatically transforms world coordinates to screen (virtual) coordinates. The SDL3 platform draws rectangles via `RenderFillRect` and renders the debug text overlay on top.
