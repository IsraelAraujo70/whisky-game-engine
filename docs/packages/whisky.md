# Package: `whisky`

The `whisky` package is the runtime entrypoint for games. It owns the high-level configuration contract and the execution loop that calls game lifecycle hooks.

## Files

| File | Purpose |
|------|---------|
| `runtime.go` | `Config`, `Context`, `Game`, and `Run()` |
| `internal/backend/desktop_*.go` | OS-selected desktop backend factories used by the runtime |
| `internal/nativewindow/desktop_*.go` | OS-selected native window factories for future Vulkan/D3D12/Metal integration |
| `internal/gfx/rhi/*.go` | graphics abstraction contracts that bind future render backends to native window handles |
| `internal/gfx/vulkan/*.go` | Vulkan backend entrypoint scaffold that will implement the RHI |
| `internal/platform/platform.go` | backend contracts for native platform and renderer integration |
| `internal/platform/sdl3/runtime.go` | transitional SDL3 backend implementing both platform and renderer responsibilities |

## Responsibilities

- normalize runtime configuration
- lock the main goroutine to the OS thread
- create a `Context` with Camera2D, Input, and Scene
- bind the runtime to an internal backend contract rather than a concrete native implementation
- create a native window with virtual resolution scaling
- poll native input state and feed it into the input system each frame
- execute `Load`, `Update`, and `Shutdown` in a predictable order
- advance the active scene each frame
- collect draw commands (`DrawRect`) and present them via the active renderer backend
- provide a simple shutdown mechanism through `Context.Quit()`

## Current Loop

```text
Run()
  -> defaults
  -> Context (with Camera2D at virtual center)
  -> create backend
  -> backend.SetLogicalSize(...)
  -> Load()
  -> repeat until quit or max frames:
       backend.UpdateInput(...)
       Pump native events
       Scene.Update(dt)
       Game.Update(dt)     // game calls ctx.DrawRect() here
       backend.DrawFrame(...)
       Reset draw queue
  -> Shutdown()
```

At the moment the only backend is SDL3, but `whisky.Run` no longer depends on the concrete `sdl3.Runtime` type. This is the first step toward native platform backends plus Vulkan and D3D12 render paths.

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
